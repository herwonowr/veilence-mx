package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Namespace is the Prometheus metric namespace for all Veilence-MX metrics.
const Namespace = "veilence_mx"

// Application-level metrics. Counters and histograms for key operations.
var (
	// HTTPRequestsTotal counts total HTTP requests by method, path, and status code.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests by method, path pattern, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks HTTP request latency.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// QueueJobsEnqueued counts total jobs enqueued by type.
	QueueJobsEnqueued = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "queue_jobs_enqueued_total",
			Help:      "Total number of jobs enqueued by job type.",
		},
		[]string{"job_type"},
	)

	// QueueJobsProcessed counts total jobs processed by type and result.
	QueueJobsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "queue_jobs_processed_total",
			Help:      "Total number of jobs processed by type and result (success/error).",
		},
		[]string{"job_type", "result"},
	)

	// AnalysisClassifications counts analysis results by classification.
	AnalysisClassifications = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "analysis_classifications_total",
			Help:      "Total analysis results by classification (benign, suspicious, malicious).",
		},
		[]string{"classification"},
	)

	// AlertsCreated counts alerts created by severity.
	AlertsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "alerts_created_total",
			Help:      "Total alerts created by severity.",
		},
		[]string{"severity"},
	)

	// NotificationsSent counts notifications dispatched by channel type.
	NotificationsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "notifications_sent_total",
			Help:      "Total notifications dispatched by channel type (email, slack, webhook).",
		},
		[]string{"channel_type"},
	)

	// AuthOperations counts auth operations by type and result.
	AuthOperations = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "auth_operations_total",
			Help:      "Total auth operations by type (login, register, refresh) and result (success/failure).",
		},
		[]string{"operation", "result"},
	)
)

// Registry is the custom Prometheus registry for Veilence-MX.
// Using a custom registry avoids polluting the default registry and gives
// explicit control over which metrics are exported.
var Registry *prometheus.Registry

func init() {
	Registry = prometheus.NewRegistry()

	// Register standard Go runtime and process collectors
	Registry.MustRegister(collectors.NewGoCollector())
	Registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Register application metrics
	Registry.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		QueueJobsEnqueued,
		QueueJobsProcessed,
		AnalysisClassifications,
		AlertsCreated,
		NotificationsSent,
		AuthOperations,
	)
}
