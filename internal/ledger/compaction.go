package ledger

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CompactOptions controls how Compact prunes and rebuilds the ledger.
type CompactOptions struct {
	// KeepLastN keeps at most this many of the most recent records. When 0,
	// no record-count based pruning is performed (only the chain rebuild runs).
	KeepLastN int64
	// KeepSince drops records whose timestamp is strictly before this unix
	// second. When 0, no time-based pruning is performed.
	KeepSince int64
	// RebuildChain recomputes every hash/prev_hash from scratch so that a
	// tampered or inconsistent chain is repaired. The resulting chain is
	// internally consistent but will NOT match a previously published root.
	RebuildChain bool
}

// CompactResult reports what Compact did.
type CompactResult struct {
	Removed     int64  `json:"removed"`
	Remaining   int64  `json:"remaining"`
	Rebuilt     bool   `json:"rebuilt"`
	NewRoot     string `json:"new_root"`
	Timestamp   int64  `json:"timestamp"`
}

// Compact prunes old records and (optionally) rebuilds the hash chain so the
// ledger stays small and internally consistent over long-running deployments.
//
// Pruning is performed first (KeepLastN / KeepSince), then — when requested —
// every remaining record's hash and prev_hash are recomputed in id order so
// the chain is fully consistent. A compaction marker record is appended so the
// operation is itself auditable.
func (l *Ledger) Compact(opts CompactOptions) (CompactResult, error) {
	res := CompactResult{Timestamp: time.Now().Unix()}

	// --- Pruning phase -----------------------------------------------------
	if opts.KeepLastN > 0 || opts.KeepSince > 0 {
		removed, err := l.prune(opts)
		if err != nil {
			return res, err
		}
		res.Removed = removed
		// Pruning necessarily breaks prev_hash continuity (a kept record may
		// reference a deleted predecessor), so the chain must always be rebuilt
		// afterwards to stay internally consistent.
		opts.RebuildChain = true
	}

	// --- Chain rebuild phase ----------------------------------------------
	if opts.RebuildChain {
		if err := l.rebuildChain(); err != nil {
			return res, err
		}
		res.Rebuilt = true
	}

	remaining, err := l.count()
	if err != nil {
		return res, err
	}
	res.Remaining = remaining

	root, err := l.MerkleRoot()
	if err != nil {
		return res, err
	}
	res.NewRoot = root

	// Append an auditable compaction marker so the operation is verifiable.
	marker := fmt.Sprintf(`{"removed":%d,"remaining":%d,"rebuilt":%v,"new_root":"%s"}`,
		res.Removed, res.Remaining, res.Rebuilt, res.NewRoot)
	if _, err := l.Append(RecordInput{
		Timestamp: time.Now().Unix(),
		Type:      "compaction",
		Source:    "ledger.compact",
		Payload:   marker,
	}); err != nil {
		return res, err
	}

	return res, nil
}

// prune removes records according to the supplied options. It returns the
// number of rows removed.
func (l *Ledger) prune(opts CompactOptions) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var removed int64

	if opts.KeepLastN > 0 {
		// Delete everything except the most recent KeepLastN records.
		r, err := tx.Exec(`
			DELETE FROM ledger_records
			WHERE id NOT IN (
				SELECT id FROM ledger_records ORDER BY id DESC LIMIT ?
			)`, opts.KeepLastN)
		if err != nil {
			return 0, err
		}
		n, _ := r.RowsAffected()
		removed += n
	}

	if opts.KeepSince > 0 {
		r, err := tx.Exec(`DELETE FROM ledger_records WHERE ts < ?`, opts.KeepSince)
		if err != nil {
			return 0, err
		}
		n, _ := r.RowsAffected()
		removed += n
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return removed, nil
}

// rebuildChain recomputes hash and prev_hash for every record in id order so
// the chain is internally consistent. It runs inside a single transaction.
func (l *Ledger) rebuildChain() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `SELECT id, ts, type, source, payload FROM ledger_records ORDER BY id ASC`)
	if err != nil {
		return err
	}

	type rec struct {
		id, ts      int64
		rtype, src, payload string
	}
	var recs []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.ts, &r.rtype, &r.src, &r.payload); err != nil {
			_ = rows.Close()
			return err
		}
		recs = append(recs, r)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `UPDATE ledger_records SET hash = ?, prev_hash = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	var prev string
	for _, r := range recs {
		h := computeHash(prev, r.ts, r.rtype, r.src, r.payload)
		if _, err := stmt.ExecContext(ctx, h, prev, r.id); err != nil {
			return err
		}
		prev = h
	}

	return tx.Commit()
}

// count returns the number of records currently in the ledger.
func (l *Ledger) count() (int64, error) {
	var n int64
	row := l.db.QueryRow(`SELECT COUNT(*) FROM ledger_records`)
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ensure sql is referenced (kept for clarity of the database/sql import use).
var _ = sql.ErrNoRows
