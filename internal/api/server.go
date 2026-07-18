package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/ledger"
)

// Server handles HTTP requests for StateLedger API
type Server struct {
	ledger  *ledger.Ledger
	router  *http.ServeMux
	addr    string
	metrics *Metrics
}

// NewServer creates a new API server
func NewServer(l *ledger.Ledger, addr string) *Server {
	s := &Server{
		ledger:  l,
		addr:    addr,
		router:  http.NewServeMux(),
		metrics: NewMetrics(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all API endpoints
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("GET /health", s.handleHealth)

	// Observability
	s.router.HandleFunc("GET /metrics", s.handleMetrics)

	// Ledger endpoints
	s.router.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.router.HandleFunc("GET /api/v1/stats", s.handleStats)
	s.router.HandleFunc("GET /api/v1/records", s.handleListRecords)
	s.router.HandleFunc("GET /api/v1/records/{id}", s.handleGetRecord)
	s.router.HandleFunc("POST /api/v1/records", s.handleCreateRecord)
	s.router.HandleFunc("POST /api/v1/records/bulk", s.handleBulkCreate)
	s.router.HandleFunc("GET /api/v1/verify", s.handleVerify)
	s.router.HandleFunc("GET /api/v1/snapshot", s.handleSnapshot)
	s.router.HandleFunc("POST /api/v1/snapshot", s.handleSnapshot)

	// Verifiable state endpoints
	s.router.HandleFunc("GET /api/v1/merkle", s.handleMerkleRoot)
	s.router.HandleFunc("GET /api/v1/proof/{id}", s.handleProof)
	s.router.HandleFunc("GET /api/v1/signed-root", s.handleSignedRoot)
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

// writeJSON encodes resp as JSON to w. Encoding errors are non-fatal (the
// connection is already being torn down), so the returned error is ignored.
func writeJSON(w http.ResponseWriter, resp *Response) {
	_ = json.NewEncoder(w).Encode(resp)
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, SuccessResponse(map[string]string{
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

	// Cursor-based pagination: when `cursor` is provided, use stable
	// id-based pagination instead of offset pagination.
	cursor := 0
	if c := r.URL.Query().Get("cursor"); c != "" {
		if val, err := strconv.Atoi(c); err == nil && val >= 0 {
			cursor = val
		}
	}

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

	if r.URL.Query().Get("cursor") != "" {
		ledgerRecs, next, err := s.ledger.ListCursor(int64(cursor), limit)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeJSON(w, ErrorResponse(err.Error()))
			return
		}
		var responses []RecordResponse
		for _, rec := range ledgerRecs {
			responses = append(responses, RecordResponse{
				ID:        rec.ID,
				Kind:      rec.Type,
				Timestamp: time.Unix(rec.Timestamp, 0).Format(time.RFC3339),
				Hash:      rec.Hash,
				Payload:   rec.Payload,
			})
		}
		w.WriteHeader(http.StatusOK)
		writeJSON(w, SuccessResponse(map[string]interface{}{
			"records": responses,
			"limit":   limit,
			"cursor":  next,
		}))
		return
	}

	// Get records from ledger
	records, err := s.ledger.List(ledger.ListQuery{
		Since: 0,
		Until: time.Now().Unix(),
		Limit: limit + offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ErrorResponse(err.Error()))
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
	writeJSON(w, SuccessResponse(map[string]interface{}{
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
		writeJSON(w, ErrorResponse("Invalid record ID"))
		return
	}

	rec, err := s.ledger.GetByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, ErrorResponse("Record not found"))
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, SuccessResponse(RecordResponse{
		ID:        rec.ID,
		Kind:      rec.Type,
		Timestamp: time.Unix(rec.Timestamp, 0).Format(time.RFC3339),
		Hash:      rec.Hash,
		Payload:   rec.Payload,
	}))
}

// CreateRecordRequest is the body for POST /api/v1/records.
type CreateRecordRequest struct {
	Type    string `json:"type"`
	Source  string `json:"source"`
	Payload string `json:"payload"`
	Time    int64  `json:"time,omitempty"` // unix seconds; defaults to now
}

// handleCreateRecord creates a new ledger record via the API. This is the
// ingestion endpoint used by external systems (e.g. UltraCache) that want to
// publish verifiable audit events into the ledger.
func (s *Server) handleCreateRecord(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		writeJSON(w, ErrorResponse("Method not allowed"))
		return
	}

	var req CreateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ErrorResponse("invalid JSON body"))
		return
	}

	if strings.TrimSpace(req.Type) == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ErrorResponse("type is required"))
		return
	}
	if strings.TrimSpace(req.Payload) == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ErrorResponse("payload is required"))
		return
	}

	ts := req.Time
	if ts == 0 {
		ts = time.Now().Unix()
	}

	rec, err := s.ledger.Append(ledger.RecordInput{
		Timestamp: ts,
		Type:      req.Type,
		Source:    req.Source,
		Payload:   req.Payload,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, SuccessResponse(RecordResponse{
		ID:        rec.ID,
		Kind:      rec.Type,
		Timestamp: time.Unix(rec.Timestamp, 0).Format(time.RFC3339),
		Hash:      rec.Hash,
		Payload:   rec.Payload,
	}))
}

