package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/igorsal/pr-documentator/internal/interfaces"
)

// PrometheusCollector implements the MetricsCollector interface using Prometheus
type PrometheusCollector struct {
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
	gauges     map[string]*prometheus.GaugeVec
}

// NewPrometheusCollector creates a new Prometheus metrics collector
func NewPrometheusCollector() interfaces.MetricsCollector {
	collector := &PrometheusCollector{
		counters:   make(map[string]*prometheus.CounterVec),
		histograms: make(map[string]*prometheus.HistogramVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
	}

	// Initialize common metrics
	collector.initializeMetrics()

	return collector
}

func (p *PrometheusCollector) initializeMetrics() {
	// HTTP request metrics
	p.counters["http_requests_total"] = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_documentator_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	p.histograms["http_request_duration_seconds"] = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pr_documentator_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	// Claude API metrics
	p.counters["claude_requests_total"] = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_documentator_claude_requests_total",
			Help: "Total number of Claude API requests",
		},
		[]string{"service", "operation", "status", "repository"},
	)

	p.histograms["claude_request_duration_seconds"] = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pr_documentator_claude_request_duration_seconds",
			Help:    "Claude API request duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		},
		[]string{"service", "operation", "repository"},
	)

	// Postman API metrics
	p.counters["postman_requests_total"] = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_documentator_postman_requests_total",
			Help: "Total number of Postman API requests",
		},
		[]string{"service", "operation", "status"},
	)

	p.histograms["postman_request_duration_seconds"] = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pr_documentator_postman_request_duration_seconds",
			Help:    "Postman API request duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"service", "operation"},
	)

	// Business metrics
	p.counters["pr_analysis_total"] = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_documentator_pr_analysis_total",
			Help: "Total number of PR analyses performed",
		},
		[]string{"repository", "action", "status"},
	)

	p.histograms["pr_analysis_duration_seconds"] = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pr_documentator_pr_analysis_duration_seconds",
			Help:    "PR analysis duration in seconds",
			Buckets: []float64{1.0, 5.0, 10.0, 30.0, 60.0, 120.0},
		},
		[]string{"repository", "action"},
	)

	p.gauges["api_routes_discovered"] = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pr_documentator_api_routes_discovered",
			Help: "Number of API routes discovered in PR analysis",
		},
		[]string{"repository", "type"}, // type: new, modified, deleted
	)

	// Circuit breaker metrics
	p.gauges["circuit_breaker_state"] = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pr_documentator_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"service", "name"},
	)

	p.counters["circuit_breaker_events_total"] = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_documentator_circuit_breaker_events_total",
			Help: "Total circuit breaker events",
		},
		[]string{"service", "name", "event"}, // event: success, failure, timeout, rejection
	)
}

// IncrementCounter increments a counter metric
func (p *PrometheusCollector) IncrementCounter(name string, labels map[string]string) {
	counter, exists := p.counters[name]
	if !exists {
		return
	}

	counter.With(labels).Inc()
}

// RecordDuration records a duration in a histogram
func (p *PrometheusCollector) RecordDuration(name string, duration float64, labels map[string]string) {
	histogram, exists := p.histograms[name]
	if !exists {
		return
	}

	histogram.With(labels).Observe(duration)
}

// SetGauge sets a gauge value
func (p *PrometheusCollector) SetGauge(name string, value float64, labels map[string]string) {
	gauge, exists := p.gauges[name]
	if !exists {
		return
	}

	gauge.With(labels).Set(value)
}

// RegisterCustomCounter registers a new counter metric
func (p *PrometheusCollector) RegisterCustomCounter(name, help string, labels []string) {
	if _, exists := p.counters[name]; exists {
		return
	}

	p.counters[name] = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pr_documentator_" + name,
			Help: help,
		},
		labels,
	)
}

// RegisterCustomHistogram registers a new histogram metric
func (p *PrometheusCollector) RegisterCustomHistogram(name, help string, labels []string, buckets []float64) {
	if _, exists := p.histograms[name]; exists {
		return
	}

	if buckets == nil {
		buckets = prometheus.DefBuckets
	}

	p.histograms[name] = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pr_documentator_" + name,
			Help:    help,
			Buckets: buckets,
		},
		labels,
	)
}

// RegisterCustomGauge registers a new gauge metric
func (p *PrometheusCollector) RegisterCustomGauge(name, help string, labels []string) {
	if _, exists := p.gauges[name]; exists {
		return
	}

	p.gauges[name] = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pr_documentator_" + name,
			Help: help,
		},
		labels,
	)
}
