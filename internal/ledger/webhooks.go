package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// WebhookEvent represents an event that can be sent via webhook
type WebhookEvent struct {
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// WebhookManager manages webhook subscriptions and delivery
type WebhookManager struct {
	mu           sync.RWMutex
	subscriptions map[string]*Subscription
	httpClient   *http.Client
	maxRetries   int
	retryDelay   time.Duration
}

// Subscription represents a webhook subscription
type Subscription struct {
	ID        string
	URL       string
	Events    []string // event types to subscribe to
	Secret    string   // for HMAC verification
	Active    bool
	CreatedAt time.Time
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager() *WebhookManager {
	return &WebhookManager{
		subscriptions: make(map[string]*Subscription),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		maxRetries: 3,
		retryDelay: 2 * time.Second,
	}
}

// Subscribe adds a new webhook subscription
func (wm *WebhookManager) Subscribe(id, url string, events []string, secret string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.subscriptions[id]; exists {
		return fmt.Errorf("subscription %s already exists", id)
	}

	wm.subscriptions[id] = &Subscription{
		ID:        id,
		URL:       url,
		Events:    events,
		Secret:    secret,
		Active:    true,
		CreatedAt: time.Now(),
	}

	return nil
}

// Unsubscribe removes a webhook subscription
func (wm *WebhookManager) Unsubscribe(id string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.subscriptions[id]; !exists {
		return fmt.Errorf("subscription %s not found", id)
	}

	delete(wm.subscriptions, id)
	return nil
}

// Publish sends an event to all matching subscribers
func (wm *WebhookManager) Publish(event WebhookEvent) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	for _, sub := range wm.subscriptions {
		if !sub.Active {
			continue
		}

		// Check if subscription is interested in this event type
		if !sub.wantsEvent(event.EventType) {
			continue
		}

		// Send webhook asynchronously
		go wm.deliverWebhook(sub, event)
	}
}

// deliverWebhook attempts to deliver a webhook with retries
func (wm *WebhookManager) deliverWebhook(sub *Subscription, event WebhookEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	for attempt := 0; attempt < wm.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(wm.retryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("POST", sub.URL, bytes.NewReader(payload))
		if err != nil {
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-ID", sub.ID)
		req.Header.Set("X-Event-Type", event.EventType)

		// Add HMAC signature if secret is provided
		if sub.Secret != "" {
			// TODO: Implement HMAC signature
			// sig := hmac256(sub.Secret, payload)
			// req.Header.Set("X-Signature", sig)
		}

		resp, err := wm.httpClient.Do(req)
		if err != nil {
			continue
		}

		resp.Body.Close()

		// Success if 2xx status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return
		}
	}

	// All retries failed - could log this or mark subscription inactive
}

// wantsEvent checks if subscription wants this event type
func (sub *Subscription) wantsEvent(eventType string) bool {
	if len(sub.Events) == 0 {
		return true // subscribe to all events
	}

	for _, e := range sub.Events {
		if e == eventType || e == "*" {
			return true
		}
	}

	return false
}

// ListSubscriptions returns all active subscriptions
func (wm *WebhookManager) ListSubscriptions() []*Subscription {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	subs := make([]*Subscription, 0, len(wm.subscriptions))
	for _, sub := range wm.subscriptions {
		subs = append(subs, sub)
	}

	return subs
}

// Event types
const (
	EventRecordAppended = "record.appended"
	EventBatchAppended  = "batch.appended"
	EventChainVerified  = "chain.verified"
	EventSnapshotTaken  = "snapshot.taken"
)
