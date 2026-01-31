package ledger

import (
	"encoding/json"
	"errors"
	"time"
)

type AuditBundle struct {
	GeneratedAt int64                `json:"generated_at"`
	TargetTime  int64                `json:"target_time"`
	Snapshot    ReconstructionReport `json:"snapshot"`
	Proof       *ProofResult         `json:"proof,omitempty"`
	Notes       []string             `json:"notes,omitempty"`
}

func (r *Reconstructor) ExportAuditBundle(targetTime int64) (AuditBundle, error) {
	if targetTime <= 0 {
		return AuditBundle{}, errors.New("target_time must be > 0")
	}

	report := r.ReconstructAtTime(targetTime)

	bundle := AuditBundle{
		GeneratedAt: time.Now().Unix(),
		TargetTime:  targetTime,
		Snapshot:    report,
		Proof:       report.Proof,
		Notes:       []string{},
	}

	if report.Proof == nil {
		bundle.Notes = append(bundle.Notes, "no proof available")
	}
	if !report.Success {
		bundle.Notes = append(bundle.Notes, "snapshot reconstruction failed")
	}
	if !report.Coverage.Complete {
		bundle.Notes = append(bundle.Notes, "snapshot missing required dimensions")
	}

	return bundle, nil
}

func (b AuditBundle) ToJSON() (string, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
