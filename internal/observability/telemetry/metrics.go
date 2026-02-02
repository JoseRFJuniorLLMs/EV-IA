package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Métricas de negócio
	ActiveChargingSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sigec_active_charging_sessions",
		Help: "Número de sessões de carregamento ativas",
	})

	EnergyDeliveredTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sigec_energy_delivered_kwh_total",
		Help: "Total de energia entregue em kWh",
	})

	VoiceCommandsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_voice_commands_total",
		Help: "Total de comandos de voz processados",
	}, []string{"intent", "status"})

	VoiceLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "sigec_voice_latency_seconds",
		Help:    "Latência de processamento de voz",
		Buckets: prometheus.DefBuckets,
	})

	// Métricas de infraestrutura
	OCPPMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "sigec_ocpp_messages_total",
		Help: "Total de mensagens OCPP",
	}, []string{"action", "direction"})

	DatabaseLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "sigec_database_latency_seconds",
		Help:    "Latência de queries no banco",
		Buckets: prometheus.DefBuckets,
	})
)
