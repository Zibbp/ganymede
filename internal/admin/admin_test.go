package admin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/tests"
)

func TestGetStats(t *testing.T) {
	ctx := context.Background()
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// create a channel for to test\
	_, err = app.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "123456789",
		Name:        "test_channel",
		DisplayName: "Test Channel",
		ImagePath:   "/vods/test_channel/test_channel.jpg",
	})
	assert.NoError(t, err)

	// test GetStats
	stats, err := app.AdminService.GetStats(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.VodCount)
	assert.Equal(t, 1, stats.ChannelCount)
}
