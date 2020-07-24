package main

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	consulMonitorError = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hobson_consul_monitor_error_total",
			Help: "Count of errors seen when fetching service status",
		},
		[]string{"service"},
	)

	queryHandleDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "hobson_query_handle_duration",
			Help:    "Histogram for the handle duration of DNS queries",
			Buckets: prometheus.LinearBuckets(0.00001, 0.00001, 10),
		},
	)

	queryUnknownName = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "hobson_query_unknown_name_total",
			Help: "Count of queries received for which there is no registered record",
		},
	)

	recordServed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hobson_record_served_total",
			Help: "Count of queries served for a record",
		},
		[]string{"service"},
	)

	recordUpdateTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hobson_record_last_updated_timestamp",
			Help: "Timestamp of the most recent update to a record",
		},
		[]string{"service"},
	)
)

// MetricsHandler wraps a Prometheus metrics exporter
type MetricsHandler struct {
	http *http.Server
}

// NewMetricsHandler creates a MetricsHandler object and prepares a Prometheus
// metrics exposition handler
func NewMetricsHandler(bind string) *MetricsHandler {
	prom := promhttp.Handler()

	http.HandleFunc("/metrics", prom.ServeHTTP)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
		<head><title>Hobson Metrics</title></head>
		<body>
		<h1>Hobson Metrics</h1>
		<img src="https://upload.wikimedia.org/wikipedia/commons/9/9d/ThomasHobson.jpg"/>
		<p><a href="/metrics">Metrics</a></p>
		</body>
		</html>`))
	})

	httpServer := &http.Server{
		Addr: bind,
	}

	return &MetricsHandler{
		http: httpServer,
	}
}

// RegisterPrometheus registers the current global Prometheus metrics variables
// into the global Prometheus registry within hobson
func (m *MetricsHandler) RegisterPrometheus() {
	prometheus.MustRegister(
		consulMonitorError,
		queryHandleDuration,
		queryUnknownName,
		recordServed,
		recordUpdateTime,
	)
}

// ListenAndServe wraps the underlying net/http ListenAndServe call
func (m *MetricsHandler) ListenAndServe() error {
	return m.http.ListenAndServe()
}

// Shutdown wraps the underlying net/http Shutdown call
func (m *MetricsHandler) Shutdown(ctx context.Context) error {
	return m.http.Shutdown(ctx)
}
