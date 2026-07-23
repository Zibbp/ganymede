package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/database"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
)

type Service struct {
	Store       *database.Database
	riverClient *tasks_client.RiverClient
	metrics     *Metrics
	Registry    *prometheus.Registry
}

type Metrics struct {
	totalVods                prometheus.Gauge
	totalChannels            prometheus.Gauge
	totalUsers               prometheus.Gauge
	totalLiveWatchedChannels prometheus.Gauge
	channelVodCount          *prometheus.GaugeVec
	totalVodsDuration        prometheus.Gauge
	channelVodDuration       *prometheus.GaugeVec
	totalVodsBytes           prometheus.Gauge
	channelVodBytes          *prometheus.GaugeVec
	totalVodsInQueue         prometheus.Gauge
	riverTotalPendingJobs    prometheus.Gauge
	riverTotalScheduledJobs  prometheus.Gauge
	riverTotalAvailableJobs  prometheus.Gauge
	riverTotalRunningJobs    prometheus.Gauge
	riverTotalRetryableJobs  prometheus.Gauge
	riverTotalCancelledJobs  prometheus.Gauge
	riverTotalDiscardedJobs  prometheus.Gauge
	riverTotalCompletedJobs  prometheus.Gauge
}

func NewService(store *database.Database, riverClient *tasks_client.RiverClient) *Service {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		totalVods: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "total_vods",
			Help: "Total number of vods",
		}),
		totalChannels: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "total_channels",
			Help: "Total number of channels",
		}),
		totalUsers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "total_users",
			Help: "Total number of users",
		}),
		totalLiveWatchedChannels: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "total_live_watched_channels",
			Help: "Total number of live watched channels",
		}),
		channelVodCount: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "channel_vod_count",
			Help: "Number of vods per channel",
		}, []string{"channel"}),
		totalVodsDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "total_vods_duration_seconds",
			Help: "Total duration of all VODs in seconds",
		}),
		channelVodDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "channel_vod_duration_seconds",
			Help: "Total duration of VODs per channel in seconds",
		}, []string{"channel"}),
		totalVodsBytes: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "total_vods_bytes",
			Help: "Total size of all VODs in bytes",
		}),
		channelVodBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "channel_vod_bytes",
			Help: "Total size of VODs per channel in bytes",
		}, []string{"channel"}),
		totalVodsInQueue: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "total_vods_in_queue",
			Help: "Total number of vods in queue",
		}),
		riverTotalPendingJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_pending_jobs",
			Help: "Total number of pending jobs",
		}),
		riverTotalScheduledJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_scheduled_jobs",
			Help: "Total number of scheduled jobs",
		}),
		riverTotalAvailableJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_available_jobs",
			Help: "Total number of available jobs",
		}),
		riverTotalRunningJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_running_jobs",
			Help: "Total number of running jobs",
		}),
		riverTotalRetryableJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_retryable_jobs",
			Help: "Total number of retryable jobs",
		}),
		riverTotalCancelledJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_cancelled_jobs",
			Help: "Total number of cancelled jobs",
		}),
		riverTotalDiscardedJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_discarded_jobs",
			Help: "Total number of discarded jobs",
		}),
		riverTotalCompletedJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "river_total_completed_jobs",
			Help: "Total number of completed jobs",
		}),
	}

	registry.MustRegister(
		metrics.totalVods,
		metrics.totalChannels,
		metrics.totalUsers,
		metrics.totalLiveWatchedChannels,
		metrics.channelVodCount,
		metrics.totalVodsDuration,
		metrics.channelVodDuration,
		metrics.totalVodsBytes,
		metrics.channelVodBytes,
		metrics.totalVodsInQueue,
		metrics.riverTotalPendingJobs,
		metrics.riverTotalScheduledJobs,
		metrics.riverTotalAvailableJobs,
		metrics.riverTotalRunningJobs,
		metrics.riverTotalRetryableJobs,
		metrics.riverTotalCancelledJobs,
		metrics.riverTotalDiscardedJobs,
		metrics.riverTotalCompletedJobs,
	)

	return &Service{Store: store, riverClient: riverClient, metrics: metrics, Registry: registry}
}

