package ledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS ledger_records (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts INTEGER NOT NULL,
	type TEXT NOT NULL,
	source TEXT NOT NULL,
	payload TEXT NOT NULL,
	hash TEXT NOT NULL,
	prev_hash TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ledger_records_ts ON ledger_records(ts);
`

type Ledger struct {
	db *sql.DB
	wal *WAL
	walSeq int64
	walMu sync.Mutex
}

// OpenOptions configures how a Ledger is opened.
type OpenOptions struct {
	// WALPath enables a write-ahead log at the given path when non-empty.
	WALPath string
	// WALFsync forces an fsync after every WAL write when true.
	WALFsync bool
}

type Record struct {
	ID        int64  `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Payload   string `json:"payload"`
	Hash      string `json:"hash"`
	PrevHash  string `json:"prev_hash"`
}

type RecordInput struct {
	Timestamp int64
	Type      string
	Source    string
	Payload   string
}

type ListQuery struct {
	Since int64
	Until int64
	Limit int
}

type VerifyResult struct {
	OK        bool   `json:"ok"`
	FailedID  int64  `json:"failed_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Checked   int64  `json:"checked"`
	Timestamp int64  `json:"timestamp"`
}

type ProofResult struct {
	OK        bool   `json:"ok"`
	FailedID  int64  `json:"failed_id,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Checked   int64  `json:"checked"`
	LastID    int64  `json:"last_id,omitempty"`
	LastHash  string `json:"last_hash,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func Open(path string) (*Ledger, error) {
	return OpenWithOptions(path, OpenOptions{})
}

// OpenWithOptions opens a ledger, optionally attaching a write-ahead log. When
// a WAL is configured, any records left pending from a previous crash are
// automatically recovered (replayed) before the ledger is returned.
func OpenWithOptions(path string, opts OpenOptions) (*Ledger, error) {
	if path == "" {
		return nil, errors.New("db path required")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Configure connection pool for scalability
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	l := &Ledger{db: db}

	if opts.WALPath != "" {
		wal, err := NewWAL(opts.WALPath, opts.WALFsync)
		if err != nil {
			_ = db.Close()
			return nil, err
		}
		l.wal = wal
		// Recover any records that were logged but not committed before a crash.
		if err := l.RecoverWAL(); err != nil {
			_ = wal.Close()
			_ = db.Close()
			return nil, err
		}
	}

	return l, nil
}

// RecoverWAL replays any WAL entries that were not durably committed to the
// database. It is called automatically by OpenWithOptions when a WAL is
// configured. It is safe to call manually after re-attaching a WAL.
func (l *Ledger) RecoverWAL() error {
	if l == nil || l.wal == nil {
		return nil
	}
	pending, err := l.wal.Recover()
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	// Replay in a single batch for efficiency.
	if _, err := l.AppendBatch(pending); err != nil {
		return err
	}
	return nil
}

func (l *Ledger) Close() error {
	if l == nil {
		return nil
	}
	if l.wal != nil {
		_ = l.wal.Close()
	}
	if l.db == nil {
		return nil
	}
	return l.db.Close()
}

func (l *Ledger) InitSchema() error {
	_, err := l.db.Exec(schema)
	return err
}

func (l *Ledger) Append(input RecordInput) (Record, error) {
	if strings.TrimSpace(input.Type) == "" {
		return Record{}, errors.New("type required")
	}
	if strings.TrimSpace(input.Payload) == "" {
		return Record{}, errors.New("payload required")
	}

	// Write-ahead: persist the intent before mutating the database so a crash
	// between this point and the DB commit can be recovered.
	seq := l.nextWALSeq()
	if err := l.walLog(seq, input); err != nil {
		return Record{}, err
	}

	prevHash, err := l.lastHash()
	if err != nil {
		return Record{}, err
	}

	hash := computeHash(prevHash, input.Timestamp, input.Type, input.Source, input.Payload)

	res, err := l.db.Exec(
		`INSERT INTO ledger_records(ts, type, source, payload, hash, prev_hash) VALUES(?, ?, ?, ?, ?, ?)`,
		input.Timestamp, input.Type, input.Source, input.Payload, hash, prevHash,
	)
	if err != nil {
		return Record{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Record{}, err
	}

	rec := Record{
		ID:        id,
		Timestamp: input.Timestamp,
		Type:      input.Type,
		Source:    input.Source,
		Payload:   input.Payload,
		Hash:      hash,
		PrevHash:  prevHash,
	}

	// Mark the WAL entry committed now that the DB row is durable.
	_ = l.walMarkCommitted(seq)

	return rec, nil
}

// nextWALSeq returns a process-unique, monotonically increasing sequence number
// for WAL entries.
func (l *Ledger) nextWALSeq() int64 {
	if l.wal == nil {
		return 0
	}
	l.walMu.Lock()
	defer l.walMu.Unlock()
	l.walSeq++
	return l.walSeq
}

// walLog writes an entry to the WAL (no-op when WAL is disabled).
func (l *Ledger) walLog(seq int64, input RecordInput) error {
	if l.wal == nil {
		return nil
	}
	return l.wal.Log(seq, input)
}

// walMarkCommitted marks a WAL entry as committed (no-op when WAL disabled).
func (l *Ledger) walMarkCommitted(seq int64) error {
	if l.wal == nil {
		return nil
	}
	return l.wal.MarkCommitted(seq)
}

// AppendBatch appends multiple records in a single transaction for better performance
func (l *Ledger) AppendBatch(inputs []RecordInput) ([]Record, error) {
	if len(inputs) == 0 {
		return nil, errors.New("no inputs provided")
	}

	tx, err := l.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	prevHash, err := l.lastHashTx(tx)
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(inputs))
	stmt, err := tx.Prepare(`INSERT INTO ledger_records(ts, type, source, payload, hash, prev_hash) VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	for _, input := range inputs {
		if strings.TrimSpace(input.Type) == "" {
			return nil, errors.New("type required")
		}
		if strings.TrimSpace(input.Payload) == "" {
			return nil, errors.New("payload required")
		}

		hash := computeHash(prevHash, input.Timestamp, input.Type, input.Source, input.Payload)

		res, err := stmt.Exec(input.Timestamp, input.Type, input.Source, input.Payload, hash, prevHash)
		if err != nil {
			return nil, err
		}

		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}

		records = append(records, Record{
			ID:        id,
			Timestamp: input.Timestamp,
			Type:      input.Type,
			Source:    input.Source,
			Payload:   input.Payload,
			Hash:      hash,
			PrevHash:  prevHash,
		})

		prevHash = hash
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return records, nil
}

