package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/ledger"
)

// Server handles HTTP requests for StateLedger API
type Server struct {
	ledger *ledger.Ledger
	router *http.ServeMux
	addr   string
}

// NewServer creates a new API server
func NewServer(l *ledger.Ledger, addr string) *Server {
	s := &Server{
		ledger: l,
		addr:   addr,
		router: http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all API endpoints
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("GET /health", s.handleHealth)

	// Ledger endpoints
	s.router.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.router.HandleFunc("GET /api/v1/records", s.handleListRecords)
	s.router.HandleFunc("GET /api/v1/records/{id}", s.handleGetRecord)
	s.router.HandleFunc("POST /api/v1/records", s.handleCreateRecord)
	s.router.HandleFunc("GET /api/v1/verify", s.handleVerify)
	s.router.HandleFunc("GET /api/v1/snapshot", s.handleSnapshot)
	s.router.HandleFunc("POST /api/v1/snapshot", s.handleSnapshot)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	fmt.Printf("Starting StateLedger API server on %s\n", s.addr)
	return http.ListenAndServe(s.addr, s.router)
}

// Response wraps API responses
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Time    string      `json:"time"`
}

// ErrorResponse creates an error response
func ErrorResponse(msg string) *Response {
	return &Response{
		Success: false,
		Error:   msg,
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
}

// SuccessResponse creates a success response
func SuccessResponse(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Time:    time.Now().UTC().Format(time.RFC3339),
	}
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse(map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}))
}

// ListRecordsRequest represents query parameters for listing records
type ListRecordsRequest struct {
	Kind      string `json:"kind,omitempty"`      // code, config, environment, mutation
	Namespace string `json:"namespace,omitempty"` // For filtering
	Limit     int    `json:"limit,omitempty"`     // Default 100
	Offset    int    `json:"offset,omitempty"`    // For pagination
	From      string `json:"from,omitempty"`      // RFC3339 timestamp
	To        string `json:"to,omitempty"`        // RFC3339 timestamp
}

// RecordResponse represents a ledger record in API response
type RecordResponse struct {
	ID        int64       `json:"id"`
	Kind      string      `json:"kind"`
	Timestamp string      `json:"timestamp"`
	Hash      string      `json:"hash"`
	Payload   interface{} `json:"payload"`
}

// handleListRecords lists records with optional filtering
func (s *Server) handleListRecords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	limit := 100
	offset := 0

	// Parse query parameters
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 1000 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	// Get records from ledger
	records, err := s.ledger.List(ledger.ListQuery{
		Since: 0,
		Until: time.Now().Unix(),
		Limit: limit + offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse(err.Error()))
		return
	}

	// Apply pagination
	if offset > len(records) {
		offset = len(records)
	}
	end := offset + limit
	if end > len(records) {
		end = len(records)
	}

	records = records[offset:end]

	// Convert to response format
	var responses []RecordResponse
	for _, rec := range records {
		responses = append(responses, RecordResponse{
			ID:        rec.ID,
			Kind:      rec.Type,
			Timestamp: time.Unix(rec.Timestamp, 0).Format(time.RFC3339),
			Hash:      rec.Hash,
			Payload:   rec.Payload,
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse(map[string]interface{}{
		"records": responses,
		"offset":  offset,
		"limit":   limit,
		"total":   len(records),
	}))
}

// handleGetRecord retrieves a specific record
func (s *Server) handleGetRecord(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse("Invalid record ID"))
		return
	}

	rec, err := s.ledger.GetByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse("Record not found"))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse(RecordResponse{
		ID:        rec.ID,
		Kind:      rec.Type,
		Timestamp: time.Unix(rec.Timestamp, 0).Format(time.RFC3339),
		Hash:      rec.Hash,
		Payload:   rec.Payload,
	}))
}

// handleCreateRecord creates a new record (placeholder)
func (s *Server) handleCreateRecord(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(ErrorResponse("Record creation via API not implemented yet"))
}

// handleVerify verifies ledger integrity
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	result, err := s.ledger.VerifyChain()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse(map[string]interface{}{
		"valid":     result.OK,
		"checked":   result.Checked,
		"failed_id": result.FailedID,
		"reason":    result.Reason,
		"time":      time.Now().UTC().Format(time.RFC3339),
	}))
}

// SnapshotRequest represents a snapshot query
type SnapshotRequest struct {
	Time      string `json:"time,omitempty"`       // RFC3339 timestamp (default: now)
	Namespace string `json:"namespace,omitempty"`  // Filter by namespace
}

// handleSnapshot reconstructs state at a point in time
func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse("Method not allowed"))
		return
	}

	targetTime := time.Now()
	if ts := r.URL.Query().Get("time"); ts != "" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			targetTime = t
		}
	}

	records, err := s.ledger.List(ledger.ListQuery{
		Since: 0,
		Until: targetTime.Unix(),
		Limit: 1000,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SuccessResponse(map[string]interface{}{
		"time":    targetTime.Format(time.RFC3339),
		"records": records,
		"count":   len(records),
	}))
}
