package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type Metrics struct {
	DNSQueriesTotal    *prometheus.CounterVec
	DNSQueryDuration   *prometheus.HistogramVec
	APIRequestsTotal   *prometheus.CounterVec
	APIRequestDuration *prometheus.HistogramVec
	Registry           *prometheus.Registry
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	m := &Metrics{
		Registry: reg,

		DNSQueriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "geodns_dns_queries_total",
				Help: "Total number of DNS queries processed.",
			},
			[]string{"qtype", "geo_tag", "status"},
		),

		DNSQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "geodns_dns_query_duration_seconds",
				Help:    "DNS query processing duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"qtype", "geo_tag"},
		),

		APIRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "geodns_api_requests_total",
				Help: "Total number of API requests.",
			},
			[]string{"method", "path", "status_code"},
		),

		APIRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "geodns_api_request_duration_seconds",
				Help:    "API request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}

	reg.MustRegister(m.DNSQueriesTotal)
	reg.MustRegister(m.DNSQueryDuration)
	reg.MustRegister(m.APIRequestsTotal)
	reg.MustRegister(m.APIRequestDuration)

	return m
}
