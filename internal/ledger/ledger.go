package ledger

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
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

func Open(path string) (*Ledger, error) {
	if path == "" {
		return nil, errors.New("db path required")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Ledger{db: db}, nil
}

func (l *Ledger) Close() error {
	if l == nil || l.db == nil {
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

	return Record{
		ID:        id,
		Timestamp: input.Timestamp,
		Type:      input.Type,
		Source:    input.Source,
		Payload:   input.Payload,
		Hash:      hash,
		PrevHash:  prevHash,
	}, nil
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
