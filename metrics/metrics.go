package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics
var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// Database metrics
	dbOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "table", "status"},
	)

	dbOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"operation", "table"},
	)

	// Authentication metrics
	authAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"type", "status"},
	)

	// Error metrics
	errorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors by type and code",
		},
		[]string{"error_type", "error_code"},
	)

	// Application metrics
	activeUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users_current",
			Help: "Current number of active users",
		},
	)

	tasksTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tasks_total",
			Help: "Total number of tasks by status",
		},
		[]string{"status"},
	)

	// System metrics
	goVersion = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_version_info",
			Help: "Go version information",
		},
		[]string{"version"},
	)

	appInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_info",
			Help: "Application information",
		},
		[]string{"version", "commit", "build_date"},
	)
)

// RecordHTTPRequest records an HTTP request metric
func RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
}

// RecordDatabaseOperation records a database operation metric
func RecordDatabaseOperation(operation, table string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	dbOperationsTotal.WithLabelValues(operation, table, status).Inc()
	dbOperationDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(authType, status string) {
	authAttemptsTotal.WithLabelValues(authType, status).Inc()
}

// RecordError records an error occurrence
func RecordError(errorType, errorCode string) {
	errorsTotal.WithLabelValues(errorType, errorCode).Inc()
}

// SetActiveUsers sets the current number of active users
func SetActiveUsers(count float64) {
	activeUsers.Set(count)
}

// SetTasksCount sets the total number of tasks by status
func SetTasksCount(status string, count float64) {
	tasksTotal.WithLabelValues(status).Set(count)
}

// InitAppInfo initializes application information metrics
func InitAppInfo(version, commit, buildDate, goVersionStr string) {
	appInfo.WithLabelValues(version, commit, buildDate).Set(1)
	goVersion.WithLabelValues(goVersionStr).Set(1)
}

// GetRegistry returns the default Prometheus registry
func GetRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}
