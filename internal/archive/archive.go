package archive

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/blocked"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

type Service struct {
	Store              *database.Database
	ChannelService     *channel.Service
	VodService         *vod.Service
	QueueService       *queue.Service
	BlockedVodsService *blocked.Service
	RiverClient        *tasks_client.RiverClient
	PlatformTwitch     platform.Platform
	PlatformKick       platform.Platform
}

type TwitchVodResponse struct {
	VOD   *ent.Vod   `json:"vod"`
	Queue *ent.Queue `json:"queue"`
}

func NewService(store *database.Database, channelService *channel.Service, vodService *vod.Service, queueService *queue.Service, blockedVodService *blocked.Service, riverClient *tasks_client.RiverClient, platformTwitch platform.Platform, platformKick platform.Platform) *Service {
	return &Service{Store: store, ChannelService: channelService, VodService: vodService, QueueService: queueService, BlockedVodsService: blockedVodService, RiverClient: riverClient, PlatformTwitch: platformTwitch, PlatformKick: platformKick}
}

// ArchiveChannel - Create channel entry in database along with folder, profile image, etc.
func (s *Service) ArchiveChannel(ctx context.Context, channelName string) (*ent.Channel, error) {
	if s.PlatformTwitch == nil {
		return nil, fmt.Errorf("twitch platform is not configured; set TWITCH_CLIENT_ID/SECRET")
	}

	env := config.GetEnvConfig()
	// get channel from platform
	platformChannel, err := s.PlatformTwitch.GetChannel(ctx, channelName)
	if err != nil {
		return nil, fmt.Errorf("error fetching twitch channel: %v", err)
	}

	// Check if channel exists in DB
	cCheck := s.ChannelService.CheckChannelExists(platformChannel.Login)
	if cCheck {
		return nil, fmt.Errorf("channel already exists")
	}

	// Create channel folder
	err = utils.CreateDirectory(fmt.Sprintf("%s/%s", env.VideosDir, platformChannel.Login))
	if err != nil {
		return nil, fmt.Errorf("error creating channel folder: %v", err)
	}

	// Download channel profile image
	err = utils.DownloadFile(platformChannel.ProfileImageURL, fmt.Sprintf("%s/%s/%s", env.VideosDir, platformChannel.Login, "profile.png"))
	if err != nil {
		log.Error().Err(err).Msg("error downloading channel profile image")
	}

	// Create channel in DB
	channelDTO := channel.Channel{
		ExtID:       platformChannel.ID,
		Name:        platformChannel.Login,
		DisplayName: platformChannel.DisplayName,
		ImagePath:   fmt.Sprintf("%s/%s/profile.png", env.VideosDir, platformChannel.Login),
	}

	dbC, err := s.ChannelService.CreateChannel(channelDTO)
	if err != nil {
		return nil, fmt.Errorf("error creating channel: %v", err)
	}

	return dbC, nil

}

type ArchiveVideoInput struct {
	VideoId     string
	ChannelId   uuid.UUID
	Quality     utils.VodQuality
	ArchiveChat bool
	RenderChat  bool
	Platform    utils.VideoPlatform
}

