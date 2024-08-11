package channel_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/tests"
)

func TestChannelCRUD(t *testing.T) {
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// test CreateChannel
	chann, err := app.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "123456789",
		Name:        "test_channel",
		DisplayName: "Test Channel",
		ImagePath:   "/vods/test_channel/test_channel.jpg",
	})

	assert.NoError(t, err)
	assert.Equal(t, "123456789", chann.ExtID)
	assert.Equal(t, "test_channel", chann.Name)
	assert.Equal(t, "Test Channel", chann.DisplayName)
	assert.Equal(t, "/vods/test_channel/test_channel.jpg", chann.ImagePath)

	// test GetChannel
	getChannel, err := app.ChannelService.GetChannel(chann.ID)
	assert.NoError(t, err)
	assert.Equal(t, chann.ID, getChannel.ID)

	// test GetChannelByName
	getChannelByName, err := app.ChannelService.GetChannelByName(chann.Name)
	assert.NoError(t, err)
	assert.Equal(t, chann.ID, getChannelByName.ID)

	// test GetChannels
	channels, err := app.ChannelService.GetChannels()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(channels))

	// test UpdateChannel
	updatedChannel, err := app.ChannelService.UpdateChannel(chann.ID, channel.Channel{
		Name:          "updated_channel",
		DisplayName:   "Updated Channel",
		ImagePath:     "/vods/updated_channel/updated_channel.jpg",
		Retention:     true,
		RetentionDays: 30,
	})
	assert.NoError(t, err)
	assert.Equal(t, "updated_channel", updatedChannel.Name)
	assert.Equal(t, "Updated Channel", updatedChannel.DisplayName)
	assert.Equal(t, "/vods/updated_channel/updated_channel.jpg", updatedChannel.ImagePath)
	assert.Equal(t, true, updatedChannel.Retention)
	assert.Equal(t, int64(30), updatedChannel.RetentionDays)

	// test CheckChannelExists
	assert.True(t, app.ChannelService.CheckChannelExists(updatedChannel.Name))

	// test DeleteChannel
	err = app.ChannelService.DeleteChannel(updatedChannel.ID)
	assert.NoError(t, err)
	assert.False(t, app.ChannelService.CheckChannelExists(updatedChannel.Name))
}

func TestPlatformTwitchChannel(t *testing.T) {
	ctx := context.Background()
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	// test ArchiveChannel
	chann, err := app.ArchiveService.ArchiveChannel(ctx, "sodapoppin")
	assert.NoError(t, err)
	assert.Equal(t, "sodapoppin", chann.Name)

	if _, err := os.Stat(chann.ImagePath); errors.Is(err, os.ErrNotExist) {
		t.Errorf("image not found: %s", chann.ImagePath)
	}

	// remove image
	err = os.Remove(chann.ImagePath)
	assert.NoError(t, err)

	// test UpdateChannelImage
	assert.NoError(t, app.ChannelService.UpdateChannelImage(ctx, chann.ID))

	// ensure image exists
	if _, err := os.Stat(chann.ImagePath); errors.Is(err, os.ErrNotExist) {
		t.Errorf("image not found: %s", chann.ImagePath)
	}
}
