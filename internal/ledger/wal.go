package ledger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// WAL provides a write-ahead log for ledger appends. Every append is first
// written (and optionally fsync'd) to the WAL before being committed to the
// SQLite database. On startup the WAL can be replayed to recover records that
// were acknowledged but not yet durably committed to the main store (e.g. after
// a crash between the WAL write and the DB commit).
type WAL struct {
	mu       sync.Mutex
	path     string
	file     *os.File
	writer   *bufio.Writer
	fsync    bool
	disabled bool
}

// WALEntry is a single serialized append operation.
type WALEntry struct {
	Seq       int64       `json:"seq"`
	Input     RecordInput `json:"input"`
	Committed bool        `json:"committed"`
}

// NewWAL opens (or creates) a WAL file at the given path. When fsync is true
// every entry is flushed to disk before returning.
func NewWAL(path string, fsync bool) (*WAL, error) {
	if path == "" {
		return &WAL{disabled: true}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &WAL{
		path:   path,
		file:   f,
		writer: bufio.NewWriter(f),
		fsync:  fsync,
	}, nil
}

// Close flushes and closes the WAL file.
func (w *WAL) Close() error {
	if w == nil || w.disabled || w.file == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Close()
}

// Log writes an append entry to the WAL. It returns the sequence number used.
func (w *WAL) Log(seq int64, input RecordInput) error {
	if w == nil || w.disabled {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	entry := WALEntry{Seq: seq, Input: input, Committed: false}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := w.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := w.writer.Flush(); err != nil {
		return err
	}
	if w.fsync {
		return w.file.Sync()
	}
	return nil
}

// MarkCommitted records that the entry with the given seq was durably
// committed to the database. It is written as a small tombstone line so the
// recovery process can skip already-committed entries.
func (w *WAL) MarkCommitted(seq int64) error {
	if w == nil || w.disabled {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	entry := WALEntry{Seq: seq, Committed: true}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := w.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	if err := w.writer.Flush(); err != nil {
		return err
	}
	if w.fsync {
		return w.file.Sync()
	}
	return nil
}

// Recover reads the WAL and returns entries that were logged but never marked
// committed. These should be replayed into the ledger. The WAL is then
// truncated so a subsequent crash does not re-replay them.
func (w *WAL) Recover() ([]RecordInput, error) {
	if w == nil || w.disabled {
		return nil, nil
	}

	raw, err := os.ReadFile(w.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	pending := make(map[int64]RecordInput)
	committed := make(map[int64]bool)

	// Parse line by line.
	var line []byte
	for _, b := range raw {
		if b == '\n' {
			if len(line) > 0 {
				var e WALEntry
				if err := json.Unmarshal(line, &e); err == nil {
					if e.Committed {
						committed[e.Seq] = true
						delete(pending, e.Seq)
					} else {
						pending[e.Seq] = e.Input
					}
				}
			}
			line = line[:0]
			continue
		}
		line = append(line, b)
	}
	// Handle a trailing line without newline.
	if len(line) > 0 {
		var e WALEntry
		if err := json.Unmarshal(line, &e); err == nil {
			if e.Committed {
				committed[e.Seq] = true
				delete(pending, e.Seq)
			} else {
				pending[e.Seq] = e.Input
			}
		}
	}

	// Drop anything already committed.
	for seq := range committed {
		delete(pending, seq)
	}

	// Return in seq order for deterministic replay.
	out := make([]RecordInput, 0, len(pending))
	for _, input := range pending {
		out = append(out, input)
	}
	// Simple insertion sort by timestamp then type for determinism.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && (out[j-1].Timestamp > out[j].Timestamp ||
			(out[j-1].Timestamp == out[j].Timestamp && out[j-1].Type > out[j].Type)); j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}

	// Truncate the WAL now that we have captured the pending entries.
	if err := os.WriteFile(w.path, []byte{}, 0o644); err != nil {
		return nil, fmt.Errorf("wal truncate: %w", err)
	}

	return out, nil
}
