package analytics

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entViewAnalytics "github.com/zibbp/ganymede/ent/viewanalytics"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/cache"
	"github.com/zibbp/ganymede/internal/database"
)

type Service struct {
	Store *database.Database
	Cache *cache.RedisCache
}

func NewService(store *database.Database, redisCache *cache.RedisCache) *Service {
	return &Service{
		Store: store,
		Cache: redisCache,
	}
}

type ViewAnalyticsData struct {
	UserID         uuid.UUID `json:"user_id"`
	VodID          uuid.UUID `json:"vod_id"`
	ViewDuration   int       `json:"view_duration"`
	MaxWatchedTime int       `json:"max_watched_time"`
	Completed      bool      `json:"completed"`
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	Referrer       string    `json:"referrer,omitempty"`
}

// RecordView records a new view analytics entry
func (s *Service) RecordView(ctx context.Context, data ViewAnalyticsData) error {
	// Check if user and video exist
	_, err := s.Store.Client.User.Get(ctx, data.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %v", err)
	}

	_, err = s.Store.Client.Vod.Get(ctx, data.VodID)
	if err != nil {
		return fmt.Errorf("video not found: %v", err)
	}

	// Create ViewAnalytics entry
	_, err = s.Store.Client.ViewAnalytics.Create().
		SetUserID(data.UserID).
		SetVodID(data.VodID).
		SetViewDuration(data.ViewDuration).
		SetMaxWatchedTime(data.MaxWatchedTime).
		SetCompleted(data.Completed).
		SetNillableIPAddress(&data.IPAddress).
		SetNillableUserAgent(&data.UserAgent).
		SetNillableReferrer(&data.Referrer).
		Save(ctx)

	if err != nil {
		return fmt.Errorf("failed to record view analytics: %v", err)
	}

	// Also update the video's local_views count for backward compatibility
	_, err = s.Store.Client.Vod.UpdateOneID(data.VodID).
		AddLocalViews(1).
		Save(ctx)

	return err
}

type VideoAnalytics struct {
	TotalViews       int     `json:"total_views"`
	UniqueViews      int     `json:"unique_views"`
	AverageWatchTime float64 `json:"average_watch_time"`
	CompletionRate   float64 `json:"completion_rate"`
	TotalWatchTime   int     `json:"total_watch_time"`
	ViewsToday       int     `json:"views_today"`
	ViewsThisWeek    int     `json:"views_this_week"`
	ViewsThisMonth   int     `json:"views_this_month"`
}

// GetVideoAnalytics returns basic analytics for a specific video
func (s *Service) GetVideoAnalytics(ctx context.Context, vodID uuid.UUID) (*VideoAnalytics, error) {
	// Verify video exists
	_, err := s.Store.Client.Vod.Get(ctx, vodID)
	if err != nil {
		return nil, fmt.Errorf("error fetching video: %v", err)
	}

	// Get analytics data from ViewAnalytics table
	analytics, err := s.Store.Client.ViewAnalytics.Query().
		Where(entViewAnalytics.VodID(vodID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching video analytics: %v", err)
	}

	// Calculate analytics from ViewAnalytics data
	totalViews := len(analytics)
	uniqueViews := 0
	var totalWatchTime int
	var completedViews int
	var totalViewDuration int

	// Track unique users
	uniqueUsers := make(map[uuid.UUID]bool)

	for _, view := range analytics {
		uniqueUsers[view.UserID] = true
		totalWatchTime += view.ViewDuration
		totalViewDuration += view.ViewDuration
		if view.Completed {
			completedViews++
		}
	}
	uniqueViews = len(uniqueUsers)

	// Calculate time-based views
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, 0, -30)

	viewsToday := 0
	viewsThisWeek := 0
	viewsThisMonth := 0

	for _, view := range analytics {
		if view.ViewedAt.After(today) {
			viewsToday++
		}
		if view.ViewedAt.After(weekAgo) {
			viewsThisWeek++
		}
		if view.ViewedAt.After(monthAgo) {
			viewsThisMonth++
		}
	}

	// Calculate averages
	var averageWatchTime float64
	var completionRate float64

	if totalViews > 0 {
		averageWatchTime = float64(totalViewDuration) / float64(totalViews)
		completionRate = float64(completedViews) / float64(totalViews) * 100
	}

	return &VideoAnalytics{
		TotalViews:       totalViews,
		UniqueViews:      uniqueViews,
		AverageWatchTime: averageWatchTime,
		CompletionRate:   completionRate,
		TotalWatchTime:   totalWatchTime,
		ViewsToday:       viewsToday,
		ViewsThisWeek:    viewsThisWeek,
		ViewsThisMonth:   viewsThisMonth,
	}, nil
}

