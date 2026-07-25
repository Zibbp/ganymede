package http

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type MetricsService interface {
	GatherMetrics(context.Context) (*prometheus.Registry, error)
}

func (h *Handler) GatherMetrics(ctx context.Context) (*prometheus.Registry, error) {
	r, err := h.Service.MetricsService.GatherMetrics(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error gathering metrics")
		return nil, err
	}
	return r, nil
}
