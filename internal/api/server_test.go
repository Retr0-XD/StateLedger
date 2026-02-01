package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/ledger"
)

func setupTestServer(t *testing.T) *Server {
	l, err := ledger.Open(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test ledger: %v", err)
	}
	if err := l.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
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
	req := httptest.NewRequest("POST", "/api/v1/records", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("Expected 501, got %d", w.Code)
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