func (s *Service) ArchiveVideo(ctx context.Context, input ArchiveVideoInput) (*ArchiveResponse, error) {
	// log.Debug().Msgf("Archiving video %s quality: %s chat: %t render chat: %t", videoId, quality, chat, renderChat)

	envConfig := config.GetEnvConfig()

	// check if video is blocked
	blocked, err := s.BlockedVodsService.IsVideoBlocked(ctx, input.VideoId)
	if err != nil {
		return nil, fmt.Errorf("error checking if vod is blocked: %v", err)
	}
	if blocked {
		return nil, fmt.Errorf("video id is blocked")
	}

	// get video
	video, err := s.PlatformTwitch.GetVideo(context.Background(), input.VideoId, false, false)
	if err != nil {
		return nil, err
	}

	if input.Platform == utils.PlatformTwitch {
		// check if video is processing
		if strings.Contains(video.ThumbnailURL, "processing") {
			return fmt.Errorf("vod is still processing")
		}
	}

	// Check if video is already archived
	vCheck, err := s.VodService.CheckVodExists(video.ID)
	if err != nil {
		return nil, fmt.Errorf("error checking if vod exists: %v", err)
	}
	if vCheck {
		return nil, fmt.Errorf("vod already exists")
	}

	// Check if channel exists
	cCheck := s.ChannelService.CheckChannelExists(video.UserLogin)
	if !cCheck {
		log.Debug().Msgf("channel does not exist: %s while archiving vod. creating now.", video.UserLogin)
		_, err := s.ArchiveChannel(ctx, video.UserLogin)
		if err != nil {
			return nil, fmt.Errorf("error creating channel: %v", err)
		}
	}

	// Fetch channel
	channel, err := s.ChannelService.GetChannelByName(video.UserLogin)
	if err != nil {
		return nil, fmt.Errorf("error fetching channel: %v", err)
	}

	// Generate Ganymede video ID for directory and file naming
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error creating vod uuid: %v", err)
	}

	storageTemplateInput := StorageTemplateInput{
		UUID:    vUUID,
		ID:      input.VideoId,
		Channel: channel.Name,
		Title:   video.Title,
		Type:    video.Type,
		Date:    video.CreatedAt.Format("2006-01-02"),
		YYYY:    video.CreatedAt.Format("2006"),
		MM:      video.CreatedAt.Format("01"),
		DD:      video.CreatedAt.Format("02"),
		HH:      video.CreatedAt.Format("15"),
	}
	// Create directory paths
	folderName, err := GetFolderName(vUUID, storageTemplateInput)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create folder name, falling back to default")
		folderName = fmt.Sprintf("%s-%s", video.ID, vUUID.String())
	}
	fileName, err := GetFileName(vUUID, storageTemplateInput)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create file name, falling back to default")
		fileName = video.ID
	}

	// set facts
	rootVideoPath := fmt.Sprintf("%s/%s/%s", envConfig.VideosDir, video.UserLogin, folderName)
	chatPath := ""
	chatVideoPath := ""
	liveChatPath := ""
	liveChatConvertPath := ""

	if input.ArchiveChat {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVideoPath, fileName)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVideoPath, fileName)
		if utils.VodType(video.Type) == utils.Live {
			liveChatPath = fmt.Sprintf("%s/%s-live-chat.json", rootVideoPath, fileName)
			liveChatConvertPath = fmt.Sprintf("%s/%s-chat-convert.json", rootVideoPath, fileName)
		}
	}

	videoExtension := "mp4"

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:                  vUUID,
		ExtID:               video.ID,
		Platform:            "twitch",
		Type:                utils.VodType(video.Type),
		Title:               video.Title,
		Duration:            int(video.Duration.Seconds()),
		Views:               int(video.ViewCount),
		Resolution:          input.Quality.String(),
		Processing:          true,
		ThumbnailPath:       fmt.Sprintf("%s/%s-thumbnail.jpg", rootVideoPath, fileName),
		WebThumbnailPath:    fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVideoPath, fileName),
		VideoPath:           fmt.Sprintf("%s/%s-video.%s", rootVideoPath, fileName, videoExtension),
		ChatPath:            chatPath,
		LiveChatPath:        liveChatPath,
		ChatVideoPath:       chatVideoPath,
		LiveChatConvertPath: liveChatConvertPath,
		InfoPath:            fmt.Sprintf("%s/%s-info.json", rootVideoPath, fileName),
		StreamedAt:          video.CreatedAt,
		FolderName:          folderName,
		FileName:            fileName,
		// create temporary paths
		TmpVideoDownloadPath:    fmt.Sprintf("%s/%s_%s-video.%s", envConfig.TempDir, video.ID, vUUID, videoExtension),
		TmpVideoConvertPath:     fmt.Sprintf("%s/%s_%s-video-convert.%s", envConfig.TempDir, video.ID, vUUID, videoExtension),
		TmpChatDownloadPath:     fmt.Sprintf("%s/%s_%s-chat.json", envConfig.TempDir, video.ID, vUUID),
		TmpLiveChatDownloadPath: fmt.Sprintf("%s/%s_%s-live-chat.json", envConfig.TempDir, video.ID, vUUID),
		TmpLiveChatConvertPath:  fmt.Sprintf("%s/%s_%s-chat-convert.json", envConfig.TempDir, video.ID, vUUID),
		TmpChatRenderPath:       fmt.Sprintf("%s/%s_%s-chat.mp4", envConfig.TempDir, video.ID, vUUID),
	}

	if config.Get().Archive.SaveAsHls {
		vodDTO.TmpVideoHLSPath = fmt.Sprintf("%s/%s_%s-video_hls0", envConfig.TempDir, video.ID, vUUID)
		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVideoPath, fileName)
		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVideoPath, fileName, video.ID)
	}

	v, err := s.VodService.CreateVod(vodDTO, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: false, ArchiveChat: input.ArchiveChat, RenderChat: input.RenderChat}, v.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if !input.ArchiveChat {
		_, err := q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatPath("").SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating vod: %v", err)
		}
	}

	// If render chat is disabled update queue
	if !input.RenderChat {
		_, err := q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating vod: %v", err)
		}
	}

	// Re-query queue from DB for updated values
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		return nil, fmt.Errorf("error fetching queue item: %v", err)
	}

	taskInput := tasks.ArchiveVideoInput{
		QueueId: q.ID,
	}

	// enqueue first task
	_, err = s.RiverClient.Client.Insert(ctx, tasks.CreateDirectoryArgs{
		Continue: true,
		Input:    taskInput,
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("error enqueueing task: %v", err)
	}

	return &ArchiveResponse{
		Queue: q,
		Video: v,
	}, nil
}

