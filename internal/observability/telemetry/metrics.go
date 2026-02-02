package telemetry

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ==================== Business Metrics ====================

	// ActiveChargingSessions tracks the number of active charging sessions
	ActiveChargingSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sigec_active_charging_sessions",
		Help: "Number of active charging sessions",
	})

	// EnergyDeliveredTotal tracks total energy delivered in kWh
	EnergyDeliveredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sigec_energy_delivered_kwh_total",
		Help: "Total energy delivered in kWh",
	})

	// RevenueTotal tracks total revenue
	RevenueTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_revenue_total",
		Help: "Total revenue by currency",
	}, []string{"currency"})

	// TransactionsTotal tracks total transactions by status
	TransactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_transactions_total",
		Help: "Total transactions by status",
	}, []string{"status"})

	// ChargingDuration tracks the duration of charging sessions
	ChargingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "sigec_charging_duration_seconds",
		Help:    "Duration of charging sessions in seconds",
		Buckets: []float64{60, 300, 600, 1800, 3600, 7200, 14400}, // 1min, 5min, 10min, 30min, 1h, 2h, 4h
	})

	// ==================== Voice Metrics ====================

	// VoiceCommandsTotal tracks voice commands by intent and status
	VoiceCommandsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_voice_commands_total",
		Help: "Total voice commands processed",
	}, []string{"intent", "status"})

	// VoiceLatency tracks voice processing latency
	VoiceLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "sigec_voice_latency_seconds",
		Help:    "Voice processing latency in seconds",
		Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
	})

	// ==================== OCPP Metrics ====================

	// OCPPMessagesTotal tracks OCPP messages by action and direction
	OCPPMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_ocpp_messages_total",
		Help: "Total OCPP messages",
	}, []string{"action", "direction"})

	// OCPPConnectionsActive tracks active OCPP connections
	OCPPConnectionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sigec_ocpp_connections_active",
		Help: "Number of active OCPP WebSocket connections",
	})

	// ==================== Device Metrics ====================

	// DevicesTotal tracks total devices by status
	DevicesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sigec_devices_total",
		Help: "Total devices by status",
	}, []string{"status"})

	// DeviceLastSeen tracks when devices were last seen
	DeviceLastSeen = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "sigec_device_last_seen_timestamp",
		Help: "Timestamp of last device heartbeat",
	}, []string{"device_id"})

	// ==================== Infrastructure Metrics ====================

	// HTTPRequestDuration tracks HTTP request duration
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sigec_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
	}, []string{"method", "path", "status"})

	// HTTPRequestsTotal tracks total HTTP requests
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_http_requests_total",
		Help: "Total HTTP requests",
	}, []string{"method", "path", "status"})

	// DatabaseLatency tracks database query latency
	DatabaseLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "sigec_database_latency_seconds",
		Help:    "Database query latency in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
	}, []string{"operation", "table"})

	// CacheHitsTotal tracks cache hits and misses
	CacheHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_cache_hits_total",
		Help: "Total cache hits and misses",
	}, []string{"result"}) // hit, miss

	// MessageQueueMessagesTotal tracks message queue messages
	MessageQueueMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_mq_messages_total",
		Help: "Total message queue messages",
	}, []string{"topic", "status"}) // status: published, consumed, failed
)

// RecordTransactionStarted increments metrics when a transaction starts
func RecordTransactionStarted() {
	ActiveChargingSessions.Inc()
	TransactionsTotal.WithLabelValues("started").Inc()
}

// RecordTransactionCompleted updates metrics when a transaction completes
func RecordTransactionCompleted(energyKWh float64, cost float64, currency string, durationSeconds float64) {
	ActiveChargingSessions.Dec()
	TransactionsTotal.WithLabelValues("completed").Inc()
	EnergyDeliveredTotal.Add(energyKWh)
	RevenueTotal.WithLabelValues(currency).Add(cost)
	ChargingDuration.Observe(durationSeconds)
}

// RecordVoiceCommand records a voice command metric
func RecordVoiceCommand(intent string, success bool, latencySeconds float64) {
	status := "success"
	if !success {
		status = "failure"
	}
	VoiceCommandsTotal.WithLabelValues(intent, status).Inc()
	VoiceLatency.Observe(latencySeconds)
}

// RecordOCPPMessage records an OCPP message metric
func RecordOCPPMessage(action string, inbound bool) {
	direction := "outbound"
	if inbound {
		direction = "inbound"
	}
	OCPPMessagesTotal.WithLabelValues(action, direction).Inc()
}

// RecordHTTPRequest records an HTTP request metric
func RecordHTTPRequest(method, path string, status int, durationSeconds float64) {
	statusStr := fmt.Sprintf("%d", status)
	HTTPRequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	HTTPRequestDuration.WithLabelValues(method, path, statusStr).Observe(durationSeconds)
}

// RecordCacheAccess records a cache access metric
func RecordCacheAccess(hit bool) {
	result := "miss"
	if hit {
		result = "hit"
	}
	CacheHitsTotal.WithLabelValues(result).Inc()
}