func (l *Ledger) GetByID(id int64) (Record, error) {
	row := l.db.QueryRow(`SELECT id, ts, type, source, payload, hash, prev_hash FROM ledger_records WHERE id = ?`, id)
	return scanRecord(row)
}

func (l *Ledger) List(q ListQuery) ([]Record, error) {
	if q.Limit <= 0 {
		q.Limit = 100
	}

	query := `SELECT id, ts, type, source, payload, hash, prev_hash FROM ledger_records`
	args := []any{}
	clauses := []string{}

	if q.Since > 0 {
		clauses = append(clauses, "ts >= ?")
		args = append(args, q.Since)
	}
	if q.Until > 0 {
		clauses = append(clauses, "ts <= ?")
		args = append(args, q.Until)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY id ASC LIMIT ?"
	args = append(args, q.Limit)

	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Timestamp, &rec.Type, &rec.Source, &rec.Payload, &rec.Hash, &rec.PrevHash); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}

	return out, rows.Err()
}

func (l *Ledger) VerifyChain() (VerifyResult, error) {
	rows, err := l.db.Query(`SELECT id, ts, type, source, payload, hash, prev_hash FROM ledger_records ORDER BY id ASC`)
	if err != nil {
		return VerifyResult{}, err
	}
	defer rows.Close()

	var prev string
	var checked int64
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Timestamp, &rec.Type, &rec.Source, &rec.Payload, &rec.Hash, &rec.PrevHash); err != nil {
			return VerifyResult{}, err
		}

		if rec.PrevHash != prev {
			return VerifyResult{
				OK:        false,
				FailedID:  rec.ID,
				Reason:    "prev_hash mismatch",
				Checked:   checked,
				Timestamp: time.Now().Unix(),
			}, nil
		}

		expected := computeHash(prev, rec.Timestamp, rec.Type, rec.Source, rec.Payload)
		if rec.Hash != expected {
			return VerifyResult{
				OK:        false,
				FailedID:  rec.ID,
				Reason:    "hash mismatch",
				Checked:   checked,
				Timestamp: time.Now().Unix(),
			}, nil
		}

		prev = rec.Hash
		checked++
	}

	if err := rows.Err(); err != nil {
		return VerifyResult{}, err
	}

	return VerifyResult{
		OK:        true,
		Checked:   checked,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (l *Ledger) VerifyUpTo(targetTime int64) (ProofResult, error) {
	rows, err := l.db.Query(`SELECT id, ts, type, source, payload, hash, prev_hash FROM ledger_records WHERE ts <= ? ORDER BY id ASC`, targetTime)
	if err != nil {
		return ProofResult{}, err
	}
	defer rows.Close()

	var prev string
	var checked int64
	var lastID int64
	var lastHash string
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Timestamp, &rec.Type, &rec.Source, &rec.Payload, &rec.Hash, &rec.PrevHash); err != nil {
			return ProofResult{}, err
		}

		if rec.PrevHash != prev {
			return ProofResult{
				OK:        false,
				FailedID:  rec.ID,
				Reason:    "prev_hash mismatch",
				Checked:   checked,
				Timestamp: time.Now().Unix(),
			}, nil
		}

		expected := computeHash(prev, rec.Timestamp, rec.Type, rec.Source, rec.Payload)
		if rec.Hash != expected {
			return ProofResult{
				OK:        false,
				FailedID:  rec.ID,
				Reason:    "hash mismatch",
				Checked:   checked,
				Timestamp: time.Now().Unix(),
			}, nil
		}

		prev = rec.Hash
		lastID = rec.ID
		lastHash = rec.Hash
		checked++
	}

	if err := rows.Err(); err != nil {
		return ProofResult{}, err
	}

	return ProofResult{
		OK:        true,
		Checked:   checked,
		LastID:    lastID,
		LastHash:  lastHash,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (l *Ledger) lastHash() (string, error) {
	row := l.db.QueryRow(`SELECT hash FROM ledger_records ORDER BY id DESC LIMIT 1`)
	var hash string
	if err := row.Scan(&hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return hash, nil
}

func (l *Ledger) lastHashTx(tx *sql.Tx) (string, error) {
	row := tx.QueryRow(`SELECT hash FROM ledger_records ORDER BY id DESC LIMIT 1`)
	var hash string
	if err := row.Scan(&hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return hash, nil
}

func computeHash(prevHash string, ts int64, rtype, source, payload string) string {
	value := fmt.Sprintf("%s|%d|%s|%s|%s", prevHash, ts, rtype, source, payload)
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func scanRecord(row *sql.Row) (Record, error) {
	var rec Record
	if err := row.Scan(&rec.ID, &rec.Timestamp, &rec.Type, &rec.Source, &rec.Payload, &rec.Hash, &rec.PrevHash); err != nil {
		return Record{}, err
	}
	return rec, nil
}

// MerkleRoot computes the current Merkle root over all record hashes in id
// order. This provides a compact, tamper-evident summary of the entire ledger
// that can be signed and published for external verification.
func (l *Ledger) MerkleRoot() (string, error) {
	records, err := l.List(ListQuery{Since: 0, Until: 1 << 62, Limit: 1 << 62})
	if err != nil {
		return "", err
	}
	tree := BuildMerkleTreeFromRecords(records)
	return tree.Root(), nil
}

// MerkleProofFor returns an inclusion proof for the record with the given id.
func (l *Ledger) MerkleProofFor(id int64) (MerkleProof, error) {
	records, err := l.List(ListQuery{Since: 0, Until: 1 << 62, Limit: 1 << 62})
	if err != nil {
		return MerkleProof{}, err
	}
	sorted := make([]Record, len(records))
	copy(sorted, records)
	sortSliceByID(sorted)

	idx := -1
	for i, r := range sorted {
		if r.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return MerkleProof{}, errors.New("record not found")
	}

	tree := BuildMerkleTreeFromRecords(sorted)
	return tree.GenerateProof(idx)
}

func sortSliceByID(recs []Record) {
	for i := 1; i < len(recs); i++ {
		for j := i; j > 0 && recs[j-1].ID > recs[j].ID; j-- {
			recs[j-1], recs[j] = recs[j], recs[j-1]
		}
	}
}
