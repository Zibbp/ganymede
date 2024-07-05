package archive

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/platform"
	platform_twitch "github.com/zibbp/ganymede/internal/platform/twitch"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
)

type Service struct {
	Store          *database.Database
	ChannelService *channel.Service
	VodService     *vod.Service
	QueueService   *queue.Service
	RiverClient    *tasks_client.RiverClient
}

type TwitchVodResponse struct {
	VOD   *ent.Vod   `json:"vod"`
	Queue *ent.Queue `json:"queue"`
}

func NewService(store *database.Database, channelService *channel.Service, vodService *vod.Service, queueService *queue.Service, riverClient *tasks_client.RiverClient) *Service {
	return &Service{Store: store, ChannelService: channelService, VodService: vodService, QueueService: queueService, RiverClient: riverClient}
}

// ArchiveTwitchChannel - Create Twitch channel folder, profile image, and database entry.
func (s *Service) ArchiveTwitchChannel(cName string) (*ent.Channel, error) {
	// Fetch channel from Twitch API
	tChannel, err := twitch.API.GetUserByLogin(cName)
	if err != nil {
		return nil, fmt.Errorf("error fetching twitch channel: %v", err)
	}

	// Check if channel exists in DB
	cCheck := s.ChannelService.CheckChannelExists(tChannel.Login)
	if cCheck {
		return nil, fmt.Errorf("channel already exists")
	}

	// Create channel folder
	err = utils.CreateFolder(tChannel.Login)
	if err != nil {
		return nil, fmt.Errorf("error creating channel folder: %v", err)
	}

	// Download channel profile image
	err = utils.DownloadFile(tChannel.ProfileImageURL, tChannel.Login, "profile.png")
	if err != nil {
		return nil, fmt.Errorf("error downloading channel profile image: %v", err)
	}

	// Create channel in DB
	channelDTO := channel.Channel{
		ExtID:       tChannel.ID,
		Name:        tChannel.Login,
		DisplayName: tChannel.DisplayName,
		ImagePath:   fmt.Sprintf("/vods/%s/profile.png", tChannel.Login),
	}

	dbC, err := s.ChannelService.CreateChannel(channelDTO)
	if err != nil {
		return nil, fmt.Errorf("error creating channel: %v", err)
	}

	return dbC, nil

}

// ! NEW!!!!!!!!!!!

type ArchiveVideoInput struct {
	VideoId     string
	ChannelId   uuid.UUID
	Quality     utils.VodQuality
	ArchiveChat bool
	RenderChat  bool
}