type ArchiveClipInput struct {
	ID          string
	ChannelId   uuid.UUID
	Quality     utils.VodQuality
	ArchiveChat bool
	RenderChat  bool
	Platform    utils.VideoPlatform
}

// ArchiveClip archives a clip from a platform
func (s *Service) ArchiveClip(ctx context.Context, input ArchiveClipInput) (*ArchiveResponse, error) {

	envConfig := config.GetEnvConfig()

	// check if video is blocked
	blocked, err := s.BlockedVodsService.IsVideoBlocked(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("error checking if clip is blocked: %v", err)
	}
	if blocked {
		return nil, fmt.Errorf("clip id is blocked")
	}

	// get clip
	clip, err := s.PlatformTwitch.GetClip(context.Background(), input.ID)
	if err != nil {
		return nil, err
	}

	// Check if video is already archived
	vCheck, err := s.VodService.CheckVodExists(clip.ID)
	if err != nil {
		return nil, fmt.Errorf("error checking if clip exists: %v", err)
	}
	if vCheck {
		return nil, fmt.Errorf("clip already exists")
	}

	// Check if channel exists
	cCheck := s.ChannelService.CheckChannelExistsByExtId(clip.ChannelID)
	if !cCheck {
		log.Debug().Msg("channel does not exist: %s while archiving clip. creating now")
		_, err := s.ArchiveChannel(ctx, *clip.ChannelName)
		if err != nil {
			return nil, fmt.Errorf("error creating channel: %v", err)
		}
	}

	// Fetch channel
	channel, err := s.ChannelService.GetChannelByExtId(clip.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("error fetching channel: %v", err)
	}

	// Generate Ganymede video ID for directory and file naming
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error creating vod uuid: %v", err)
	}

	storageTemplateInput := StorageTemplateInput{
		UUID:    vUUID,
		ID:      clip.ID,
		Channel: channel.Name,
		Title:   clip.Title,
		Type:    string(utils.Clip),
		Date:    clip.CreatedAt.Format("2006-01-02"),
		YYYY:    clip.CreatedAt.Format("2006"),
		MM:      clip.CreatedAt.Format("01"),
		DD:      clip.CreatedAt.Format("02"),
		HH:      clip.CreatedAt.Format("15"),
	}
	// Create directory paths
	folderName, err := GetFolderName(vUUID, storageTemplateInput)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create folder name, falling back to default")
		folderName = fmt.Sprintf("%s-%s", clip.ID, vUUID.String())
	}
	fileName, err := GetFileName(vUUID, storageTemplateInput)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create file name, falling back to default")
		fileName = clip.ID
	}

	// set facts
	rootVideoPath := fmt.Sprintf("%s/%s/%s", envConfig.VideosDir, channel.Name, folderName)
	chatPath := ""
	chatVideoPath := ""
	liveChatPath := ""
	liveChatConvertPath := ""

	// Disable chat archive if the clip doesn't have a VOD to fetch the chat from
	if clip.VideoID == "" {
		input.ArchiveChat = false
	}

	if input.ArchiveChat {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVideoPath, fileName)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVideoPath, fileName)
	}

	videoExtension := "mp4"

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:                  vUUID,
		ExtID:               clip.ID,
		ClipExtVodID:        clip.VideoID,
		Platform:            "twitch",
		Type:                utils.Clip,
		Title:               clip.Title,
		Duration:            clip.Duration,
		ClipVodOffset:       *clip.VodOffset,
		Views:               int(clip.ViewCount),
		Resolution:          input.Quality.String(),
		Processing:          true,
		ThumbnailPath:       fmt.Sprintf("%s/%s-thumbnail.jpg", rootVideoPath, fileName),
		WebThumbnailPath:    fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVideoPath, fileName),
		VideoPath:           fmt.Sprintf("%s/%s-video.%s", rootVideoPath, fileName, videoExtension),
		ChatPath:            chatPath,
		LiveChatPath:        liveChatPath,
		ChatVideoPath:       chatVideoPath,
		LiveChatConvertPath: liveChatConvertPath,
		InfoPath:            fmt.Sprintf("%s/%s-info.json", rootVideoPath, fileName),
		StreamedAt:          clip.CreatedAt,
		FolderName:          folderName,
		FileName:            fileName,
		// create temporary paths
		TmpVideoDownloadPath:    fmt.Sprintf("%s/%s_%s-video.%s", envConfig.TempDir, clip.ID, vUUID, videoExtension),
		TmpVideoConvertPath:     fmt.Sprintf("%s/%s_%s-video-convert.%s", envConfig.TempDir, clip.ID, vUUID, videoExtension),
		TmpChatDownloadPath:     fmt.Sprintf("%s/%s_%s-chat.json", envConfig.TempDir, clip.ID, vUUID),
		TmpLiveChatDownloadPath: fmt.Sprintf("%s/%s_%s-live-chat.json", envConfig.TempDir, clip.ID, vUUID),
		TmpLiveChatConvertPath:  fmt.Sprintf("%s/%s_%s-chat-convert.json", envConfig.TempDir, clip.ID, vUUID),
		TmpChatRenderPath:       fmt.Sprintf("%s/%s_%s-chat.mp4", envConfig.TempDir, clip.ID, vUUID),
	}

	if config.Get().Archive.SaveAsHls {
		vodDTO.TmpVideoHLSPath = fmt.Sprintf("%s/%s_%s-video_hls0", envConfig.TempDir, clip.ID, vUUID)
		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVideoPath, fileName)
		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVideoPath, fileName, clip.ID)
	}

	v, err := s.VodService.CreateVod(vodDTO, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: false, ArchiveChat: input.ArchiveChat, RenderChat: input.RenderChat}, v.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if !input.ArchiveChat {
		_, err := q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatPath("").SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating vod: %v", err)
		}
	}

	// If render chat is disabled update queue
	if !input.RenderChat {
		_, err := q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating vod: %v", err)
		}
	}

	// Re-query queue from DB for updated values
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		return nil, fmt.Errorf("error fetching queue item: %v", err)
	}

	taskInput := tasks.ArchiveVideoInput{
		QueueId: q.ID,
	}

	// enqueue first task
	_, err = s.RiverClient.Client.Insert(ctx, tasks.CreateDirectoryArgs{
		Continue: true,
		Input:    taskInput,
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("error enqueueing task: %v", err)
	}

	return &ArchiveResponse{
		Queue: q,
		Video: v,
	}, nil
}

