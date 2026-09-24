package admin

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zibbp/ganymede/ent"
	entqueue "github.com/zibbp/ganymede/ent/queue"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/utils"
)

type GetVideoStatisticsResponse struct {
	VideoCount           int            `json:"video_count"`
	ChannelCount         int            `json:"channel_count"`
	ChannelVideos        map[string]int `json:"channel_videos"`
	VideoTypes           map[string]int `json:"video_types"`
	TotalDurationSeconds int64          `json:"total_duration_seconds"`
	TotalViews           int64          `json:"total_views"`
	TotalLocalViews      int64          `json:"total_local_views"`
	TotalStorageBytes    int64          `json:"total_storage_bytes"`
}

type QueueOverview struct {
	Total         int `json:"total"`
	Processing    int `json:"processing"`
	OnHold        int `json:"on_hold"`
	LiveArchiving int `json:"live_archiving"`
	Failed        int `json:"failed"`
}

type GetSystemOverviewResponse struct {
	VideosDirectoryFreeSpace  int64         `json:"videos_directory_free_space"`  // Free space in bytes
	VideosDirectoryUsedSpace  int64         `json:"videos_directory_used_space"`  // Tracked VOD storage in bytes
	VideosDirectoryTotalSpace int64         `json:"videos_directory_total_space"` // Filesystem capacity in bytes
	CPUCores                  int           `json:"cpu_cores"`                    // Number of CPU cores
	MemoryTotal               int64         `json:"memory_total"`                 // Total memory in bytes
	Queue                     QueueOverview `json:"queue"`
	UserCount                 int           `json:"user_count"`
}

type LargestVideoItem struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	ChannelName      string    `json:"channel_name"`
	StorageSizeBytes int64     `json:"storage_size_bytes"`
	Duration         int       `json:"duration"`
	StreamedAt       time.Time `json:"streamed_at"`
}

type GetStorageDistributionResponse struct {
	StorageDistribution map[string]int64   `json:"storage_distribution"` // Map of channel names to total storage used
	StorageByType       map[string]int64   `json:"storage_by_type"`      // Map of video types to total storage used in bytes
	LargestVideos       []LargestVideoItem `json:"largest_videos"`       // Top largest videos by storage size
}

// GetVideoStatistics retrieves statistics about videos in the system.
func (s *Service) GetVideoStatistics(ctx context.Context) (GetVideoStatisticsResponse, error) {
	var resp GetVideoStatisticsResponse

	// Get total video count
	vC, err := s.Store.Client.Vod.Query().Count(ctx)
	if err != nil {
		return resp, fmt.Errorf("error getting video count: %w", err)
	}

	// Get total channel count
	cC, err := s.Store.Client.Channel.Query().Count(ctx)
	if err != nil {
		return resp, fmt.Errorf("error getting channel count: %w", err)
	}

	// Group videos by channel ID
	vods, err := s.Store.Client.Vod.Query().
		WithChannel().
		All(ctx)
	if err != nil {
		return resp, fmt.Errorf("error getting videos with channel edge: %w", err)
	}

	channelVideosMap := make(map[string]int)
	var totalDuration, totalViews, totalLocalViews, totalStorage int64
	for _, vod := range vods {
		if vod.Edges.Channel != nil {
			channelVideosMap[vod.Edges.Channel.Name]++
		}
		totalDuration += int64(vod.Duration)
		totalViews += int64(vod.Views)
		totalLocalViews += int64(vod.LocalViews)
		totalStorage += vod.StorageSizeBytes
	}
	// Group videos by type
	var videoTypeStats []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	err = s.Store.Client.Vod.Query().
		GroupBy(entVod.FieldType).
		Aggregate(ent.Count()).
		Scan(ctx, &videoTypeStats)
	if err != nil {
		return resp, fmt.Errorf("error getting video types: %w", err)
	}

	videoTypesMap := make(map[string]int)
	for _, vt := range videoTypeStats {
		videoTypesMap[vt.Type] = vt.Count
	}

	// Final response
	resp.VideoCount = vC
	resp.ChannelCount = cC
	resp.ChannelVideos = channelVideosMap
	resp.VideoTypes = videoTypesMap
	resp.TotalDurationSeconds = totalDuration
	resp.TotalViews = totalViews
	resp.TotalLocalViews = totalLocalViews
	resp.TotalStorageBytes = totalStorage

	return resp, nil
}

// GetSystemOverview retrieves an overview of the system including free/used space, CPU cores, and memory.
func (s *Service) GetSystemOverview(ctx context.Context) (GetSystemOverviewResponse, error) {
	var resp GetSystemOverviewResponse
	env := config.GetEnvConfig()

	// Get data directory free space
	freeSpace, err := utils.GetFreeSpaceOfDirectory(env.VideosDir)
	if err != nil {
		return resp, fmt.Errorf("error getting data directory free space: %w", err)
	}
	resp.VideosDirectoryFreeSpace = freeSpace

	// Get data directory total capacity from the filesystem itself. Do not derive
	// this from free space plus tracked VOD bytes: the volume may contain files
	// outside of Ganymede's control (e.g. shared NFS/SMB mounts).
	totalSpace, err := utils.GetTotalSpaceOfDirectory(env.VideosDir)
	if err != nil {
		return resp, fmt.Errorf("error getting data directory total space: %w", err)
	}
	resp.VideosDirectoryTotalSpace = totalSpace

	// Get data directory used space by querying all vods and summing their storage sizes
	// Could check the directory size directly, but this information is already stored in the database
	type UsedSpaceResult struct {
		Sum sql.NullInt64
	}

	var result []UsedSpaceResult

	// TODO: improve this query to avoid loading all videos into memory
	err = s.Store.Client.Vod.Query().
		Aggregate(ent.Sum(entVod.FieldStorageSizeBytes)).
		Scan(ctx, &result)
	if err != nil {
		return resp, fmt.Errorf("error getting data directory used space: %w", err)
	}

	var storageSize int64
	if len(result) > 0 && result[0].Sum.Valid {
		storageSize = result[0].Sum.Int64
	} else {
		storageSize = 0
	}
	resp.VideosDirectoryUsedSpace = storageSize

	// Get CPU cores
	cpuCores := utils.GetCPUCores()
	resp.CPUCores = cpuCores

	// Get total memory
	totalMemory, err := utils.GetMemoryTotal()
	if err != nil {
		return resp, fmt.Errorf("error getting total memory: %w", err)
	}
	resp.MemoryTotal = totalMemory

	// Queue health - cheap count queries, failure of these should not fail the whole overview
	resp.Queue, _ = s.getQueueOverview(ctx)

	// User count
	if userCount, err := s.Store.Client.User.Query().Count(ctx); err == nil {
		resp.UserCount = userCount
	}

	return resp, nil
}

