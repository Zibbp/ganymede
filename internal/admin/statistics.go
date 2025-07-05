package admin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zibbp/ganymede/ent"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/utils"
)

type GetVideoStatisticsResponse struct {
	VideoCount    int            `json:"video_count"`
	ChannelCount  int            `json:"channel_count"`
	ChannelVideos map[string]int `json:"channel_videos"`
	VideoTypes    map[string]int `json:"video_types"`
}

type GetSystemOverviewResponse struct {
	VideosDirectoryFreeSpace int64 `json:"videos_directory_free_space"` // Free space in bytes
	VideosDirectoryUsedSpace int64 `json:"videos_directory_used_space"` // Used space in bytes
	CPUCores                 int   `json:"cpu_cores"`                   // Number of CPU cores
	MemoryTotal              int64 `json:"memory_total"`                // Total memory in bytes
}

type GetStorageDistributionResponse struct {
	StorageDistribution map[string]int64 `json:"storage_distribution"` // Map of channel names to total storage used
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
	for _, vod := range vods {
		if vod.Edges.Channel != nil {
			channelVideosMap[vod.Edges.Channel.Name]++
		}
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

	return resp, nil
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

	return resp, nil
}
