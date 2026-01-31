package ledger

import (
	"encoding/json"
	"strings"

	"github.com/Retr0-XD/StateLedger/internal/collectors"
)

type DeterminismAnalysis struct {
	Score          float64  `json:"score"`
	Violations     []string `json:"violations,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	RiskLevel      string   `json:"risk_level"`
	Recommendation string   `json:"recommendation"`
	CanReplay      bool     `json:"can_replay"`
	ExternalDeps   []string `json:"external_deps,omitempty"`
}

func AnalyzeEnvironment(env *collectors.EnvironmentPayload) DeterminismAnalysis {
	analysis := DeterminismAnalysis{
		Score:        100.0,
		Violations:   []string{},
		Warnings:     []string{},
		ExternalDeps: []string{},
		CanReplay:    true,
	}

	if env == nil {
		analysis.Score = 0
		analysis.Violations = append(analysis.Violations, "no environment snapshot")
		analysis.CanReplay = false
		analysis.RiskLevel = "high"
		analysis.Recommendation = "Environment unknown; reconstruction will likely fail"
		return analysis
	}

	if env.TimeSource != "system" && env.TimeSource != "virtualized" && env.TimeSource != "fixed" {
		analysis.Warnings = append(analysis.Warnings, "unknown time source: "+env.TimeSource)
		analysis.Score -= 10
	}

	if strings.Contains(env.Runtime, "nondeterministic") {
		analysis.Violations = append(analysis.Violations, "runtime flagged as nondeterministic")
		analysis.CanReplay = false
		analysis.Score -= 30
	}

	if analysis.Score >= 80 {
		analysis.RiskLevel = "low"
		analysis.Recommendation = "Environment state captured with high confidence; replay advisable"
	} else if analysis.Score >= 50 {
		analysis.RiskLevel = "medium"
		analysis.Recommendation = "Environment partially captured; some nondeterminism likely"
	} else {
		analysis.RiskLevel = "high"
		analysis.CanReplay = false
		analysis.Recommendation = "Environment poorly captured; replay not recommended"
	}

	return analysis
}

func AnalyzeCode(code *collectors.CodePayload) DeterminismAnalysis {
	analysis := DeterminismAnalysis{
		Score:        100.0,
		Violations:   []string{},
		Warnings:     []string{},
		ExternalDeps: []string{},
		CanReplay:    true,
	}

	if code == nil {
		analysis.Score = 0
		analysis.Violations = append(analysis.Violations, "no code snapshot")
		analysis.CanReplay = false
		analysis.RiskLevel = "high"
		analysis.Recommendation = "Code version unknown; replay will use whatever is deployed"
		return analysis
	}

	if code.Commit == "" {
		analysis.Violations = append(analysis.Violations, "commit hash missing")
		analysis.CanReplay = false
		analysis.Score -= 40
	}

	if analysis.Score >= 80 {
		analysis.RiskLevel = "low"
		analysis.Recommendation = "Code version pinned; deterministic replay possible"
	} else {
		analysis.RiskLevel = "high"
		analysis.CanReplay = false
		analysis.Recommendation = "Code version not fully captured; replay will be nondeterministic"
	}

	return analysis
}

func AnalyzeConfig(config *collectors.ConfigPayload) DeterminismAnalysis {
	analysis := DeterminismAnalysis{
		Score:        100.0,
		Violations:   []string{},
		Warnings:     []string{},
		ExternalDeps: []string{},
		CanReplay:    true,
	}

	if config == nil {
		analysis.Score = 0
		analysis.Violations = append(analysis.Violations, "no config snapshot")
		analysis.CanReplay = false
		analysis.RiskLevel = "high"
		analysis.Recommendation = "Configuration not captured; replay will use live config"
		return analysis
	}

	if config.Hash == "" {
		analysis.Warnings = append(analysis.Warnings, "config hash missing (integrity cannot be verified)")
		analysis.Score -= 10
	}

	if config.Snapshot == "" {
		analysis.Violations = append(analysis.Violations, "config snapshot empty")
		analysis.CanReplay = false
		analysis.Score -= 50
	}

	if analysis.Score >= 80 {
		analysis.RiskLevel = "low"
		analysis.Recommendation = "Configuration captured with integrity; replay will use recorded config"
	} else if analysis.Score >= 50 {
		analysis.RiskLevel = "medium"
		analysis.Recommendation = "Configuration partially captured; some replay errors likely"
	} else {
		analysis.RiskLevel = "high"
		analysis.CanReplay = false
		analysis.Recommendation = "Configuration not usable; replay will be nondeterministic"
	}

	return analysis
}

func SummarizeAnalyses(envAnalysis, codeAnalysis, configAnalysis DeterminismAnalysis) DeterminismAnalysis {
	summary := DeterminismAnalysis{
		Score:        (envAnalysis.Score + codeAnalysis.Score + configAnalysis.Score) / 3.0,
		Violations:   []string{},
		Warnings:     []string{},
		ExternalDeps: []string{},
	}

	summary.Violations = append(summary.Violations, envAnalysis.Violations...)
	summary.Violations = append(summary.Violations, codeAnalysis.Violations...)
	summary.Violations = append(summary.Violations, configAnalysis.Violations...)

	summary.Warnings = append(summary.Warnings, envAnalysis.Warnings...)
	summary.Warnings = append(summary.Warnings, codeAnalysis.Warnings...)
	summary.Warnings = append(summary.Warnings, configAnalysis.Warnings...)

	summary.CanReplay = envAnalysis.CanReplay && codeAnalysis.CanReplay && configAnalysis.CanReplay

	if summary.Score >= 80 {
		summary.RiskLevel = "low"
		summary.Recommendation = "All three dimensions well-captured; deterministic replay is highly likely"
	} else if summary.Score >= 50 {
		summary.RiskLevel = "medium"
		summary.Recommendation = "Partial capture across dimensions; replay possible but with caveats"
	} else {
		summary.RiskLevel = "high"
		summary.Recommendation = "Insufficient capture; treat reconstruction as forensic only, not authoritative"
	}

	return summary
}

func ReportJSON(analysis DeterminismAnalysis) string {
	data, _ := json.MarshalIndent(analysis, "", "  ")
	return string(data)
}