func (s *Service) ArchiveVideo(ctx context.Context, input ArchiveVideoInput) error {
	// log.Debug().Msgf("Archiving video %s quality: %s chat: %t render chat: %t", videoId, quality, chat, renderChat)

	envConfig := config.GetEnvConfig()

	// setup platform service
	var platformService platform.PlatformService[platform_twitch.TwitchVideoInfo, platform_twitch.TwitchLivestreamInfo, platform_twitch.TwitchChannel]
	platformService, err := platform_twitch.NewTwitchPlatformService(
		envConfig.TwitchClientId,
		envConfig.TwitchClientSecret,
	)
	if err != nil {
		return err
	}

	// get video
	video, err := platformService.GetVideoById(context.Background(), input.VideoId)
	if err != nil {
		return err
	}

	// check if video is processing
	if strings.Contains(video.ThumbnailURL, "processing") {
		return fmt.Errorf("vod is still processing")
	}

	// Check if video is already archived
	vCheck, err := s.VodService.CheckVodExists(video.ID)
	if err != nil {
		return fmt.Errorf("error checking if vod exists: %v", err)
	}
	if vCheck {
		return fmt.Errorf("vod already exists")
	}

	// Check if channel exists
	cCheck := s.ChannelService.CheckChannelExists(video.UserLogin)
	if !cCheck {
		log.Debug().Msgf("channel does not exist: %s while archiving vod. creating now.", video.UserLogin)
		_, err := s.ArchiveTwitchChannel(video.UserLogin)
		if err != nil {
			return fmt.Errorf("error creating channel: %v", err)
		}
	}

	// Fetch channel
	channel, err := s.ChannelService.GetChannelByName(video.UserLogin)
	if err != nil {
		return fmt.Errorf("error fetching channel: %v", err)
	}

	// Generate Ganymede video ID for directory and file naming
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("error creating vod uuid: %v", err)
	}

	storageTemplateDate, err := parseDate(video.CreatedAt)
	if err != nil {
		return fmt.Errorf("error parsing date: %v", err)
	}

	storageTemplateInput := StorageTemplateInput{
		UUID:    vUUID,
		ID:      input.VideoId,
		Channel: channel.Name,
		Title:   video.Title,
		Type:    video.Type,
		Date:    storageTemplateDate,
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
	// Parse new Twitch API duration
	parsedDuration, err := time.ParseDuration(video.Duration)
	if err != nil {
		return fmt.Errorf("error parsing duration: %v", err)
	}

	// Parse Twitch date to time.Time
	parsedDate, err := time.Parse(time.RFC3339, video.CreatedAt)
	if err != nil {
		return fmt.Errorf("error parsing date: %v", err)
	}

	videoExtension := "mp4"

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:                  vUUID,
		ExtID:               video.ID,
		Platform:            "twitch",
		Type:                utils.VodType(video.Type),
		Title:               video.Title,
		Duration:            int(parsedDuration.Seconds()),
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
		StreamedAt:          parsedDate,
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

	if viper.GetBool("archive.save_as_hls") {
		vodDTO.TmpVideoHLSPath = fmt.Sprintf("%s/%s_%s-video_hls0", envConfig.TempDir, video.ID, vUUID)
		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVideoPath, fileName)
		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVideoPath, fileName, video.ID)
	}

	v, err := s.VodService.CreateVod(vodDTO, channel.ID)
	if err != nil {
		return fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: false, ArchiveChat: input.ArchiveChat, RenderChat: input.RenderChat}, v.ID)
	if err != nil {
		return fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if !input.ArchiveChat {
		_, err := q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatPath("").SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating vod: %v", err)
		}
	}

	// If render chat is disabled update queue
	if !input.RenderChat {
		_, err := q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating vod: %v", err)
		}
	}

	// Re-query queue from DB for updated values
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		return fmt.Errorf("error fetching queue item: %v", err)
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
		return fmt.Errorf("error enqueueing task: %v", err)
	}

	return nil
}

func (s *Service) ArchiveLivestream(ctx context.Context, input ArchiveVideoInput) error {
	envConfig := config.GetEnvConfig()

	channel, err := s.ChannelService.GetChannel(input.ChannelId)
	if err != nil {
		return fmt.Errorf("error fetching channel: %v", err)
	}

	// setup platform service
	var platformService platform.PlatformService[platform_twitch.TwitchVideoInfo, platform_twitch.TwitchLivestreamInfo, platform_twitch.TwitchChannel]
	platformService, err = platform_twitch.NewTwitchPlatformService(
		envConfig.TwitchClientId,
		envConfig.TwitchClientSecret,
	)
	if err != nil {
		return err
	}

	// get video
	video, err := platformService.GetLivestreamInfo(context.Background(), channel.Name)
	if err != nil {
		return err
	}

	// Generate Ganymede video ID for directory and file naming
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return fmt.Errorf("error creating vod uuid: %v", err)
	}

	storageTemplateDate, err := parseDate(video.StartedAt)
	if err != nil {
		return fmt.Errorf("error parsing date: %v", err)
	}

	storageTemplateInput := StorageTemplateInput{
		UUID:    vUUID,
		ID:      input.ChannelId.String(),
		Channel: channel.Name,
		Title:   video.Title,
		Type:    video.Type,
		Date:    storageTemplateDate,
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

	// Parse Twitch date to time.Time
	parsedDate, err := time.Parse(time.RFC3339, video.StartedAt)
	if err != nil {
		return fmt.Errorf("error parsing date: %v", err)
	}

	videoExtension := "mp4"

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:                  vUUID,
		ExtID:               video.ID,
		ExtStreamID:         video.ID,
		Platform:            "twitch",
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
		StreamedAt:          parsedDate,
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

	if viper.GetBool("archive.save_as_hls") {
		vodDTO.TmpVideoHLSPath = fmt.Sprintf("%s/%s_%s-video_hls0", envConfig.TempDir, video.ID, vUUID)
		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVideoPath, fileName)
		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVideoPath, fileName, video.ID)
	}

	v, err := s.VodService.CreateVod(vodDTO, channel.ID)
	if err != nil {
		return fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: true, ArchiveChat: input.ArchiveChat, RenderChat: input.RenderChat}, v.ID)
	if err != nil {
		return fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if !input.ArchiveChat {
		_, err := q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatPath("").SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating vod: %v", err)
		}
	}

	// If render chat is disabled update queue
	if !input.RenderChat {
		_, err := q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating queue item: %v", err)
		}
		_, err = v.Update().SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return fmt.Errorf("error updating vod: %v", err)
		}
	}

	// Re-query queue from DB for updated values
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		return fmt.Errorf("error fetching queue item: %v", err)
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
		return fmt.Errorf("error enqueueing task: %v", err)
	}

	return nil
}

