package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Retr0-XD/StateLedger/internal/ledger"
)

func BenchmarkHealthEndpoint(b *testing.B) {
	s := setupBenchServer(b)
	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
	}
}

func BenchmarkListRecords(b *testing.B) {
	s := setupBenchServer(b)

	// Seed with 100 records
	for i := 0; i < 100; i++ {
		_, _ = s.ledger.Append(ledger.RecordInput{
			Timestamp: 1234567890,
			Type:      "code",
			Source:    "benchmark",
			Payload:   `{"test": "data"}`,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/records?limit=50", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
	}
}

func BenchmarkGetRecord(b *testing.B) {
	s := setupBenchServer(b)

	// Seed with record
	_, _ = s.ledger.Append(ledger.RecordInput{
		Timestamp: 1234567890,
		Type:      "code",
		Source:    "benchmark",
		Payload:   `{"test": "data"}`,
	})

	req := httptest.NewRequest("GET", "/api/v1/records/1", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
	}
}

func BenchmarkVerifyEndpoint(b *testing.B) {
	s := setupBenchServer(b)

	// Seed with 50 records
	for i := 0; i < 50; i++ {
		_, _ = s.ledger.Append(ledger.RecordInput{
			Timestamp: 1234567890,
			Type:      "code",
			Source:    "benchmark",
			Payload:   `{"test": "data"}`,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/verify", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
	}
}

func BenchmarkSnapshotEndpoint(b *testing.B) {
	s := setupBenchServer(b)

	// Seed with 100 records
	for i := 0; i < 100; i++ {
		_, _ = s.ledger.Append(ledger.RecordInput{
			Timestamp: 1234567890,
			Type:      "code",
			Source:    "benchmark",
			Payload:   `{"test": "data"}`,
		})
	}

	req := httptest.NewRequest("GET", "/api/v1/snapshot", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
	}
}

func BenchmarkJSONEncoding(b *testing.B) {
	resp := SuccessResponse(map[string]interface{}{
		"records": []RecordResponse{
			{ID: 1, Kind: "code", Timestamp: "2026-01-01T00:00:00Z", Hash: "abc123", Payload: `{"test": "data"}`},
			{ID: 2, Kind: "config", Timestamp: "2026-01-01T01:00:00Z", Hash: "def456", Payload: `{"test": "data2"}`},
		},
		"total": 2,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		_ = json.NewEncoder(buf).Encode(resp)
	}
}

func setupBenchServer(b *testing.B) *Server {
	l, err := ledger.Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	if err := l.InitSchema(); err != nil {
		b.Fatal(err)
	}
	return NewServer(l, "localhost:8080")
}
