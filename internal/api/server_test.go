package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/ledger"
)

func setupTestServer(t *testing.T) *Server {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	l, err := ledger.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test ledger: %v", err)
	}
	if err := l.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	t.Cleanup(func() {
		_ = l.Close()
		_ = os.Remove(dbPath)
	})
	return NewServer(l, "localhost:8080")
}

func TestHandleHealth(t *testing.T) {
	s := setupTestServer(t)
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success response")
	}
}

func TestHandleListRecords(t *testing.T) {
	s := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/records", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success response")
	}
}

func TestHandleListRecordsWithPagination(t *testing.T) {
	s := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/records?limit=50&offset=0", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success response")
	}
}

func TestHandleGetRecord(t *testing.T) {
	s := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/records/1", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	// Record may not exist, but request should be valid
	if w.Code != http.StatusNotFound && w.Code != http.StatusOK {
		t.Errorf("Expected 200 or 404, got %d", w.Code)
	}
}

func TestHandleGetRecordInvalidID(t *testing.T) {
	s := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/records/invalid", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleVerify(t *testing.T) {
	s := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/verify", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success response")
	}
}

func TestHandleSnapshot(t *testing.T) {
	s := setupTestServer(t)

	// Test with GET
	req := httptest.NewRequest("GET", "/api/v1/snapshot?time="+time.Now().Format(time.RFC3339), nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success response")
	}
}

func TestHandleSnapshotPOST(t *testing.T) {
	s := setupTestServer(t)

	reqBody := SnapshotRequest{
		Time: time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/snapshot", bytes.NewReader(body))
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestHandleCreateRecord(t *testing.T) {
	s := setupTestServer(t)

	body, _ := json.Marshal(CreateRecordRequest{
		Type:    "deployment",
		Source:  "ci-pipeline",
		Payload: "Deployed v1.2.3",
	})
	req := httptest.NewRequest("POST", "/api/v1/records", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Data    RecordResponse
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Success {
		t.Error("Expected success=true")
	}
	if resp.Data.ID == 0 {
		t.Error("Expected a non-zero record id")
	}
	if resp.Data.Kind != "deployment" {
		t.Errorf("Expected kind 'deployment', got %q", resp.Data.Kind)
	}
}

func TestHandleCreateRecordValidation(t *testing.T) {
	s := setupTestServer(t)
	// Missing type and payload -> 400 Bad Request.
	req := httptest.NewRequest("POST", "/api/v1/records", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestErrorResponse(t *testing.T) {
	resp := ErrorResponse("test error")
	if resp.Success {
		t.Error("Expected success=false")
	}
	if resp.Error != "test error" {
		t.Errorf("Expected 'test error', got '%s'", resp.Error)
	}
}

func TestSuccessResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := SuccessResponse(data)
	if !resp.Success {
		t.Error("Expected success=true")
	}
	if resp.Data == nil {
		t.Error("Expected data to be set")
	}
}

func TestHandleStats(t *testing.T) {
	s := setupTestServer(t)

	// Seed a couple of records.
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(CreateRecordRequest{
			Type:    "config",
			Source:  "test",
			Payload: "payload",
		})
		req := httptest.NewRequest("POST", "/api/v1/records", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("seed record failed: %d", w.Code)
		}
	}

	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	var resp struct {
		Success bool `json:"success"`
		Data    ledger.LedgerStats
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Error("Expected success=true")
	}
	if resp.Data.RecordCount != 3 {
		t.Errorf("Expected 3 records, got %d", resp.Data.RecordCount)
	}
	if resp.Data.LastHash == "" {
		t.Error("Expected non-empty last hash")
	}
	if resp.Data.MerkleRoot == "" {
		t.Error("Expected non-empty merkle root")
	}
}

func TestHandleBulkCreate(t *testing.T) {
	s := setupTestServer(t)

	bulk := BulkCreateRequest{
		Records: []CreateRecordRequest{
			{Type: "code", Source: "s1", Payload: "p1"},
			{Type: "code", Source: "s2", Payload: "p2"},
			{Type: "code", Source: "s3", Payload: "p3"},
		},
	}
	body, _ := json.Marshal(bulk)
	req := httptest.NewRequest("POST", "/api/v1/records/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Records []RecordResponse `json:"records"`
			Count   int              `json:"count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Success {
		t.Error("Expected success=true")
	}
	if resp.Data.Count != 3 {
		t.Errorf("Expected 3 created, got %d", resp.Data.Count)
	}
}

func TestHandleListRecordsCursor(t *testing.T) {
	s := setupTestServer(t)

	// Seed 5 records.
	for i := 0; i < 5; i++ {
		body, _ := json.Marshal(CreateRecordRequest{
			Type:    "config",
			Source:  "test",
			Payload: "payload",
		})
		req := httptest.NewRequest("POST", "/api/v1/records", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("seed record failed: %d", w.Code)
		}
	}

	// First page (limit 2).
	req := httptest.NewRequest("GET", "/api/v1/records?cursor=0&limit=2", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var page1 struct {
		Data struct {
			Records []RecordResponse `json:"records"`
			Cursor  int64            `json:"cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page1); err != nil {
		t.Fatalf("failed to decode page1: %v", err)
	}
	if len(page1.Data.Records) != 2 {
		t.Fatalf("Expected 2 records on page1, got %d", len(page1.Data.Records))
	}
	if page1.Data.Cursor == 0 {
		t.Error("Expected a non-zero next cursor")
	}

	// Second page using the cursor.
	req2 := httptest.NewRequest("GET", "/api/v1/records?cursor="+itoa(page1.Data.Cursor)+"&limit=2", nil)
	w2 := httptest.NewRecorder()
	s.router.ServeHTTP(w2, req2)
	var page2 struct {
		Data struct {
			Records []RecordResponse `json:"records"`
			Cursor  int64            `json:"cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &page2); err != nil {
		t.Fatalf("failed to decode page2: %v", err)
	}
	if len(page2.Data.Records) != 2 {
		t.Fatalf("Expected 2 records on page2, got %d", len(page2.Data.Records))
	}
	if page2.Data.Records[0].ID <= page1.Data.Records[1].ID {
		t.Error("Expected page2 records to have higher ids than page1")
	}
}

func TestHandleMetrics(t *testing.T) {
	s := setupTestServer(t)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain; version=0.0.4" {
		t.Errorf("Expected prometheus content type, got %q", ct)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("stateledger_requests_total")) {
		t.Error("Expected prometheus metric in output")
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
