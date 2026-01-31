package ledger

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestLedger(t *testing.T) *Ledger {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ledger.db")

	l, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open ledger: %v", err)
	}

	if err := l.InitSchema(); err != nil {
		_ = l.Close()
		t.Fatalf("init schema: %v", err)
	}

	return l
}

func TestAppendAndVerifyChain(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	_, err := l.Append(RecordInput{Timestamp: 1000, Type: "code", Source: "test", Payload: `{"repo":"app","commit":"abc1234"}`})
	if err != nil {
		t.Fatalf("append code: %v", err)
	}

	_, err = l.Append(RecordInput{Timestamp: 1001, Type: "environment", Source: "test", Payload: `{"os":"linux","runtime":"go","arch":"amd64","time_source":"system"}`})
	if err != nil {
		t.Fatalf("append env: %v", err)
	}

	result, err := l.VerifyChain()
	if err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	if !result.OK || result.Checked != 2 {
		t.Fatalf("verify result unexpected: %+v", result)
	}
}

func TestVerifyUpTo(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	_, _ = l.Append(RecordInput{Timestamp: 1000, Type: "code", Source: "test", Payload: `{"repo":"app","commit":"abc1234"}`})
	_, _ = l.Append(RecordInput{Timestamp: 2000, Type: "config", Source: "test", Payload: `{"source":"cfg","version":"1","hash":"sha256:abc","snapshot":"x"}`})

	proof, err := l.VerifyUpTo(1500)
	if err != nil {
		t.Fatalf("verify up to: %v", err)
	}
	if !proof.OK || proof.Checked != 1 {
		t.Fatalf("proof unexpected: %+v", proof)
	}
}

func TestReconstructAtTimeIncludesProof(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	_, _ = l.Append(RecordInput{Timestamp: 1000, Type: "code", Source: "test", Payload: `{"repo":"app","commit":"abc1234"}`})
	_, _ = l.Append(RecordInput{Timestamp: 1001, Type: "environment", Source: "test", Payload: `{"os":"linux","runtime":"go","arch":"amd64","time_source":"system"}`})
	_, _ = l.Append(RecordInput{Timestamp: 1002, Type: "mutation", Source: "test", Payload: `{"type":"order_created","id":"evt-1","source":"svc","hash":"sha256:x","external_ref":"kafka:42"}`})

	rec := New(l)
	report := rec.ReconstructAtTime(1002)

	if report.Proof == nil || !report.Proof.OK {
		t.Fatalf("expected proof ok, got: %+v", report.Proof)
	}
	if report.ReplayPlan == nil || report.ReplayPlan.Total != 1 {
		t.Fatalf("expected replay plan, got: %+v", report.ReplayPlan)
	}
}

func TestReplayPlanOrderingByNamespace(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	_, _ = l.Append(RecordInput{Timestamp: 1000, Type: "mutation", Source: "test", Payload: `{"type":"order_created","id":"evt-2","source":"svc","hash":"sha256:x","external_ref":"kafka:2"}`})
	_, _ = l.Append(RecordInput{Timestamp: 1000, Type: "mutation", Source: "test", Payload: `{"type":"order_created","id":"evt-1","source":"svc","hash":"sha256:x","external_ref":"kafka:1"}`})

	rec := New(l)
	report := rec.ReconstructAtTime(2000)
	if report.ReplayPlan == nil || len(report.ReplayPlan.Namespaces) != 1 {
		t.Fatalf("expected single namespace plan, got: %+v", report.ReplayPlan)
	}

	plan := report.ReplayPlan.Namespaces[0]
	if !plan.Ordered || plan.Count != 2 {
		t.Fatalf("unexpected ordering: %+v", plan)
	}
	if plan.Records[0].ExternalRef != "kafka:1" {
		t.Fatalf("expected kafka:1 first, got: %s", plan.Records[0].ExternalRef)
	}
}

func TestConfigHashProvenance(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	payload := `{"source":"cfg","version":"1","hash":"sha256:wrong","snapshot":"value"}`
	_, err := l.Append(RecordInput{Timestamp: 1000, Type: "config", Source: "test", Payload: payload})
	if err != nil {
		t.Fatalf("append config: %v", err)
	}

	rec := New(l)
	report := rec.ReconstructAtTime(1000)
	found := false
	for _, issue := range report.Issues {
		if issue == "provenance: config hash mismatch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected config hash mismatch warning, got: %+v", report.Issues)
	}
}

func TestArtifactStore(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	path := filepath.Join(t.TempDir(), "artifact.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := l.Append(RecordInput{Timestamp: 1000, Type: "code", Source: "test", Payload: `{"repo":"app","commit":"abc1234"}`})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
}