func (s *Service) ArchiveLivestream(ctx context.Context, input ArchiveVideoInput) (*ArchiveResponse, error) {
	envConfig := config.GetEnvConfig()

	channel, err := s.ChannelService.GetChannel(input.ChannelId)
	if err != nil {
		return nil, fmt.Errorf("error fetching channel: %v", err)
	}

	// get video
	var video *platform.LiveStreamInfo
	switch input.Platform {
	case utils.PlatformTwitch:
		video, err = s.PlatformTwitch.GetLiveStream(context.Background(), channel.Name)
		if err != nil {
			return fmt.Errorf("error fetching live stream: %v", err)
		}
	case utils.PlatformKick:
		video, err = s.PlatformKick.GetLiveStream(context.Background(), channel.Name)
		if err != nil {
			return fmt.Errorf("error fetching live stream: %v", err)
		}
	default:
		return fmt.Errorf("unsupported platform: %s", input.Platform)
	}

	// Generate Ganymede video ID for directory and file naming
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error creating vod uuid: %v", err)
	}

	storageTemplateInput := StorageTemplateInput{
		UUID:    vUUID,
		ID:      video.ID,
		Channel: channel.Name,
		Title:   video.Title,
		Type:    video.Type,
		Date:    video.StartedAt.Format("2006-01-02"),
		YYYY:    video.StartedAt.Format("2006"),
		MM:      video.StartedAt.Format("01"),
		DD:      video.StartedAt.Format("02"),
		HH:      video.StartedAt.Format("15"),
	}
	// Create directory paths
	folderName, err := GetFolderName(vUUID, storageTemplateInput)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create folder name, falling back to default")
		folderName = fmt.Sprintf("%s-%s", video.ID, vUUID.String())
	}
	fileName, err := GetFileName(vUUID, storageTemplateInput)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create file name, falling back to default")
		fileName = video.ID
	}

	// set facts
	rootVideoPath := fmt.Sprintf("%s/%s/%s", envConfig.VideosDir, video.UserLogin, folderName)
	chatPath := ""
	chatVideoPath := ""
	liveChatPath := ""
	liveChatConvertPath := ""

	if input.ArchiveChat {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVideoPath, fileName)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVideoPath, fileName)
		liveChatPath = fmt.Sprintf("%s/%s-live-chat.json", rootVideoPath, fileName)
		liveChatConvertPath = fmt.Sprintf("%s/%s-chat-convert.json", rootVideoPath, fileName)
	}

	videoExtension := "mp4"

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:                  vUUID,
		ExtID:               video.ID,
		ExtStreamID:         video.ID,
		Platform:            input.Platform,
		Type:                utils.VodType(video.Type),
		Title:               video.Title,
		Duration:            1,
		Views:               1,
		Resolution:          input.Quality.String(),
		Processing:          true,
		ThumbnailPath:       fmt.Sprintf("%s/%s-thumbnail.jpg", rootVideoPath, fileName),
		WebThumbnailPath:    fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVideoPath, fileName),
		VideoPath:           fmt.Sprintf("%s/%s-video.%s", rootVideoPath, fileName, videoExtension),
		ChatPath:            chatPath,
		LiveChatPath:        liveChatPath,
		ChatVideoPath:       chatVideoPath,
		LiveChatConvertPath: liveChatConvertPath,
		InfoPath:            fmt.Sprintf("%s/%s-info.json", rootVideoPath, fileName),
		StreamedAt:          video.StartedAt,
		FolderName:          folderName,
		FileName:            fileName,
		// create temporary paths
		TmpVideoDownloadPath:    fmt.Sprintf("%s/%s_%s-video.%s", envConfig.TempDir, video.ID, vUUID, videoExtension),
		TmpVideoConvertPath:     fmt.Sprintf("%s/%s_%s-video-convert.%s", envConfig.TempDir, video.ID, vUUID, videoExtension),
		TmpChatDownloadPath:     fmt.Sprintf("%s/%s_%s-chat.json", envConfig.TempDir, video.ID, vUUID),
		TmpLiveChatDownloadPath: fmt.Sprintf("%s/%s_%s-live-chat.json", envConfig.TempDir, video.ID, vUUID),
		TmpLiveChatConvertPath:  fmt.Sprintf("%s/%s_%s-chat-convert.json", envConfig.TempDir, video.ID, vUUID),
		TmpChatRenderPath:       fmt.Sprintf("%s/%s_%s-chat.mp4", envConfig.TempDir, video.ID, vUUID),
	}

	if config.Get().Archive.SaveAsHls {
		vodDTO.TmpVideoHLSPath = fmt.Sprintf("%s/%s_%s-video_hls0", envConfig.TempDir, video.ID, vUUID)
		vodDTO.TmpVideoDownloadPath = fmt.Sprintf("%s/%s-video.m3u8", vodDTO.TmpVideoHLSPath, video.ID)
		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVideoPath, fileName)
		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVideoPath, fileName, video.ID)
	}

	v, err := s.VodService.CreateVod(vodDTO, channel.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: true, ArchiveChat: input.ArchiveChat, RenderChat: input.RenderChat}, v.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if !input.ArchiveChat {
		_, err := q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatPath("").SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating vod: %v", err)
		}
	}

	// If render chat is disabled update queue
	if !input.RenderChat {
		_, err := q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating vod: %v", err)
		}
	}

	// Re-query queue from DB for updated values
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		return nil, fmt.Errorf("error fetching queue item: %v", err)
	}

	taskInput := tasks.ArchiveVideoInput{
		QueueId: q.ID,
	}

	// enqueue first task
	_, err = s.RiverClient.Client.Insert(ctx, tasks.CreateDirectoryArgs{
		Continue: true,
		Input:    taskInput,
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("error enqueueing task: %v", err)
	}

	return &ArchiveResponse{
		Queue: q,
		Video: v,
	}, nil
}
