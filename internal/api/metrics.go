package api

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks API performance metrics
type Metrics struct {
	mu sync.RWMutex

	// Request metrics
	totalRequests     atomic.Uint64
	failedRequests    atomic.Uint64
	requestDurations  []time.Duration
	maxDurationsCount int

	// Endpoint-specific metrics
	healthChecks      atomic.Uint64
	listRecords       atomic.Uint64
	getRecord         atomic.Uint64
	verifyChain       atomic.Uint64
	snapshots         atomic.Uint64
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		maxDurationsCount: 1000,
		requestDurations:  make([]time.Duration, 0, 1000),
	}
}

// RecordRequest records a completed request
func (m *Metrics) RecordRequest(endpoint string, duration time.Duration, err error) {
	m.totalRequests.Add(1)

	if err != nil {
		m.failedRequests.Add(1)
	}

	m.mu.Lock()
	if len(m.requestDurations) < m.maxDurationsCount {
		m.requestDurations = append(m.requestDurations, duration)
	} else {
		m.requestDurations = m.requestDurations[1:]
		m.requestDurations = append(m.requestDurations, duration)
	}
	m.mu.Unlock()

	// Track endpoint-specific metrics
	switch endpoint {
	case "health":
		m.healthChecks.Add(1)
	case "list":
		m.listRecords.Add(1)
	case "get":
		m.getRecord.Add(1)
	case "verify":
		m.verifyChain.Add(1)
	case "snapshot":
		m.snapshots.Add(1)
	}
}

// GetStats returns current metrics statistics
func (m *Metrics) GetStats() MetricsStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total time.Duration
	var min, max time.Duration
	if len(m.requestDurations) > 0 {
		min = m.requestDurations[0]
		max = m.requestDurations[0]
		for _, d := range m.requestDurations {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}
	}

	avg := time.Duration(0)
	if len(m.requestDurations) > 0 {
		avg = total / time.Duration(len(m.requestDurations))
	}

	return MetricsStats{
		TotalRequests:  m.totalRequests.Load(),
		FailedRequests: m.failedRequests.Load(),
		AvgDuration:    avg,
		MinDuration:    min,
		MaxDuration:    max,
		HealthChecks:   m.healthChecks.Load(),
		ListRecords:    m.listRecords.Load(),
		GetRecord:      m.getRecord.Load(),
		VerifyChain:    m.verifyChain.Load(),
		Snapshots:      m.snapshots.Load(),
	}
}

// MetricsStats holds metrics statistics
type MetricsStats struct {
	TotalRequests  uint64        `json:"total_requests"`
	FailedRequests uint64        `json:"failed_requests"`
	AvgDuration    time.Duration `json:"avg_duration_ns"`
	MinDuration    time.Duration `json:"min_duration_ns"`
	MaxDuration    time.Duration `json:"max_duration_ns"`
	HealthChecks   uint64        `json:"health_checks"`
	ListRecords    uint64        `json:"list_records"`
	GetRecord      uint64        `json:"get_record"`
	VerifyChain    uint64        `json:"verify_chain"`
	Snapshots      uint64        `json:"snapshots"`
}

// PrometheusMetrics exports metrics in Prometheus format
func (m *Metrics) PrometheusMetrics() string {
	stats := m.GetStats()
	return `# HELP stateledger_requests_total Total number of HTTP requests
# TYPE stateledger_requests_total counter
stateledger_requests_total ` + formatUint64(stats.TotalRequests) + `

# HELP stateledger_requests_failed Total number of failed HTTP requests
# TYPE stateledger_requests_failed counter
stateledger_requests_failed ` + formatUint64(stats.FailedRequests) + `

# HELP stateledger_request_duration_avg Average request duration in nanoseconds
# TYPE stateledger_request_duration_avg gauge
stateledger_request_duration_avg ` + formatDuration(stats.AvgDuration) + `

# HELP stateledger_health_checks_total Total number of health check requests
# TYPE stateledger_health_checks_total counter
stateledger_health_checks_total ` + formatUint64(stats.HealthChecks) + `

# HELP stateledger_list_records_total Total number of list records requests
# TYPE stateledger_list_records_total counter
stateledger_list_records_total ` + formatUint64(stats.ListRecords) + `

# HELP stateledger_get_record_total Total number of get record requests
# TYPE stateledger_get_record_total counter
stateledger_get_record_total ` + formatUint64(stats.GetRecord) + `

# HELP stateledger_verify_chain_total Total number of verify chain requests
# TYPE stateledger_verify_chain_total counter
stateledger_verify_chain_total ` + formatUint64(stats.VerifyChain) + `

# HELP stateledger_snapshots_total Total number of snapshot requests
# TYPE stateledger_snapshots_total counter
stateledger_snapshots_total ` + formatUint64(stats.Snapshots) + `
`
}

func formatUint64(v uint64) string {
	return string(rune(v + '0'))
}

func formatDuration(d time.Duration) string {
	return string(rune(d.Nanoseconds() + '0'))
}