func (s *Service) gatherRiverJobMetrics(ctx context.Context) error {
	gauges := map[string]prometheus.Gauge{
		"pending":   s.metrics.riverTotalPendingJobs,
		"scheduled": s.metrics.riverTotalScheduledJobs,
		"available": s.metrics.riverTotalAvailableJobs,
		"running":   s.metrics.riverTotalRunningJobs,
		"retryable": s.metrics.riverTotalRetryableJobs,
		"cancelled": s.metrics.riverTotalCancelledJobs,
		"discarded": s.metrics.riverTotalDiscardedJobs,
		"completed": s.metrics.riverTotalCompletedJobs,
	}
	for _, gauge := range gauges {
		gauge.Set(0)
	}

	rows, err := s.Store.SQLDB.QueryContext(ctx, `SELECT state, COUNT(*) FROM river_job GROUP BY state`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		var count int64
		if err := rows.Scan(&state, &count); err != nil {
			return err
		}
		if gauge, ok := gauges[state]; ok {
			gauge.Set(float64(count))
		}
	}
	return rows.Err()
}

func (s *Service) GatherMetrics(ctx context.Context) (*prometheus.Registry, error) {
	// Gather metric data
	// Total number of Vods
	vCount, err := s.Store.Client.Vod.Query().Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting total vods")
		s.metrics.totalVods.Set(0)
	}
	s.metrics.totalVods.Set(float64(vCount))
	// Total number of Channels
	cCount, err := s.Store.Client.Channel.Query().Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting total channels")
		s.metrics.totalChannels.Set(0)
	}
	s.metrics.totalChannels.Set(float64(cCount))
	// Total number of Users
	uCount, err := s.Store.Client.User.Query().Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting total users")
		s.metrics.totalUsers.Set(0)
	}
	s.metrics.totalUsers.Set(float64(uCount))
	// Total number of Live Watched Channels
	lwCount, err := s.Store.Client.Live.Query().Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting total live watched channels")
		s.metrics.totalLiveWatchedChannels.Set(0)
	}
	s.metrics.totalLiveWatchedChannels.Set(float64(lwCount))
	// Get all channels with the number of VODs they have, their total duration and their size
	var totalDurationSeconds int64 = 0
	var totalBytes int64 = 0
	channels, err := s.Store.Client.Channel.Query().WithVods().All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting all channels")
		return nil, err
	}
	for _, channel := range channels {
		cVCount := len(channel.Edges.Vods)
		s.metrics.channelVodCount.With(prometheus.Labels{"channel": channel.Name}).Set(float64(cVCount))
		var channelDuration int64 = 0
		var channelBytes int64 = 0
		for _, vod := range channel.Edges.Vods {
			channelDuration += int64(vod.Duration)
			channelBytes += vod.StorageSizeBytes
		}
		s.metrics.channelVodDuration.With(prometheus.Labels{"channel": channel.Name}).Set(float64(channelDuration))
		s.metrics.channelVodBytes.With(prometheus.Labels{"channel": channel.Name}).Set(float64(channelBytes))
		totalDurationSeconds += channelDuration
		totalBytes += channelBytes
	}
	s.metrics.totalVodsDuration.Set(float64(totalDurationSeconds))
	s.metrics.totalVodsBytes.Set(float64(totalBytes))
	// Total VODs in queue
	qCount, err := s.Store.Client.Queue.Query().Where(queue.Processing(true)).Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error getting total vods in queue")
		s.metrics.totalVodsInQueue.Set(0)
	}
	s.metrics.totalVodsInQueue.Set(float64(qCount))

	// gather River job metrics
	err = s.gatherRiverJobMetrics(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error gathering river job metrics")
		return nil, err
	}

	return s.Registry, nil
}