// BulkCreateRequest is the body for POST /api/v1/records/bulk.
type BulkCreateRequest struct {
	Records []CreateRecordRequest `json:"records"`
}

// handleBulkCreate appends multiple records in a single transaction. This is
// the high-throughput ingestion path used when many audit events arrive at
// once (e.g. a batch of cache mutations from UltraCache).
func (s *Server) handleBulkCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		writeJSON(w, ErrorResponse("Method not allowed"))
		return
	}

	var req BulkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ErrorResponse("invalid JSON body"))
		return
	}

	if len(req.Records) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ErrorResponse("records array is required"))
		return
	}

	inputs := make([]ledger.RecordInput, 0, len(req.Records))
	for i, rec := range req.Records {
		if strings.TrimSpace(rec.Type) == "" {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, ErrorResponse(fmt.Sprintf("records[%d].type is required", i)))
			return
		}
		if strings.TrimSpace(rec.Payload) == "" {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, ErrorResponse(fmt.Sprintf("records[%d].payload is required", i)))
			return
		}
		ts := rec.Time
		if ts == 0 {
			ts = time.Now().Unix()
		}
		inputs = append(inputs, ledger.RecordInput{
			Timestamp: ts,
			Type:      rec.Type,
			Source:    rec.Source,
			Payload:   rec.Payload,
		})
	}

	created, err := s.ledger.AppendBatch(inputs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	responses := make([]RecordResponse, 0, len(created))
	for _, rec := range created {
		responses = append(responses, RecordResponse{
			ID:        rec.ID,
			Kind:      rec.Type,
			Timestamp: time.Unix(rec.Timestamp, 0).Format(time.RFC3339),
			Hash:      rec.Hash,
			Payload:   rec.Payload,
		})
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, SuccessResponse(map[string]interface{}{
		"records": responses,
		"count":   len(responses),
	}))
}

// handleStats returns a summary of the ledger's current state.
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats, err := s.ledger.Stats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, SuccessResponse(stats))
}

// handleMetrics exposes Prometheus-format metrics for scraping.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	if s.metrics == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.metrics.PrometheusMetrics()))
}

// handleVerify verifies ledger integrity
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	result, err := s.ledger.VerifyChain()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, SuccessResponse(map[string]interface{}{
		"valid":     result.OK,
		"checked":   result.Checked,
		"failed_id": result.FailedID,
		"reason":    result.Reason,
		"time":      time.Now().UTC().Format(time.RFC3339),
	}))
}

// SnapshotRequest represents a snapshot query
type SnapshotRequest struct {
	Time      string `json:"time,omitempty"`      // RFC3339 timestamp (default: now)
	Namespace string `json:"namespace,omitempty"` // Filter by namespace
}

// handleSnapshot reconstructs state at a point in time
func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		writeJSON(w, ErrorResponse("Method not allowed"))
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
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, SuccessResponse(map[string]interface{}{
		"time":    targetTime.Format(time.RFC3339),
		"records": records,
		"count":   len(records),
	}))
}

// handleMerkleRoot returns the current Merkle root over all record hashes.
func (s *Server) handleMerkleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	root, err := s.ledger.MerkleRoot()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	writeJSON(w, SuccessResponse(map[string]string{
		"merkle_root": root,
	}))
}

// handleProof returns a Merkle inclusion proof for a record id.
func (s *Server) handleProof(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, ErrorResponse("invalid id"))
		return
	}

	proof, err := s.ledger.MerkleProofFor(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	writeJSON(w, SuccessResponse(proof))
}

// handleSignedRoot returns a signed Merkle root for external verification.
func (s *Server) handleSignedRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	keyPath := r.URL.Query().Get("key")
	if keyPath == "" {
		keyPath = "data/ledger.key"
	}

	signed, err := s.ledger.SignedRootAt(keyPath, time.Now().Unix())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, ErrorResponse(err.Error()))
		return
	}

	writeJSON(w, SuccessResponse(signed))
}
