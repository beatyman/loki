package http

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics tracks Prometheus metrics for the HTTP target.
type Metrics struct {
	scrapeEntries   *prometheus.CounterVec
	scrapeErrors    *prometheus.CounterVec
	requestCounter  *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewMetrics creates and registers Prometheus metrics for the HTTP target.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	var m Metrics

	m.scrapeEntries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "promtail",
		Name:      "http_target_entries_total",
		Help:      "Number of successful log entries scraped by the HTTP target",
	}, []string{"url"})

	m.scrapeErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "promtail",
		Name:      "http_target_scrape_errors_total",
		Help:      "Number of errors while scraping HTTP endpoint",
	}, []string{"url"})

	m.requestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "promtail",
		Name:      "http_target.requests_total",
		Help:      "Total number of HTTP requests made by the HTTP target",
	}, []string{"url", "status"})

	m.requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "promtail",
		Name:      "http_target_request_duration_seconds",
		Help:      "Duration of HTTP requests made by the HTTP target",
	}, []string{"url"})

	reg.MustRegister(m.scrapeEntries, m.scrapeErrors, m.requestCounter, m.requestDuration)
	return &m
}
