package metrics

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
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

func (s *Service) gatherRiverJobMetrics() error {
	pendingJobsParams := river.NewJobListParams().States(rivertype.JobStatePending).First(10000)
	pendingJobs, err := s.riverClient.JobList(context.Background(), pendingJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalPendingJobs.Set(float64(len(pendingJobs.Jobs)))

	scheduledJobsParams := river.NewJobListParams().States(rivertype.JobStateScheduled).First(10000)
	scheduledJobs, err := s.riverClient.JobList(context.Background(), scheduledJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalScheduledJobs.Set(float64(len(scheduledJobs.Jobs)))

	availableJobsParams := river.NewJobListParams().States(rivertype.JobStateAvailable).First(10000)
	availableJobs, err := s.riverClient.JobList(context.Background(), availableJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalAvailableJobs.Set(float64(len(availableJobs.Jobs)))

	runningJobsParams := river.NewJobListParams().States(rivertype.JobStateRunning).First(10000)
	runningJobs, err := s.riverClient.JobList(context.Background(), runningJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalRunningJobs.Set(float64(len(runningJobs.Jobs)))

	retryableJobsParams := river.NewJobListParams().States(rivertype.JobStateRetryable).First(10000)
	retryableJobs, err := s.riverClient.JobList(context.Background(), retryableJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalRetryableJobs.Set(float64(len(retryableJobs.Jobs)))

	cancelledJobsParams := river.NewJobListParams().States(rivertype.JobStateCancelled).First(10000)
	cancelledJobs, err := s.riverClient.JobList(context.Background(), cancelledJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalCancelledJobs.Set(float64(len(cancelledJobs.Jobs)))

	discardedJobsParams := river.NewJobListParams().States(rivertype.JobStateDiscarded).First(10000)
	discardedJobs, err := s.riverClient.JobList(context.Background(), discardedJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalDiscardedJobs.Set(float64(len(discardedJobs.Jobs)))

	cancelledJobsParams = river.NewJobListParams().States(rivertype.JobStateCancelled).First(10000)
	cancelledJobs, err = s.riverClient.JobList(context.Background(), cancelledJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalCancelledJobs.Set(float64(len(cancelledJobs.Jobs)))

	completedJobsParams := river.NewJobListParams().States(rivertype.JobStateCompleted).First(10000)
	completedJobs, err := s.riverClient.JobList(context.Background(), completedJobsParams)
	if err != nil {
		return err
	}
	s.metrics.riverTotalCompletedJobs.Set(float64(len(completedJobs.Jobs)))

	return nil
}

func (s *Service) GatherMetrics() (*prometheus.Registry, error) {
	// Gather metric data
	// Total number of Vods
	vCount, err := s.Store.Client.Vod.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total vods")
		s.metrics.totalVods.Set(0)
	}
	s.metrics.totalVods.Set(float64(vCount))
	// Total number of Channels
	cCount, err := s.Store.Client.Channel.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total channels")
		s.metrics.totalChannels.Set(0)
	}
	s.metrics.totalChannels.Set(float64(cCount))
	// Total number of Users
	uCount, err := s.Store.Client.User.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total users")
		s.metrics.totalUsers.Set(0)
	}
	s.metrics.totalUsers.Set(float64(uCount))
	// Total number of Live Watched Channels
	lwCount, err := s.Store.Client.Live.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total live watched channels")
		s.metrics.totalLiveWatchedChannels.Set(0)
	}
	s.metrics.totalLiveWatchedChannels.Set(float64(lwCount))
	// Get all channels and the number of VODs they have
	channels, err := s.Store.Client.Channel.Query().WithVods().All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting all channels")
		return nil, err
	}
	for _, channel := range channels {
		cVCount := len(channel.Edges.Vods)
		s.metrics.channelVodCount.With(prometheus.Labels{"channel": channel.Name}).Set(float64(cVCount))
	}
	// Total VODs in queue
	qCount, err := s.Store.Client.Queue.Query().Where(queue.Processing(true)).Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total vods in queue")
		s.metrics.totalVodsInQueue.Set(0)
	}
	s.metrics.totalVodsInQueue.Set(float64(qCount))

	// gather River job metrics
	err = s.gatherRiverJobMetrics()
	if err != nil {
		log.Error().Err(err).Msg("error gathering river job metrics")
		return nil, err
	}

	return s.Registry, nil
}
