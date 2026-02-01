package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestHandleListRecords_LimitParams(t *testing.T) {
	s := setupTestServer(t)

	// negative limit
	req := httptest.NewRequest("GET", "/api/v1/records?limit=-1", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for negative limit, got %d", w.Code)
	}

	// illegal limit value
	req = httptest.NewRequest("GET", "/api/v1/records?limit=abd", nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for illegal limit value, got %d", w.Code)
	}

	// max limit exceed
	req = httptest.NewRequest("GET", "/api/v1/records?limit=2000", nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for exceeding max limit, got %d", w.Code)
	}
}

func TestHandleListRecords_OffsetParams(t *testing.T) {
	s := setupTestServer(t)

	// invalid offset field
	req := httptest.NewRequest("GET", "/api/v1/records?offset=abc", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid offset value, got %d", w.Code)
	}

	// negatie offset field
	req = httptest.NewRequest("GET", "/api/v1/records?offset=-1", nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for negative offset, got %d", w.Code)
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
	validTime := url.QueryEscape(time.Now().Format(time.RFC3339))
	req := httptest.NewRequest("GET", "/api/v1/snapshot?time="+validTime, nil)
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

func TestHandleSnapShot_Timestamp(t *testing.T) {
	s := setupTestServer(t)

	// illegal timestamp
	req := httptest.NewRequest("GET", "/api/v1/snapshot?time=abbc", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for illegal timestamp, got %d", w.Code)
	}

	// non-RFC3339 format
	req = httptest.NewRequest("GET", "/api/v1/snapshot?time=01/01/2001", nil)
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for non-RFC3339 format, got %d", w.Code)
	}
}
func TestHandleSnapShot_Payload(t *testing.T) {
	s := setupTestServer(t)

	// malformed json
	req := httptest.NewRequest("POST", "/api/v1/snapshot", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for malformed json, got %d", w.Code)
	}

	// empty body
	req = httptest.NewRequest("POST", "/api/v1/snapshot", bytes.NewReader([]byte("")))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for empty body, got %d", w.Code)
	}

	// invalid field types in json
	req = httptest.NewRequest("POST", "/api/v1/snapshot", bytes.NewReader([]byte(`{"time": 12345}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid field type, got %d", w.Code)
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
