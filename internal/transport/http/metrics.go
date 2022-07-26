package http

import "github.com/prometheus/client_golang/prometheus"

type MetricsService interface {
	GatherMetrics() *prometheus.Registry
}

func (h *Handler) GatherMetrics() *prometheus.Registry {
	return h.Service.MetricsService.GatherMetrics()
}
