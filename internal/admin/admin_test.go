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

	t.Run("TestGetStats", adminTest.GetStatsTest)

}

// GetStatsTest tests the GetStats function
func (s *AdminTest) GetStatsTest(t *testing.T) {
	_, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "123456789",
		Name:        "test_channel",
		DisplayName: "Test Channel",
		ImagePath:   "/vods/test_channel/test_channel.jpg",
	})
	assert.NoError(t, err)

	stats, err := s.App.AdminService.GetStats(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.VodCount)
	assert.Equal(t, 1, stats.ChannelCount)
}