// func (s *Service) ArchiveTwitchLive(lwc *ent.Live, live twitch.Live) (*TwitchVodResponse, error) {
// 	// Check if channel exists
// 	cCheck := s.ChannelService.CheckChannelExists(live.UserLogin)
// 	if !cCheck {
// 		log.Debug().Msgf("channel does not exist: %s while archiving live stream. creating now.", live.UserLogin)
// 		_, err := s.ArchiveTwitchChannel(live.UserLogin)
// 		if err != nil {
// 			return nil, fmt.Errorf("error creating channel: %v", err)
// 		}
// 	}
// 	// Fetch channel
// 	dbC, err := s.ChannelService.GetChannelByName(live.UserLogin)
// 	if err != nil {
// 		return nil, fmt.Errorf("error fetching channel: %v", err)
// 	}

// 	// Generate VOD ID for folder name
// 	vUUID, err := uuid.NewUUID()
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating vod uuid: %v", err)
// 	}

// 	// Create vodDto for storage templates
// 	tVodDto := twitch.Vod{
// 		ID:        live.ID,
// 		UserLogin: live.UserLogin,
// 		Title:     live.Title,
// 		Type:      "live",
// 		CreatedAt: live.StartedAt,
// 	}
// 	folderName, err := GetFolderName(vUUID, tVodDto)
// 	if err != nil {
// 		log.Error().Err(err).Msg("error using template to create folder name, falling back to default")
// 		folderName = fmt.Sprintf("%s-%s", tVodDto.ID, vUUID.String())
// 	}
// 	fileName, err := GetFileName(vUUID, tVodDto)
// 	if err != nil {
// 		log.Error().Err(err).Msg("error using template to create file name, falling back to default")
// 		fileName = tVodDto.ID
// 	}

// 	// Sets
// 	rootVodPath := fmt.Sprintf("/vods/%s/%s", live.UserLogin, folderName)
// 	chatPath := ""
// 	chatVideoPath := ""
// 	liveChatPath := ""
// 	liveChatConvertPath := ""

// 	if lwc.ArchiveChat {
// 		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVodPath, fileName)
// 		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVodPath, fileName)
// 		liveChatPath = fmt.Sprintf("%s/%s-live-chat.json", rootVodPath, fileName)
// 		liveChatConvertPath = fmt.Sprintf("%s/%s-chat-convert.json", rootVodPath, fileName)
// 	}

// 	videoExtension := "mp4"

