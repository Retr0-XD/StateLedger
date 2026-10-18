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

func TestCompactKeepLastN(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	for i := int64(0); i < 10; i++ {
		if _, err := l.Append(RecordInput{Timestamp: 1000 + i, Type: "code", Source: "test", Payload: `{"n":1}`}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	res, err := l.Compact(CompactOptions{KeepLastN: 3})
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if res.Remaining != 3 {
		t.Fatalf("expected 3 remaining, got %d", res.Remaining)
	}
	if res.Removed != 7 {
		t.Fatalf("expected 7 removed, got %d", res.Removed)
	}

	// Chain must still verify after pruning.
	vr, err := l.VerifyChain()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !vr.OK {
		t.Fatalf("chain broken after compact: %+v", vr)
	}
}

func TestCompactRebuildChain(t *testing.T) {
	l := newTestLedger(t)
	defer l.Close()

	for i := int64(0); i < 5; i++ {
		if _, err := l.Append(RecordInput{Timestamp: 1000 + i, Type: "code", Source: "test", Payload: `{"n":1}`}); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	// Tamper with a stored hash to simulate corruption.
	if _, err := l.db.Exec(`UPDATE ledger_records SET hash = 'deadbeef' WHERE id = 2`); err != nil {
		t.Fatalf("tamper: %v", err)
	}
	if vr, _ := l.VerifyChain(); vr.OK {
		t.Fatalf("expected chain to be broken before rebuild")
	}

	res, err := l.Compact(CompactOptions{RebuildChain: true})
	if err != nil {
		t.Fatalf("compact rebuild: %v", err)
	}
	if !res.Rebuilt {
		t.Fatalf("expected rebuilt=true")
	}

	vr, err := l.VerifyChain()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !vr.OK {
		t.Fatalf("chain should be consistent after rebuild: %+v", vr)
	}
}

func TestWALRecovery(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ledger.db")
	walPath := filepath.Join(dir, "ledger.wal")

	l, err := OpenWithOptions(dbPath, OpenOptions{WALPath: walPath, WALFsync: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := l.InitSchema(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Append a record; it should be committed and the WAL truncated on recovery.
	if _, err := l.Append(RecordInput{Timestamp: 1000, Type: "code", Source: "test", Payload: `{"n":1}`}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen: WAL should be empty (committed), so no duplicate records.
	l2, err := OpenWithOptions(dbPath, OpenOptions{WALPath: walPath, WALFsync: true})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer l2.Close()

	recs, err := l2.List(ListQuery{Limit: 1000})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record after recovery, got %d", len(recs))
	}
}

func TestWALRecoveryReplaysPending(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ledger.db")
	walPath := filepath.Join(dir, "ledger.wal")

	l, err := OpenWithOptions(dbPath, OpenOptions{WALPath: walPath, WALFsync: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := l.InitSchema(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Manually write a WAL entry that is NOT marked committed, simulating a
	// crash between WAL write and DB commit.
	w := l.wal
	if err := w.Log(1, RecordInput{Timestamp: 2000, Type: "code", Source: "test", Payload: `{"pending":true}`}); err != nil {
		t.Fatalf("wal log: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen: the pending entry must be replayed into the ledger.
	l2, err := OpenWithOptions(dbPath, OpenOptions{WALPath: walPath, WALFsync: true})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer l2.Close()

	recs, err := l2.List(ListQuery{Limit: 1000})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 replayed record, got %d", len(recs))
	}
	if recs[0].Payload != `{"pending":true}` {
		t.Fatalf("unexpected payload: %s", recs[0].Payload)
	}
}