type PopularVideo struct {
	Vod struct {
		ID            string `json:"id"`
		Title         string `json:"title"`
		LocalViews    int    `json:"local_views"`
		Duration      int    `json:"duration"`
		ThumbnailPath string `json:"thumbnail_path"`
		ChannelName   string `json:"channel_name"`
	} `json:"vod"`
	TotalViews int `json:"total_views"`
}

// GetPopularVideos returns the most viewed videos with optional caching
func (s *Service) GetPopularVideos(ctx context.Context, limit int, useCache bool) ([]*PopularVideo, error) {
	// If caching is enabled, try to retrieve from cache first
	if useCache && s.Cache != nil {
		cacheKey := fmt.Sprintf("popular_videos:%d", limit)
		var cachedVideos []*PopularVideo
		err := s.Cache.Get(ctx, cacheKey, &cachedVideos)
		if err == nil {
			return cachedVideos, nil
		}
	}

	// Use a single query with sorting and limit
	videos, err := s.Store.Client.Vod.Query().
		WithChannel().
		Order(entVod.ByLocalViews(sql.OrderDesc())).
		Limit(limit).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("error fetching popular videos: %v", err)
	}

	var popularVideos []*PopularVideo
	for _, video := range videos {
		// Safely access channel name
		var channelName string
		if video.Edges.Channel != nil {
			channelName = video.Edges.Channel.Name
		}

		popularVideos = append(popularVideos, &PopularVideo{
			Vod: struct {
				ID            string `json:"id"`
				Title         string `json:"title"`
				LocalViews    int    `json:"local_views"`
				Duration      int    `json:"duration"`
				ThumbnailPath string `json:"thumbnail_path"`
				ChannelName   string `json:"channel_name"`
			}{
				ID:            video.ID.String(),
				Title:         video.Title,
				LocalViews:    video.LocalViews,
				Duration:      video.Duration,
				ThumbnailPath: video.ThumbnailPath,
				ChannelName:   channelName,
			},
			TotalViews: video.LocalViews, // Use local_views as total views
		})
	}

	// If caching is enabled, store the result in cache
	if useCache && s.Cache != nil {
		cacheKey := fmt.Sprintf("popular_videos:%d", limit)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := s.Cache.Set(ctx, cacheKey, popularVideos, 10*time.Minute)
			if err != nil {
				log.Error().Err(err).Msg("Failed to cache popular videos")
			}
		}()
	}

	return popularVideos, nil
}

// GetChannelAnalytics returns analytics for all videos in a channel
func (s *Service) GetChannelAnalytics(ctx context.Context, channelID uuid.UUID) (*ChannelAnalytics, error) {
	videos, err := s.Store.Client.Vod.Query().
		Where(entVod.HasChannelWith(entChannel.ID(channelID))).
		All(ctx)

	if err != nil {
		return nil, fmt.Errorf("error fetching channel videos: %v", err)
	}

	var totalViews int
	var totalDuration int
	var totalWatchTime int

	for _, video := range videos {
		totalViews += video.LocalViews
		totalDuration += video.Duration
		totalWatchTime += video.Duration * video.LocalViews
	}

	var averageViews float64
	var averageWatchTime float64

	if len(videos) > 0 {
		averageViews = float64(totalViews) / float64(len(videos))
		averageWatchTime = float64(totalWatchTime) / float64(totalViews)
	}

	return &ChannelAnalytics{
		TotalVideos:      len(videos),
		TotalViews:       totalViews,
		AverageViews:     averageViews,
		TotalDuration:    totalDuration,
		AverageWatchTime: averageWatchTime,
	}, nil
}

