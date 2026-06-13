package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ActiveProblems = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "k8dclusterlife_active_problems_total",
		Help: "Number of currently active cluster problems",
	}, []string{"cluster", "kind", "severity"})

	HealingAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "k8dclusterlife_healing_attempts_total",
		Help: "Total number of healing attempts started",
	}, []string{"cluster", "kind", "autonomy_mode"})

	HealingSuccess = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "k8dclusterlife_healing_success_total",
		Help: "Total number of successful healing attempts",
	}, []string{"cluster", "kind"})

	HealingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "k8dclusterlife_healing_duration_seconds",
		Help:    "Duration of healing sessions in seconds",
		Buckets: prometheus.ExponentialBuckets(5, 2, 8), // 5s … 640s
	}, []string{"cluster", "status"})
)

// Handler returns the Prometheus HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}
