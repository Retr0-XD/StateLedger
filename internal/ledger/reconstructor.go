package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/collectors"
)

type SnapshotState struct {
	Timestamp       int64                          `json:"timestamp"`
	Code            *collectors.CodePayload        `json:"code,omitempty"`
	Config          *collectors.ConfigPayload      `json:"config,omitempty"`
	Environment     *collectors.EnvironmentPayload `json:"environment,omitempty"`
	Mutations       []collectors.MutationPayload   `json:"mutations,omitempty"`
	MutationRecords []MutationRecord               `json:"mutation_records,omitempty"`
}

type MutationRecord struct {
	LedgerID    int64  `json:"ledger_id"`
	Timestamp   int64  `json:"timestamp"`
	Type        string `json:"type"`
	ID          string `json:"id"`
	Source      string `json:"source"`
	Hash        string `json:"hash"`
	ExternalRef string `json:"external_ref,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Offset      int64  `json:"offset,omitempty"`
}

type ReconstructionReport struct {
	RequestTime      int64          `json:"request_time"`
	TargetTime       int64          `json:"target_time"`
	Success          bool           `json:"success"`
	RecordsMatched   int            `json:"records_matched"`
	Coverage         CoverageReport `json:"coverage"`
	DeterminismScore float64        `json:"determinism_score"`
	Issues           []string       `json:"issues,omitempty"`
	Proof            *ProofResult   `json:"proof,omitempty"`
	ReplayPlan       *ReplayPlan    `json:"replay_plan,omitempty"`
	State            *SnapshotState `json:"state,omitempty"`
}

type CoverageReport struct {
	HasCode        bool `json:"has_code"`
	HasConfig      bool `json:"has_config"`
	HasEnvironment bool `json:"has_environment"`
	HasMutations   bool `json:"has_mutations"`
	Complete       bool `json:"complete"`
}

type ReplayPlan struct {
	Namespaces []NamespacePlan `json:"namespaces"`
	Total      int             `json:"total"`
}

type NamespacePlan struct {
	Namespace string           `json:"namespace"`
	Count     int              `json:"count"`
	Ordered   bool             `json:"ordered"`
	Records   []MutationRecord `json:"records"`
}

type Reconstructor struct {
	l *Ledger
}

func New(l *Ledger) *Reconstructor {
	return &Reconstructor{l: l}
}

func (r *Reconstructor) ReconstructAtTime(targetTime int64) ReconstructionReport {
	report := ReconstructionReport{
		RequestTime: time.Now().Unix(),
		TargetTime:  targetTime,
		Issues:      []string{},
	}

	recs, err := r.l.List(ListQuery{
		Since: 0,
		Until: targetTime,
		Limit: 10000,
	})
	if err != nil {
		report.Issues = append(report.Issues, err.Error())
		return report
	}

	if proof, err := r.l.VerifyUpTo(targetTime); err == nil {
		report.Proof = &proof
	} else {
		report.Issues = append(report.Issues, "proof: "+err.Error())
	}

	report.RecordsMatched = len(recs)

	state := &SnapshotState{
		Timestamp:       targetTime,
		Mutations:       []collectors.MutationPayload{},
		MutationRecords: []MutationRecord{},
	}

	coverage := CoverageReport{}

	for _, rec := range recs {
		if rec.Timestamp > targetTime {
			continue
		}

		switch rec.Type {
		case "code":
			var cp collectors.CodePayload
			if err := collectors.ParseJSON(rec.Payload, &cp); err != nil {
				report.Issues = append(report.Issues, "code parse error: "+err.Error())
				continue
			}
			state.Code = &cp
			coverage.HasCode = true

		case "config":
			var cp collectors.ConfigPayload
			if err := collectors.ParseJSON(rec.Payload, &cp); err != nil {
				report.Issues = append(report.Issues, "config parse error: "+err.Error())
				continue
			}
			state.Config = &cp
			coverage.HasConfig = true

		case "environment":
			var ep collectors.EnvironmentPayload
			if err := collectors.ParseJSON(rec.Payload, &ep); err != nil {
				report.Issues = append(report.Issues, "environment parse error: "+err.Error())
				continue
			}
			state.Environment = &ep
			coverage.HasEnvironment = true

		case "mutation":
			var mp collectors.MutationPayload
			if err := collectors.ParseJSON(rec.Payload, &mp); err != nil {
				report.Issues = append(report.Issues, "mutation parse error: "+err.Error())
				continue
			}
			namespace, offset, _ := parseExternalRef(mp.ExternalRef)
			state.Mutations = append(state.Mutations, mp)
			state.MutationRecords = append(state.MutationRecords, MutationRecord{
				LedgerID:    rec.ID,
				Timestamp:   rec.Timestamp,
				Type:        mp.Type,
				ID:          mp.ID,
				Source:      mp.Source,
				Hash:        mp.Hash,
				ExternalRef: mp.ExternalRef,
				Namespace:   namespace,
				Offset:      offset,
			})
			coverage.HasMutations = len(state.Mutations) > 0
		}
	}

	if len(state.MutationRecords) > 1 {
		orderMutationRecords(state.MutationRecords)
	}

	coverage.Complete = coverage.HasCode && coverage.HasConfig && coverage.HasEnvironment && coverage.HasMutations

	report.Coverage = coverage
	report.State = state
	report.Success = true

	report.DeterminismScore = r.calculateDeterminismScore(state, coverage)
	report.ReplayPlan = buildReplayPlan(state.MutationRecords)

	applyProvenanceChecks(state, &report)

	if !coverage.HasCode {
		report.Issues = append(report.Issues, "warning: no code snapshot")
	}
	if !coverage.HasConfig {
		report.Issues = append(report.Issues, "warning: no config snapshot")
	}
	if !coverage.HasEnvironment {
		report.Issues = append(report.Issues, "warning: no environment snapshot")
	}
	if !coverage.HasMutations {
		report.Issues = append(report.Issues, "warning: no mutations recorded")
	}

	return report
}

func (r *Reconstructor) calculateDeterminismScore(state *SnapshotState, coverage CoverageReport) float64 {
	score := 0.0

	if coverage.HasCode {
		score += 25.0
	}
	if coverage.HasConfig {
		score += 25.0
	}
	if coverage.HasEnvironment {
		score += 25.0
	}
	if coverage.HasMutations {
		score += 25.0
	}

	if state.Environment != nil {
		if state.Environment.TimeSource != "system" {
			score -= 5.0
		}
	}

	if len(state.Mutations) > 0 {
		for _, m := range state.Mutations {
			if strings.TrimSpace(m.ExternalRef) == "" {
				score -= 2.0
				break
			}
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func applyProvenanceChecks(state *SnapshotState, report *ReconstructionReport) {
	if report == nil || state == nil {
		return
	}

	if state.Code != nil {
		if len(strings.TrimSpace(state.Code.Commit)) < 7 {
			report.Issues = append(report.Issues, "provenance: code commit hash too short")
		}
	}

	if state.Config != nil {
		snapshot := strings.TrimSpace(state.Config.Snapshot)
		if snapshot == "" {
			report.Issues = append(report.Issues, "provenance: config snapshot empty")
		} else {
			hash := computeConfigHash(snapshot)
			if strings.TrimSpace(state.Config.Hash) != "" && state.Config.Hash != hash {
				report.Issues = append(report.Issues, "provenance: config hash mismatch")
			}
		}
	}

	if state.Environment != nil {
		if strings.TrimSpace(state.Environment.OS) == "" || strings.TrimSpace(state.Environment.Runtime) == "" {
			report.Issues = append(report.Issues, "provenance: environment fields missing")
		}
	}

	if len(state.MutationRecords) > 0 {
		seen := map[string]bool{}
		seenExternal := map[string]bool{}
		namespaces := map[string]bool{}
		for _, m := range state.MutationRecords {
			if strings.TrimSpace(m.ID) != "" {
				if seen[m.ID] {
					report.Issues = append(report.Issues, "provenance: duplicate mutation id "+m.ID)
					break
				}
				seen[m.ID] = true
			}
			if strings.TrimSpace(m.ExternalRef) == "" {
				report.Issues = append(report.Issues, "provenance: mutation missing external_ref")
				break
			}
			if seenExternal[m.ExternalRef] {
				report.Issues = append(report.Issues, "provenance: duplicate external_ref "+m.ExternalRef)
				break
			}
			seenExternal[m.ExternalRef] = true
			if strings.TrimSpace(m.Namespace) != "" {
				namespaces[m.Namespace] = true
			}
		}
		if len(namespaces) > 1 {
			report.Issues = append(report.Issues, "provenance: mixed external_ref namespaces detected")
		}
	}
}

func computeConfigHash(snapshot string) string {
	sum := sha256.Sum256([]byte(snapshot))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func orderMutationRecords(records []MutationRecord) {
	allNumeric := true
	sameNamespace := true
	var namespace string
	parsed := make([]int64, len(records))
	for i, rec := range records {
		refNamespace, value, ok := parseExternalRef(rec.ExternalRef)
		if !ok {
			allNumeric = false
			break
		}
		parsed[i] = value
		if namespace == "" {
			namespace = refNamespace
		} else if namespace != refNamespace {
			sameNamespace = false
		}
	}

	if allNumeric && sameNamespace {
		sort.Slice(records, func(i, j int) bool {
			if parsed[i] == parsed[j] {
				return records[i].LedgerID < records[j].LedgerID
			}
			return parsed[i] < parsed[j]
		})
		return
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Timestamp == records[j].Timestamp {
			return records[i].LedgerID < records[j].LedgerID
		}
		return records[i].Timestamp < records[j].Timestamp
	})
}

func parseExternalRef(value string) (string, int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, false
	}
	namespace := ""
	if idx := strings.LastIndex(value, ":"); idx != -1 {
		namespace = value[:idx]
		value = value[idx+1:]
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return namespace, 0, false
	}
	return namespace, parsed, true
}

func buildReplayPlan(records []MutationRecord) *ReplayPlan {
	if len(records) == 0 {
		return nil
	}

	byNamespace := map[string][]MutationRecord{}
	order := []string{}

	for _, rec := range records {
		ns := rec.Namespace
		if strings.TrimSpace(ns) == "" {
			ns = "default"
		}
		if _, ok := byNamespace[ns]; !ok {
			order = append(order, ns)
		}
		byNamespace[ns] = append(byNamespace[ns], rec)
	}

	plan := &ReplayPlan{Namespaces: []NamespacePlan{}, Total: len(records)}
	for _, ns := range order {
		list := byNamespace[ns]
		ordered := canOrderByOffset(list)
		if ordered {
			sort.Slice(list, func(i, j int) bool {
				if list[i].Offset == list[j].Offset {
					return list[i].LedgerID < list[j].LedgerID
				}
				return list[i].Offset < list[j].Offset
			})
		} else {
			sort.Slice(list, func(i, j int) bool {
				if list[i].Timestamp == list[j].Timestamp {
					return list[i].LedgerID < list[j].LedgerID
				}
				return list[i].Timestamp < list[j].Timestamp
			})
		}
		plan.Namespaces = append(plan.Namespaces, NamespacePlan{
			Namespace: ns,
			Count:     len(list),
			Ordered:   ordered,
			Records:   list,
		})
	}

	return plan
}

func canOrderByOffset(records []MutationRecord) bool {
	if len(records) == 0 {
		return false
	}
	for _, rec := range records {
		if strings.TrimSpace(rec.ExternalRef) == "" {
			return false
		}
		if _, _, ok := parseExternalRef(rec.ExternalRef); !ok {
			return false
		}
	}
	return true
}

func (r *Reconstructor) ExplainFailure(report ReconstructionReport) string {
	if report.Success && report.Coverage.Complete {
		return "reconstruction possible: all dimensions captured"
	}

	explanation := "Reconstruction not fully possible. Missing:\n"

	if !report.Coverage.HasCode {
		explanation += "  - Code snapshot (cannot verify binary/version)\n"
	}
	if !report.Coverage.HasConfig {
		explanation += "  - Configuration snapshot (cannot replicate settings)\n"
	}
	if !report.Coverage.HasEnvironment {
		explanation += "  - Environment snapshot (OS/runtime/arch unknown)\n"
	}
	if !report.Coverage.HasMutations {
		explanation += "  - Mutation records (data mutations untracked)\n"
	}

	if len(report.Issues) > 0 {
		explanation += "\nErrors encountered:\n"
		for _, issue := range report.Issues {
			explanation += "  - " + issue + "\n"
		}
	}

	explanation += "\nDeterminism Score: " + formatScore(report.DeterminismScore) + "%\n"

	if report.DeterminismScore < 50.0 {
		explanation += "(Low confidence: use for forensics only, not audit proof)\n"
	} else if report.DeterminismScore < 100.0 {
		explanation += "(Partial: some dimensions missing but state may be representative)\n"
	}

	return explanation
}

func formatScore(score float64) string {
	if score >= 100 {
		return "100"
	}
	if score <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.1f", score)
}

func (report ReconstructionReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
