package metrics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

// Define metrics
var (
	totalVods = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_vods",
		Help: "Total number of vods",
	})
	totalChannels = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_channels",
		Help: "Total number of channels",
	})
	totalUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_users",
		Help: "Total number of users",
	})
	totalLiveWatchedChannels = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_live_watched_channels",
		Help: "Total number of live watched channels",
	})
	channelVodCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "channel_vod_count",
		Help: "Number of vods per channel",
	}, []string{"channel"})
	totalVodsInQueue = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "total_vods_in_queue",
		Help: "Total number of vods in queue",
	})
)

func (s *Service) GatherMetrics() *prometheus.Registry {
	// Gather metric data
	// Total number of Vods
	vCount, err := s.Store.Client.Vod.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total vods")
		totalVods.Set(0)
	}
	// Total number of Channels
	cCount, err := s.Store.Client.Channel.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total channels")
		totalChannels.Set(0)
	}
	// Total number of Users
	uCount, err := s.Store.Client.User.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total users")
		totalUsers.Set(0)
	}
	// Total number of Live Watched Channels
	lwCount, err := s.Store.Client.Live.Query().Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total live watched channels")
		totalLiveWatchedChannels.Set(0)
	}
	// Get all channels and the number of VODs they have
	channels, err := s.Store.Client.Channel.Query().WithVods().All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting all channels")
		return nil
	}
	for _, channel := range channels {
		cVCount := len(channel.Edges.Vods)
		channelVodCount.With(prometheus.Labels{"channel": channel.Name}).Set(float64(cVCount))

	}
	// Total VODs in queue
	qCount, err := s.Store.Client.Queue.Query().Where(queue.Processing(true)).Count(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting total vods in queue")
		totalVodsInQueue.Set(0)
	}

	// Set metric data
	totalVods.Set(float64(vCount))
	totalChannels.Set(float64(cCount))
	totalUsers.Set(float64(uCount))
	totalLiveWatchedChannels.Set(float64(lwCount))
	totalVodsInQueue.Set(float64(qCount))

	// Create registry
	r := prometheus.NewRegistry()
	r.MustRegister(totalVods)
	r.MustRegister(totalChannels)
	r.MustRegister(totalUsers)
	r.MustRegister(totalLiveWatchedChannels)
	r.MustRegister(channelVodCount)
	r.MustRegister(totalVodsInQueue)
	return r
}