// 	// Create VOD in DB
// 	vodDTO := vod.Vod{
// 		ID:                  vUUID,
// 		ExtID:               live.ID,
// 		Platform:            "twitch",
// 		Type:                utils.VodType("live"),
// 		Title:               live.Title,
// 		Duration:            1,
// 		Views:               1,
// 		Resolution:          lwc.Resolution,
// 		Processing:          true,
// 		ThumbnailPath:       fmt.Sprintf("%s/%s-thumbnail.jpg", rootVodPath, fileName),
// 		WebThumbnailPath:    fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVodPath, fileName),
// 		VideoPath:           fmt.Sprintf("%s/%s-video.%s", rootVodPath, fileName, videoExtension),
// 		ChatPath:            chatPath,
// 		LiveChatPath:        liveChatPath,
// 		ChatVideoPath:       chatVideoPath,
// 		LiveChatConvertPath: liveChatConvertPath,
// 		InfoPath:            fmt.Sprintf("%s/%s-info.json", rootVodPath, fileName),
// 		StreamedAt:          time.Now(),
// 		FolderName:          folderName,
// 		FileName:            fileName,
// 		// create temporary paths
// 		TmpVideoDownloadPath:    fmt.Sprintf("/tmp/%s_%s-video.%s", live.ID, vUUID, videoExtension),
// 		TmpVideoConvertPath:     fmt.Sprintf("/tmp/%s_%s-video-convert.%s", live.ID, vUUID, videoExtension),
// 		TmpChatDownloadPath:     fmt.Sprintf("/tmp/%s_%s-chat.json", live.ID, vUUID),
// 		TmpLiveChatDownloadPath: fmt.Sprintf("/tmp/%s_%s-live-chat.json", live.ID, vUUID),
// 		TmpLiveChatConvertPath:  fmt.Sprintf("/tmp/%s_%s-chat-convert.json", live.ID, vUUID),
// 		TmpChatRenderPath:       fmt.Sprintf("/tmp/%s_%s-chat.mp4", live.ID, vUUID),
// 	}

// 	if viper.GetBool("archive.save_as_hls") {
// 		vodDTO.TmpVideoHLSPath = fmt.Sprintf("/tmp/%s_%s-video_hls0", live.ID, vUUID)
// 		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVodPath, fileName)
// 		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVodPath, fileName, live.ID)
// 	}

// 	v, err := s.VodService.CreateVod(vodDTO, dbC.ID)
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating vod: %v", err)
// 	}

// 	// Create queue item
// 	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: true}, v.ID)
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating queue item: %v", err)
// 	}

// 	// If chat is disabled update queue
// 	if !lwc.ArchiveChat {
// 		_, err := q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(context.Background())
// 		if err != nil {
// 			return nil, fmt.Errorf("error updating queue item: %v", err)
// 		}

// 		_, err = v.Update().SetChatPath("").SetChatVideoPath("").Save(context.Background())
// 		if err != nil {
// 			return nil, fmt.Errorf("error updating vod: %v", err)
// 		}

// 	}

// 	if !lwc.RenderChat {
// 		_, err := q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).Save(context.Background())
// 		if err != nil {
// 			return nil, fmt.Errorf("error updating queue item: %v", err)
// 		}
// 		_, err = v.Update().SetChatVideoPath("").Save(context.Background())
// 		if err != nil {
// 			return nil, fmt.Errorf("error updating vod: %v", err)
// 		}
// 	}

// 	// Re-query queue from DB for updated values
// 	q, err = s.QueueService.GetQueueItem(q.ID)
// 	if err != nil {
// 		return nil, fmt.Errorf("error fetching queue item: %v", err)
// 	}

// 	wfOptions := client.StartWorkflowOptions{
// 		ID:        vUUID.String(),
// 		TaskQueue: "archive",
// 	}

// 	input := dto.ArchiveVideoInput{
// 		VideoID:          live.ID,
// 		Type:             "live",
// 		Platform:         "twitch",
// 		Resolution:       lwc.Resolution,
// 		DownloadChat:     lwc.ArchiveChat,
// 		RenderChat:       lwc.RenderChat,
// 		Vod:              v,
// 		Channel:          dbC,
// 		Queue:            q,
// 		LiveWatchChannel: lwc,
// 	}

// 	we, err := temporal.GetTemporalClient().Client.ExecuteWorkflow(context.Background(), wfOptions, workflows.ArchiveLiveVideoWorkflow, input)
// 	if err != nil {
// 		log.Error().Err(err).Msg("error starting workflow")
// 		return nil, fmt.Errorf("error starting workflow: %v", err)
// 	}

// 	log.Debug().Msgf("workflow id %s started for live stream %s", we.GetID(), live.ID)

// 	// set IDs in queue
// 	_, err = q.Update().SetWorkflowID(we.GetID()).SetWorkflowRunID(we.GetRunID()).Save(context.Background())
// 	if err != nil {
// 		log.Error().Err(err).Msg("error updating queue item")
// 		return nil, fmt.Errorf("error updating queue item: %v", err)
// 	}

// 	// go s.TaskVodCreateFolder(dbC, v, q, true)

// 	return &TwitchVodResponse{
// 		VOD:   v,
// 		Queue: q,
// 	}, nil
// }
