package channel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/server"
	"github.com/zibbp/ganymede/tests"
)

type ChannelTest struct {
	App *server.Application
}

var (
	TestChannelName        = "test_channel"
	TestChannelExtID       = "123456789"
	TestChannelDisplayName = "Test Channel"
	TestChannelImagePath   = "/vods/test_channel/test_channel.jpg"
)

// TestChannel tests the channel service. This function runs all the tests to avoid spinning up multiple containers.
func TestChannel(t *testing.T) {
	app, err := tests.Setup(t)
	assert.NoError(t, err)

	channelTest := ChannelTest{App: app}

	t.Run("TestCreateChannel", channelTest.CreateChannelTest)
	t.Run("TestCreateChannelInvalid", channelTest.CreateChannelInvalidTest)
	t.Run("TestGetChannels", channelTest.GetChannelsTest)
	t.Run("TestGetChannel", channelTest.GetChannelTest)
	t.Run("TestGetChannelByName", channelTest.GetChannelByNameTest)
	t.Run("TestGetChannelByExtId", channelTest.GetChannelByExtIdTest)
	t.Run("TestDeleteChannel", channelTest.DeleteChannelTest)
	t.Run("TestUpdateChannel", channelTest.UpdateChannelTest)
	t.Run("TestCheckChannelExists", channelTest.CheckChannelExistsTest)
	t.Run("TestCheckChannelExistsByExtId", channelTest.CheckChannelExistsByExtIdTest)

}

// CreateChannelTest tests the CreateChannel function
func (s *ChannelTest) CreateChannelTest(t *testing.T) {
	channel, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       TestChannelExtID,
		Name:        TestChannelName,
		DisplayName: TestChannelDisplayName,
		ImagePath:   TestChannelImagePath,
	})
	assert.NoError(t, err)
	assert.Equal(t, "123456789", channel.ExtID)
	assert.Equal(t, "test_channel", channel.Name)
	assert.Equal(t, "Test Channel", channel.DisplayName)
	assert.Equal(t, "/vods/test_channel/test_channel.jpg", channel.ImagePath)
}

// CreateChannelInvalid tests the CreateChannel function with invalid data
func (s *ChannelTest) CreateChannelInvalidTest(t *testing.T) {
	_, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "123456789", // duplicate of the previous test
		Name:        "duplicate channel",
		DisplayName: "Duplicate Channel",
		ImagePath:   "/vods/test_channel/duplicate_channel.jpg",
	})
	assert.Error(t, err)
}

// GetChannelsTest tests the GetChannels function
func (s *ChannelTest) GetChannelsTest(t *testing.T) {
	channels, err := s.App.ChannelService.GetChannels()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(channels))
}

// GetChannelTest tests the GetChannel function
func (s *ChannelTest) GetChannelTest(t *testing.T) {
	channel, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "12345",
		Name:        "get_channel_test",
		DisplayName: "Get Channel Test",
		ImagePath:   "/vods/get_channel_test/get_channel_test.jpg",
	})
	assert.NoError(t, err)

	getChannel, err := s.App.ChannelService.GetChannel(channel.ID)
	assert.NoError(t, err)
	assert.Equal(t, channel.ID, getChannel.ID)
	assert.Equal(t, "get_channel_test", getChannel.Name)
	assert.Equal(t, "Get Channel Test", getChannel.DisplayName)
	assert.Equal(t, "/vods/get_channel_test/get_channel_test.jpg", getChannel.ImagePath)
}

// GetChannelByNameTest tests the GetChannelByName function
func (s *ChannelTest) GetChannelByNameTest(t *testing.T) {
	channel, err := s.App.ChannelService.GetChannelByName(TestChannelName)
	assert.NoError(t, err)
	assert.Equal(t, TestChannelName, channel.Name)
	assert.Equal(t, TestChannelExtID, channel.ExtID)
	assert.Equal(t, TestChannelDisplayName, channel.DisplayName)
	assert.Equal(t, TestChannelImagePath, channel.ImagePath)
}

// GetChannelByExtIdTest tests the GetChannelByExtId function
func (s *ChannelTest) GetChannelByExtIdTest(t *testing.T) {
	channel, err := s.App.ChannelService.GetChannelByExtId(TestChannelExtID)
	assert.NoError(t, err)
	assert.Equal(t, TestChannelName, channel.Name)
	assert.Equal(t, TestChannelExtID, channel.ExtID)
	assert.Equal(t, TestChannelDisplayName, channel.DisplayName)
	assert.Equal(t, TestChannelImagePath, channel.ImagePath)
}

// DeleteChannelTest tests the DeleteChannel function
func (s *ChannelTest) DeleteChannelTest(t *testing.T) {
	channel, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "1234",
		Name:        "delete_me",
		DisplayName: "Delete Me",
		ImagePath:   "/vods/delete_me/delete_me.jpg",
	})
	assert.NoError(t, err)

	err = s.App.ChannelService.DeleteChannel(channel.ID)
	assert.NoError(t, err)
	_, err = s.App.ChannelService.GetChannel(channel.ID)
	assert.Error(t, err)
}

// UpdateChannelTest tests the UpdateChannel function
func (s *ChannelTest) UpdateChannelTest(t *testing.T) {
	createdChannel, err := s.App.ChannelService.CreateChannel(channel.Channel{
		ExtID:       "1234",
		Name:        "update_me",
		DisplayName: "Update Me",
		ImagePath:   "/vods/update_me/update_me.jpg",
	})
	assert.NoError(t, err)

	updatedChannel, err := s.App.ChannelService.UpdateChannel(createdChannel.ID, channel.Channel{
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
}

// CheckChannelExistsTest tests the CheckChannelExists function
func (s *ChannelTest) CheckChannelExistsTest(t *testing.T) {
	exists := s.App.ChannelService.CheckChannelExists(TestChannelName)
	assert.True(t, exists)

	exists = s.App.ChannelService.CheckChannelExists("non_existent_channel")
	assert.False(t, exists)
}

// CheckChannelExistsByExtIdTest tests the CheckChannelExistsByExtId function
func (s *ChannelTest) CheckChannelExistsByExtIdTest(t *testing.T) {
	exists := s.App.ChannelService.CheckChannelExistsByExtId(TestChannelExtID)
	assert.True(t, exists)

	exists = s.App.ChannelService.CheckChannelExistsByExtId("123")
	assert.False(t, exists)
}

// func TestPlatformTwitchChannel(t *testing.T) {
// 	ctx := context.Background()
// 	app, err := tests.Setup(t)
// 	assert.NoError(t, err)

// 	// test ArchiveChannel
// 	chann, err := app.ArchiveService.ArchiveChannel(ctx, "sodapoppin")
// 	assert.NoError(t, err)
// 	assert.Equal(t, "sodapoppin", chann.Name)

// 	if _, err := os.Stat(chann.ImagePath); errors.Is(err, os.ErrNotExist) {
// 		t.Errorf("image not found: %s", chann.ImagePath)
// 	}

// 	// remove image
// 	err = os.Remove(chann.ImagePath)
// 	assert.NoError(t, err)

// 	// test UpdateChannelImage
// 	assert.NoError(t, app.ChannelService.UpdateChannelImage(ctx, chann.ID))

// 	// ensure image exists
// 	if _, err := os.Stat(chann.ImagePath); errors.Is(err, os.ErrNotExist) {
// 		t.Errorf("image not found: %s", chann.ImagePath)
// 	}
// }
