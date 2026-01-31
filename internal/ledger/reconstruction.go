package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type Snapshot struct {
	ID                int64
	Timestamp         int64
	Records           []Record
	CodeRecord        *Record
	ConfigRecord      *Record
	EnvironmentRecord *Record
	MutationRecords   []Record
}

type ReconstructionState struct {
	Valid               bool
	Timestamp           int64
	Reason              string
	MissingComponents   []string
	NondeterminismRisks []string
	VerificationStatus  string
	SnapshotHash        string
}

func (l *Ledger) ResolveSnapshotAt(ts int64) (Snapshot, error) {
	query := `
		SELECT id, ts, type, source, payload, hash, prev_hash 
		FROM ledger_records 
		WHERE ts <= ? 
		ORDER BY id DESC
	`
	rows, err := l.db.Query(query, ts)
	if err != nil {
		return Snapshot{}, err
	}
	defer rows.Close()

	snapshot := Snapshot{
		Timestamp:       ts,
		Records:         []Record{},
		MutationRecords: []Record{},
	}

	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Timestamp, &rec.Type, &rec.Source, &rec.Payload, &rec.Hash, &rec.PrevHash); err != nil {
			return Snapshot{}, err
		}

		snapshot.Records = append(snapshot.Records, rec)

		switch rec.Type {
		case "code":
			if snapshot.CodeRecord == nil {
				snapshot.CodeRecord = &rec
			}
		case "config":
			if snapshot.ConfigRecord == nil {
				snapshot.ConfigRecord = &rec
			}
		case "environment":
			if snapshot.EnvironmentRecord == nil {
				snapshot.EnvironmentRecord = &rec
			}
		case "mutation":
			snapshot.MutationRecords = append(snapshot.MutationRecords, rec)
		}
	}

	if err := rows.Err(); err != nil {
		return Snapshot{}, err
	}

	if snapshot.ID == 0 && len(snapshot.Records) > 0 {
		snapshot.ID = snapshot.Records[0].ID
	}

	return snapshot, nil
}

func (s Snapshot) Validate() ReconstructionState {
	state := ReconstructionState{
		Valid:               true,
		Timestamp:           s.Timestamp,
		VerificationStatus:  "OK",
		MissingComponents:   []string{},
		NondeterminismRisks: []string{},
	}

	if s.CodeRecord == nil {
		state.MissingComponents = append(state.MissingComponents, "code")
		state.Valid = false
	}
	if s.ConfigRecord == nil {
		state.MissingComponents = append(state.MissingComponents, "config")
	}
	if s.EnvironmentRecord == nil {
		state.MissingComponents = append(state.MissingComponents, "environment")
		state.Valid = false
	}

	if len(s.MutationRecords) == 0 {
		state.NondeterminismRisks = append(state.NondeterminismRisks, "no mutations recorded")
	}

	if state.Valid {
		state.Reason = "Full reconstruction possible"
	} else if len(state.MissingComponents) > 0 {
		state.Reason = fmt.Sprintf("Missing: %s", strings.Join(state.MissingComponents, ", "))
	}

	state.SnapshotHash = s.ComputeHash()

	return state
}

func (s Snapshot) ComputeHash() string {
	var parts []string
	for _, rec := range s.Records {
		parts = append(parts, rec.Hash)
	}

	combined := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(sum[:])
}

func (s Snapshot) Summary() map[string]interface{} {
	return map[string]interface{}{
		"id":              s.ID,
		"timestamp":       s.Timestamp,
		"record_count":    len(s.Records),
		"has_code":        s.CodeRecord != nil,
		"has_config":      s.ConfigRecord != nil,
		"has_environment": s.EnvironmentRecord != nil,
		"mutation_count":  len(s.MutationRecords),
		"snapshot_hash":   s.ComputeHash(),
	}
}
