package admin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/tests"
)

type AdminTest struct {
	App *server.Application
}

// TestAdmin tests the admin service. This function runs all the tests to avoid spinning up multiple containers.
func TestAdmin(t *testing.T) {
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	adminTest := AdminTest{App: app}

	t.Run("TestGetVideoStatistics", adminTest.GetVideoStatisticsTest)
	t.Run("TestGetSystemOverview", adminTest.GetSystemOverviewTest)
	t.Run("TestGetStorageDistribution", adminTest.GetStorageDistributionTest)

}

// GetVideoStatisticsTest tests the GetVideoStatistics function
func (s *AdminTest) GetVideoStatisticsTest(t *testing.T) {
	_, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "123456789",
		Name:        "test_channel",
		DisplayName: "Test Channel",
		ImagePath:   "/vods/test_channel/test_channel.jpg",
	})
	assert.NoError(t, err)

	stats, err := s.App.AdminService.GetVideoStatistics(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.VideoCount)
	assert.Equal(t, 1, stats.ChannelCount)
	assert.Equal(t, 0, stats.ChannelVideos["test_channel"])
	assert.Equal(t, 0, stats.VideoTypes["live"])
	assert.Equal(t, int64(0), stats.TotalDurationSeconds)
	assert.Equal(t, int64(0), stats.TotalViews)
	assert.Equal(t, int64(0), stats.TotalLocalViews)
	assert.Equal(t, int64(0), stats.TotalStorageBytes)
	assert.NotNil(t, stats.ChannelVideos)
	assert.NotNil(t, stats.VideoTypes)
}

// GetSystemOverviewTest tests the GetVideoStatistics function
func (s *AdminTest) GetSystemOverviewTest(t *testing.T) {
	_, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "12345678910",
		Name:        "test_channel1",
		DisplayName: "Test Channel1",
		ImagePath:   "/vods/test_channel/test_channel.jpg",
	})
	assert.NoError(t, err)

	overview, err := s.App.AdminService.GetSystemOverview(context.Background())
	assert.NoError(t, err)
	assert.LessOrEqual(t, 1, overview.CPUCores)
	assert.LessOrEqual(t, int64(1), overview.MemoryTotal)
	assert.LessOrEqual(t, int64(1), overview.VideosDirectoryFreeSpace)
	assert.Equal(t, int64(0), overview.VideosDirectoryUsedSpace)
	assert.Equal(t, overview.VideosDirectoryFreeSpace+overview.VideosDirectoryUsedSpace, overview.VideosDirectoryTotalSpace)
	assert.GreaterOrEqual(t, overview.Queue.Total, 0)
	assert.GreaterOrEqual(t, overview.Queue.Processing, 0)
	assert.GreaterOrEqual(t, overview.Queue.OnHold, 0)
	assert.GreaterOrEqual(t, overview.Queue.LiveArchiving, 0)
	assert.GreaterOrEqual(t, overview.Queue.Failed, 0)
	assert.GreaterOrEqual(t, overview.UserCount, 0)
}

// GetStorageDistributionTest tests the GetStorageDistribution function
func (s *AdminTest) GetStorageDistributionTest(t *testing.T) {
	_, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "12345678911",
		Name:        "test_channel2",
		DisplayName: "Test Channel2",
		ImagePath:   "/vods/test_channel/test_channel.jpg",
	})
	assert.NoError(t, err)

	resp, err := s.App.AdminService.GetStorageDistribution(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), resp.StorageDistribution["test_channel2"])
	assert.NotNil(t, resp.StorageDistribution)
	assert.NotNil(t, resp.StorageByType)
	assert.NotNil(t, resp.LargestVideos)
}
