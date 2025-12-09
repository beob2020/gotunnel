package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// ActiveConnections Connection metrics
	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gotunnel_active_connections",
		Help: "Number of active tunnel connections",
	})

	TotalConnections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gotunnel_connections_total",
		Help: "Total number of connections established",
	})

	ConnectionErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gotunnel_connection_errors_total",
		Help: "Total connection errors by type",
	}, []string{"error_type"})

	// BytesTransferred Traffic metrics
	BytesTransferred = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gotunnel_bytes_transferred_total",
		Help: "Total bytes transferred",
	}, []string{"direction"})

	// RequestDuration Request metrics
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gotunnel_request_duration_seconds",
		Help:    "Request duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "status"})

	// CertificateExpiry Certificate metrics
	CertificateExpiry = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gotunnel_certificate_expiry_timestamp",
		Help: "Certificate expiry timestamp",
	})

	// HealthStatus Health metrics
	HealthStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gotunnel_health_status",
		Help: "Health status (1 = healthy, 0 = unhealthy)",
	})
)

// RecordConnection records a new connection
func RecordConnection() {
	TotalConnections.Inc()
	ActiveConnections.Inc()
}

// RecordDisconnection records a disconnection
func RecordDisconnection() {
	ActiveConnections.Dec()
}

// RecordTraffic records bytes transferred
func RecordTraffic(direction string, bytes int64) {
	BytesTransferred.WithLabelValues(direction).Add(float64(bytes))
}

// RecordRequest records request metrics
func RecordRequest(method, status string, duration time.Duration) {
	RequestDuration.WithLabelValues(method, status).Observe(duration.Seconds())
}

// RecordConnectionError records connection errors
func RecordConnectionError(errorType string) {
	ConnectionErrors.WithLabelValues(errorType).Inc()
}

// SetHealthStatus sets the health status
func SetHealthStatus(healthy bool) {
	if healthy {
		HealthStatus.Set(1)
	} else {
		HealthStatus.Set(0)
	}
}

// SetCertificateExpiry sets certificate expiry timestamp
func SetCertificateExpiry(timestamp float64) {
	CertificateExpiry.Set(timestamp)
}

// MetricsHandler returns the Prometheus metrics handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
