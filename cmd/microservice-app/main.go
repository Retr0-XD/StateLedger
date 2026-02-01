package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Retr0-XD/StateLedger/internal/api"
	"github.com/Retr0-XD/StateLedger/internal/ledger"
)

type EventType string

const (
	EventUserSignup   EventType = "user.signup"
	EventUserLogin    EventType = "user.login"
	EventUserLogout   EventType = "user.logout"
	EventOrderCreated EventType = "order.created"
	EventOrderShipped EventType = "order.shipped"
	EventPayment      EventType = "payment.processed"
)

type AppEvent struct {
	ID        string      `json:"id"`
	EventType EventType   `json:"event_type"`
	UserID    string      `json:"user_id"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Order struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"created_at"`
}

type MicroserviceApp struct {
	ledger  *ledger.Ledger
	cache   *ledger.Cache
	webhook *ledger.WebhookManager
	metrics *api.Metrics
	port    string

	// In-memory state for demo
	users  map[string]*User
	orders map[string]*Order
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Time    string      `json:"time"`
}

func NewMicroserviceApp(dbPath, port string) (*MicroserviceApp, error) {
	l, err := ledger.Open(dbPath)
	if err != nil {
		return nil, err
	}

	if err := l.InitSchema(); err != nil {
		return nil, err
	}

	return &MicroserviceApp{
		ledger:  l,
		cache:   ledger.NewCache(5 * time.Minute),
		webhook: ledger.NewWebhookManager(),
		metrics: api.NewMetrics(),
		port:    port,
		users:   make(map[string]*User),
		orders:  make(map[string]*Order),
	}, nil
}

func (app *MicroserviceApp) Start() error {
	mux := http.NewServeMux()

	// User endpoints
	mux.HandleFunc("POST /users/register", app.wrap("user.register", app.handleRegisterUser))
	mux.HandleFunc("GET /users/{id}", app.wrap("user.get", app.handleGetUser))
	mux.HandleFunc("POST /users/{id}/login", app.wrap("user.login", app.handleUserLogin))
	mux.HandleFunc("POST /users/{id}/logout", app.wrap("user.logout", app.handleUserLogout))

	// Order endpoints
	mux.HandleFunc("POST /orders", app.wrap("order.create", app.handleCreateOrder))
	mux.HandleFunc("GET /orders/{id}", app.wrap("order.get", app.handleGetOrder))
	mux.HandleFunc("POST /orders/{id}/ship", app.wrap("order.ship", app.handleShipOrder))

	// Payment endpoints
	mux.HandleFunc("POST /payments", app.wrap("payment.process", app.handleProcessPayment))

	// Query & Audit
	mux.HandleFunc("GET /events", app.wrap("events.list", app.handleListEvents))
	mux.HandleFunc("GET /audit/user/{id}", app.wrap("audit.user", app.handleUserAudit))
	mux.HandleFunc("GET /audit/order/{id}", app.wrap("audit.order", app.handleOrderAudit))
	mux.HandleFunc("GET /health", app.wrap("health", app.handleHealth))
	mux.HandleFunc("GET /metrics", app.wrap("metrics", app.handleMetrics))

	handler := api.Chain(
		mux,
		api.RecoveryMiddleware(),
		api.LoggingMiddleware(),
		api.RequestIDMiddleware(),
		api.RateLimitMiddleware(api.NewRateLimiter(100, 200)),
		api.CORSMiddleware([]string{"*"}),
	)

	log.Printf("Microservice starting on :%s", app.port)
	return http.ListenAndServe(":"+app.port, handler)
}

func (app *MicroserviceApp) wrap(endpoint string, handler func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		err := handler(w, r)
		duration := time.Since(start)
		app.metrics.RecordRequest(endpoint, duration, err)
		if err != nil && w.Header().Get("Content-Type") == "" {
			writeJSON(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   err.Error(),
				Time:    time.Now().UTC().Format(time.RFC3339),
			})
		}
	}
}

func (app *MicroserviceApp) recordEvent(eventType EventType, userID string, data interface{}) error {
	eventData := AppEvent{
		ID:        fmt.Sprintf("%d-%s", time.Now().UnixNano(), userID),
		EventType: eventType,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	payload, err := json.Marshal(eventData)
	if err != nil {
		return err
	}

	compressed, err := ledger.CompressPayload(string(payload))
	if err != nil {
		return err
	}

	payloadStr := string(compressed)
	if len(payloadStr) > 100 {
		payloadStr = payloadStr[:100]
	}

	rec, err := app.ledger.Append(ledger.RecordInput{
		Timestamp: time.Now().Unix(),
		Type:      "event",
		Source:    "microservice",
		Payload:   fmt.Sprintf("%s:%s:%s", eventType, userID, payloadStr),
	})
	if err != nil {
		return err
	}

	app.webhook.Publish(ledger.WebhookEvent{
		EventType: ledger.EventRecordAppended,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"event_type": eventType,
			"user_id":    userID,
			"record_id":  rec.ID,
		},
	})

	cacheKey := fmt.Sprintf("event:%s:%s", eventType, userID)
	app.cache.Set(cacheKey, eventData)

	return nil
}