func (s *Service) getQueueOverview(ctx context.Context) (QueueOverview, error) {
	var q QueueOverview
	var err error

	if q.Total, err = s.Store.Client.Queue.Query().Count(ctx); err != nil {
		return q, err
	}
	if q.Processing, err = s.Store.Client.Queue.Query().Where(entqueue.Processing(true)).Count(ctx); err != nil {
		return q, err
	}
	if q.OnHold, err = s.Store.Client.Queue.Query().Where(entqueue.OnHold(true)).Count(ctx); err != nil {
		return q, err
	}
	// live_archive records how the item was created and stays true after it
	// finishes, so restrict to actively processing items to report the
	// number currently archiving rather than all historical live archives.
	if q.LiveArchiving, err = s.Store.Client.Queue.Query().Where(
		entqueue.And(
			entqueue.LiveArchive(true),
			entqueue.Processing(true),
		),
	).Count(ctx); err != nil {
		return q, err
	}
	failedPredicate := entqueue.Or(
		entqueue.TaskVodCreateFolderEQ(utils.Failed),
		entqueue.TaskVodDownloadThumbnailEQ(utils.Failed),
		entqueue.TaskVodSaveInfoEQ(utils.Failed),
		entqueue.TaskVideoDownloadEQ(utils.Failed),
		entqueue.TaskVideoConvertEQ(utils.Failed),
		entqueue.TaskVideoMoveEQ(utils.Failed),
		entqueue.TaskChatDownloadEQ(utils.Failed),
		entqueue.TaskChatConvertEQ(utils.Failed),
		entqueue.TaskChatRenderEQ(utils.Failed),
		entqueue.TaskChatMoveEQ(utils.Failed),
	)
	if q.Failed, err = s.Store.Client.Queue.Query().Where(failedPredicate).Count(ctx); err != nil {
		return q, err
	}

	return q, nil
}

// GetStorageDistribution retrieves the storage distribution across channels and the largest videos.
func (s *Service) GetStorageDistribution(ctx context.Context) (GetStorageDistributionResponse, error) {
	var resp GetStorageDistributionResponse

	// Get all channels with their total storage used
	channels, err := s.Store.Client.Channel.Query().
		WithVods(func(q *ent.VodQuery) {
			q.Select(entVod.FieldStorageSizeBytes)
		}).
		All(ctx)
	if err != nil {
		return resp, fmt.Errorf("error getting channels with vods: %w", err)
	}

	storageDistribution := make(map[string]int64)
	for _, channel := range channels {
		var totalStorage int64
		for _, vod := range channel.Edges.Vods {
			totalStorage += vod.StorageSizeBytes
		}
		storageDistribution[channel.Name] = totalStorage
	}

	resp.StorageDistribution = storageDistribution

	// Total storage used per video type (archive, live, clip, highlight, upload)
	var storageByTypeStats []struct {
		Type    string        `json:"type"`
		Storage sql.NullInt64 `json:"storage"`
	}
	err = s.Store.Client.Vod.Query().
		GroupBy(entVod.FieldType).
		Aggregate(ent.As(ent.Sum(entVod.FieldStorageSizeBytes), "storage")).
		Scan(ctx, &storageByTypeStats)
	if err != nil {
		return resp, fmt.Errorf("error getting storage by type: %w", err)
	}

	storageByType := make(map[string]int64)
	for _, st := range storageByTypeStats {
		if st.Storage.Valid {
			storageByType[st.Type] = st.Storage.Int64
		} else {
			storageByType[st.Type] = 0
		}
	}
	resp.StorageByType = storageByType

	// Top largest videos by storage size
	largestVods, err := s.Store.Client.Vod.Query().
		WithChannel().
		Order(ent.Desc(entVod.FieldStorageSizeBytes)).
		Limit(10).
		All(ctx)
	if err != nil {
		return resp, fmt.Errorf("error getting largest videos: %w", err)
	}

	resp.LargestVideos = make([]LargestVideoItem, 0, len(largestVods))
	for _, vod := range largestVods {
		item := LargestVideoItem{
			ID:               vod.ID.String(),
			Title:            vod.Title,
			StorageSizeBytes: vod.StorageSizeBytes,
			Duration:         vod.Duration,
			StreamedAt:       vod.StreamedAt,
		}
		if vod.Edges.Channel != nil {
			item.ChannelName = vod.Edges.Channel.Name
		}
		resp.LargestVideos = append(resp.LargestVideos, item)
	}

	return resp, nil
}