type ChannelAnalytics struct {
	TotalVideos      int     `json:"total_videos"`
	TotalViews       int     `json:"total_views"`
	AverageViews     float64 `json:"average_views"`
	TotalDuration    int     `json:"total_duration"`
	AverageWatchTime float64 `json:"average_watch_time"`
}

// PageViewData represents the data for tracking a page view
type PageViewData struct {
	Page      string `json:"page"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Referrer  string `json:"referrer,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// SiteAnalytics represents site-wide analytics data
type SiteAnalytics struct {
	TotalVisitors    int     `json:"total_visitors"`
	TodayVisitors    int     `json:"today_visitors"`
	WeeklyVisitors   int     `json:"weekly_visitors"`
	MonthlyVisitors  int     `json:"monthly_visitors"`
	MostViewedPage   string  `json:"most_viewed_page"`
	MostViewedVideo  string  `json:"most_viewed_video"`
	AveragePageViews float64 `json:"average_page_views"`
	BounceRate       float64 `json:"bounce_rate"`
}

// DailyVisitorData represents daily visitor counts
type DailyVisitorData struct {
	Date     string `json:"date"`
	Visitors int    `json:"visitors"`
}

// PopularPageData represents popular page data
type PopularPageData struct {
	Page  string `json:"page"`
	Views int    `json:"views"`
}

// RecordPageView records a new page view (placeholder for now)
func (s *Service) RecordPageView(ctx context.Context, data PageViewData) error {
	// TODO: Implement when PageView entity is available
	// For now, just log the page view
	log.Info().
		Str("page", data.Page).
		Str("ip", data.IPAddress).
		Str("user_agent", data.UserAgent).
		Msg("Page view recorded")
	return nil
}

// GetSiteAnalytics returns comprehensive site analytics (placeholder for now)
func (s *Service) GetSiteAnalytics(ctx context.Context) (*SiteAnalytics, error) {
	// TODO: Implement when PageView entity is available
	// For now, return placeholder data based on video views

	// Get most viewed video
	mostViewedVideo, err := s.Store.Client.Vod.Query().
		Order(entVod.ByLocalViews(sql.OrderDesc())).
		Limit(1).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting most viewed video: %v", err)
	}

	var videoTitle string
	if len(mostViewedVideo) > 0 {
		videoTitle = mostViewedVideo[0].Title
	}

	// Calculate total views across all videos
	var totalViews []int
	err = s.Store.Client.Vod.Query().
		Aggregate(ent.Sum(entVod.FieldLocalViews)).
		Scan(ctx, &totalViews)
	if err != nil {
		return nil, fmt.Errorf("error calculating total views: %v", err)
	}

	// Placeholder values for now
	return &SiteAnalytics{
		TotalVisitors:    totalViews[0] / 10, // Rough estimate
		TodayVisitors:    totalViews[0] / 100,
		WeeklyVisitors:   totalViews[0] / 20,
		MonthlyVisitors:  totalViews[0] / 5,
		MostViewedPage:   "/videos",
		MostViewedVideo:  videoTitle,
		AveragePageViews: 2.5,
		BounceRate:       35.0,
	}, nil
}

// GetDailyVisitors returns visitor counts for the last 7 days (placeholder)
func (s *Service) GetDailyVisitors(ctx context.Context, days int) ([]DailyVisitorData, error) {
	// TODO: Implement when PageView entity is available
	// For now, return placeholder data
	var results []DailyVisitorData
	now := time.Now()

	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		results = append(results, DailyVisitorData{
			Date:     date.Format("2006-01-02"),
			Visitors: 10 + (i * 2), // Placeholder data
		})
	}

	return results, nil
}

// GetPopularPages returns the most viewed pages (placeholder)
func (s *Service) GetPopularPages(ctx context.Context, limit int) ([]PopularPageData, error) {
	// TODO: Implement when PageView entity is available
	// For now, return placeholder data
	return []PopularPageData{
		{Page: "/videos", Views: 150},
		{Page: "/admin", Views: 50},
		{Page: "/", Views: 30},
	}, nil
}