func (app *MicroserviceApp) handleHealth(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"status": "healthy",
			"app":    "microservice",
		},
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleRegisterUser(w http.ResponseWriter, r *http.Request) error {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return err
	}

	if user.ID == "" || user.Name == "" || user.Email == "" {
		return errors.New("id, name, and email required")
	}

	app.users[user.ID] = &user

	if err := app.recordEvent(EventUserSignup, user.ID, map[string]interface{}{
		"name":  user.Name,
		"email": user.Email,
	}); err != nil {
		return err
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    user,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleGetUser(w http.ResponseWriter, r *http.Request) error {
	userID := r.PathValue("id")
	user, ok := app.users[userID]
	if !ok {
		return errors.New("user not found")
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    user,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleUserLogin(w http.ResponseWriter, r *http.Request) error {
	userID := r.PathValue("id")
	if _, ok := app.users[userID]; !ok {
		return errors.New("user not found")
	}

	if err := app.recordEvent(EventUserLogin, userID, map[string]interface{}{
		"ip": r.RemoteAddr,
	}); err != nil {
		return err
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"user_id": userID,
			"status":  "logged_in",
		},
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleUserLogout(w http.ResponseWriter, r *http.Request) error {
	userID := r.PathValue("id")
	if _, ok := app.users[userID]; !ok {
		return errors.New("user not found")
	}

	if err := app.recordEvent(EventUserLogout, userID, map[string]interface{}{}); err != nil {
		return err
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"user_id": userID,
			"status":  "logged_out",
		},
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleCreateOrder(w http.ResponseWriter, r *http.Request) error {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		return err
	}

	if order.ID == "" || order.UserID == "" || order.Amount <= 0 {
		return errors.New("id, user_id, and amount required")
	}

	order.Status = "created"
	order.CreatedAt = time.Now().Unix()
	app.orders[order.ID] = &order

	if err := app.recordEvent(EventOrderCreated, order.UserID, order); err != nil {
		return err
	}

	writeJSON(w, http.StatusCreated, APIResponse{
		Success: true,
		Data:    order,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleGetOrder(w http.ResponseWriter, r *http.Request) error {
	orderID := r.PathValue("id")
	order, ok := app.orders[orderID]
	if !ok {
		return errors.New("order not found")
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    order,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleShipOrder(w http.ResponseWriter, r *http.Request) error {
	orderID := r.PathValue("id")
	order, ok := app.orders[orderID]
	if !ok {
		return errors.New("order not found")
	}

	order.Status = "shipped"

	if err := app.recordEvent(EventOrderShipped, order.UserID, order); err != nil {
		return err
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    order,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleProcessPayment(w http.ResponseWriter, r *http.Request) error {
	var payment struct {
		OrderID string  `json:"order_id"`
		Amount  float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		return err
	}

	order, ok := app.orders[payment.OrderID]
	if !ok {
		return errors.New("order not found")
	}

	if err := app.recordEvent(EventPayment, order.UserID, map[string]interface{}{
		"order_id": payment.OrderID,
		"amount":   payment.Amount,
		"status":   "processed",
	}); err != nil {
		return err
	}

	order.Status = "paid"

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"order_id": payment.OrderID,
			"amount":   payment.Amount,
			"status":   "processed",
		},
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleListEvents(w http.ResponseWriter, r *http.Request) error {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	records, err := app.ledger.List(ledger.ListQuery{
		Since: 0,
		Until: time.Now().Unix(),
		Limit: limit,
	})
	if err != nil {
		return err
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"count":   len(records),
			"records": records,
		},
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleUserAudit(w http.ResponseWriter, r *http.Request) error {
	userID := r.PathValue("id")
	records, err := app.ledger.List(ledger.ListQuery{
		Since: 0,
		Until: time.Now().Unix(),
		Limit: 1000,
	})
	if err != nil {
		return err
	}

	filtered := []ledger.Record{}
	for _, rec := range records {
		if contains(rec.Payload, userID) {
			filtered = append(filtered, rec)
		}
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"user_id": userID,
			"count":   len(filtered),
			"events":  filtered,
		},
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleOrderAudit(w http.ResponseWriter, r *http.Request) error {
	orderID := r.PathValue("id")
	records, err := app.ledger.List(ledger.ListQuery{
		Since: 0,
		Until: time.Now().Unix(),
		Limit: 1000,
	})
	if err != nil {
		return err
	}

	filtered := []ledger.Record{}
	for _, rec := range records {
		if contains(rec.Payload, orderID) {
			filtered = append(filtered, rec)
		}
	}

	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"order_id": orderID,
			"count":    len(filtered),
			"events":   filtered,
		},
		Time: time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func (app *MicroserviceApp) handleMetrics(w http.ResponseWriter, r *http.Request) error {
	stats := app.metrics.GetStats()
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    stats,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if s[i : i+1] == substr[0:1] {
			if len(s[i:]) >= len(substr) && s[i:i+len(substr)] == substr {
				return true
			}
		}
	}
	return false
}

func (app *MicroserviceApp) Close() error {
	return app.ledger.Close()
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data/microservice.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app, err := NewMicroserviceApp(dbPath, port)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}
	defer app.Close()

	if err := app.Start(); err != nil {
		log.Fatalf("failed to start app: %v", err)
	}
}
