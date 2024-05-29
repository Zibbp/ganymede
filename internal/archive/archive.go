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
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/dto"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/temporal"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
	"github.com/zibbp/ganymede/internal/workflows"
	"go.temporal.io/sdk/client"
)

type Service struct {
	Store          *database.Database
	TwitchService  *twitch.Service
	ChannelService *channel.Service
	VodService     *vod.Service
	QueueService   *queue.Service
}

type TwitchVodResponse struct {
	VOD   *ent.Vod   `json:"vod"`
	Queue *ent.Queue `json:"queue"`
}

func NewService(store *database.Database, twitchService *twitch.Service, channelService *channel.Service, vodService *vod.Service, queueService *queue.Service) *Service {
	return &Service{Store: store, TwitchService: twitchService, ChannelService: channelService, VodService: vodService, QueueService: queueService}
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

func (s *Service) ArchiveTwitchVod(vID string, quality string, chat bool, renderChat bool) (*TwitchVodResponse, error) {
	log.Debug().Msgf("Archiving video %s quality: %s chat: %t render chat: %t", vID, quality, chat, renderChat)
	// Fetch VOD from Twitch API
	tVod, err := s.TwitchService.GetVodByID(vID)
	if err != nil {
		return nil, fmt.Errorf("error fetching twitch vod: %v", err)
	}
	// check if vod is processing
	// the best way I know to check if a vod is processing / still being streamed
	if strings.Contains(tVod.ThumbnailURL, "processing") {
		return nil, fmt.Errorf("vod is still processing")
	}
	// Check if vod is already archived
	vCheck, err := s.VodService.CheckVodExists(tVod.ID)
	if err != nil {
		return nil, fmt.Errorf("error checking if vod exists: %v", err)
	}
	if vCheck {
		return nil, fmt.Errorf("vod already exists")
	}
	// Check if channel exists
	cCheck := s.ChannelService.CheckChannelExists(tVod.UserLogin)
	if !cCheck {
		log.Debug().Msgf("channel does not exist: %s while archiving vod. creating now.", tVod.UserLogin)
		_, err := s.ArchiveTwitchChannel(tVod.UserLogin)
		if err != nil {
			return nil, fmt.Errorf("error creating channel: %v", err)
		}
	}
	// Fetch channel
	dbC, err := s.ChannelService.GetChannelByName(tVod.UserLogin)
	if err != nil {
		return nil, fmt.Errorf("error fetching channel: %v", err)
	}

	// Generate VOD ID for folder name
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error creating vod uuid: %v", err)
	}

	// Storage templates
	folderName, err := GetFolderName(vUUID, tVod)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create folder name, falling back to default")
		folderName = fmt.Sprintf("%s-%s", tVod.ID, vUUID.String())
	}
	fileName, err := GetFileName(vUUID, tVod)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create file name, falling back to default")
		fileName = tVod.ID
	}

	// Sets
	rootVodPath := fmt.Sprintf("/vods/%s/%s", tVod.UserLogin, folderName)
	chatPath := ""
	chatVideoPath := ""
	liveChatPath := ""
	liveChatConvertPath := ""

	if chat {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVodPath, fileName)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVodPath, fileName)
		liveChatPath = fmt.Sprintf("%s/%s-live-chat.json", rootVodPath, fileName)
		liveChatConvertPath = fmt.Sprintf("%s/%s-chat-convert.json", rootVodPath, fileName)
	}
	// Parse new Twitch API duration
	parsedDuration, err := time.ParseDuration(tVod.Duration)
	if err != nil {
		return nil, fmt.Errorf("error parsing duration: %v", err)
	}

	// Parse Twitch date to time.Time
	parsedDate, err := time.Parse(time.RFC3339, tVod.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("error parsing date: %v", err)
	}

	videoExtension := "mp4"

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:                  vUUID,
		ExtID:               tVod.ID,
		Platform:            "twitch",
		Type:                utils.VodType(tVod.Type),
		Title:               tVod.Title,
		Duration:            int(parsedDuration.Seconds()),
		Views:               int(tVod.ViewCount),
		Resolution:          quality,
		Processing:          true,
		ThumbnailPath:       fmt.Sprintf("%s/%s-thumbnail.jpg", rootVodPath, fileName),
		WebThumbnailPath:    fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVodPath, fileName),
		VideoPath:           fmt.Sprintf("%s/%s-video.%s", rootVodPath, fileName, videoExtension),
		ChatPath:            chatPath,
		LiveChatPath:        liveChatPath,
		ChatVideoPath:       chatVideoPath,
		LiveChatConvertPath: liveChatConvertPath,
		InfoPath:            fmt.Sprintf("%s/%s-info.json", rootVodPath, fileName),
		StreamedAt:          parsedDate,
		FolderName:          folderName,
		FileName:            fileName,
		// create temporary paths
		TmpVideoDownloadPath:    fmt.Sprintf("/tmp/%s_%s-video.%s", tVod.ID, vUUID, videoExtension),
		TmpVideoConvertPath:     fmt.Sprintf("/tmp/%s_%s-video-convert.%s", tVod.ID, vUUID, videoExtension),
		TmpChatDownloadPath:     fmt.Sprintf("/tmp/%s_%s-chat.json", tVod.ID, vUUID),
		TmpLiveChatDownloadPath: fmt.Sprintf("/tmp/%s_%s-live-chat.json", tVod.ID, vUUID),
		TmpLiveChatConvertPath:  fmt.Sprintf("/tmp/%s_%s-chat-convert.json", tVod.ID, vUUID),
		TmpChatRenderPath:       fmt.Sprintf("/tmp/%s_%s-chat.mp4", tVod.ID, vUUID),
	}

	if viper.GetBool("archive.save_as_hls") {
		vodDTO.TmpVideoHLSPath = fmt.Sprintf("/tmp/%s_%s-video_hls0", tVod.ID, vUUID)
		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVodPath, fileName)
		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVodPath, fileName, tVod.ID)
	}

	v, err := s.VodService.CreateVod(vodDTO, dbC.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: false}, v.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if !chat {
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
	if !renderChat {
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

	wfOptions := client.StartWorkflowOptions{
		ID:        vUUID.String(),
		TaskQueue: "archive",
	}

	input := dto.ArchiveVideoInput{
		VideoID:      vID,
		Type:         "vod",
		Platform:     "twitch",
		Resolution:   "source",
		DownloadChat: true,
		RenderChat:   true,
		Vod:          v,
		Channel:      dbC,
		Queue:        q,
	}
	we, err := temporal.GetTemporalClient().Client.ExecuteWorkflow(context.Background(), wfOptions, workflows.ArchiveVideoWorkflow, input)
	if err != nil {
		log.Error().Err(err).Msg("error starting workflow")
		return nil, fmt.Errorf("error starting workflow: %v", err)
	}

	log.Debug().Msgf("workflow id %s started for vod %s", we.GetID(), vID)

	return &TwitchVodResponse{
		VOD:   v,
		Queue: q,
	}, nil
}

func (s *Service) ArchiveTwitchLive(lwc *ent.Live, live twitch.Live) (*TwitchVodResponse, error) {
	// Check if channel exists
	cCheck := s.ChannelService.CheckChannelExists(live.UserLogin)
	if !cCheck {
		log.Debug().Msgf("channel does not exist: %s while archiving live stream. creating now.", live.UserLogin)
		_, err := s.ArchiveTwitchChannel(live.UserLogin)
		if err != nil {
			return nil, fmt.Errorf("error creating channel: %v", err)
		}
	}
	// Fetch channel
	dbC, err := s.ChannelService.GetChannelByName(live.UserLogin)
	if err != nil {
		return nil, fmt.Errorf("error fetching channel: %v", err)
	}

	// Generate VOD ID for folder name
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error creating vod uuid: %v", err)
	}

	// Create vodDto for storage templates
	tVodDto := twitch.Vod{
		ID:        live.ID,
		UserLogin: live.UserLogin,
		Title:     live.Title,
		Type:      "live",
		CreatedAt: live.StartedAt,
	}
	folderName, err := GetFolderName(vUUID, tVodDto)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create folder name, falling back to default")
		folderName = fmt.Sprintf("%s-%s", tVodDto.ID, vUUID.String())
	}
	fileName, err := GetFileName(vUUID, tVodDto)
	if err != nil {
		log.Error().Err(err).Msg("error using template to create file name, falling back to default")
		fileName = tVodDto.ID
	}

	// Sets
	rootVodPath := fmt.Sprintf("/vods/%s/%s", live.UserLogin, folderName)
	chatPath := ""
	chatVideoPath := ""
	liveChatPath := ""
	liveChatConvertPath := ""

	if lwc.ArchiveChat {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVodPath, fileName)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVodPath, fileName)
		liveChatPath = fmt.Sprintf("%s/%s-live-chat.json", rootVodPath, fileName)
		liveChatConvertPath = fmt.Sprintf("%s/%s-chat-convert.json", rootVodPath, fileName)
	}

	videoExtension := "mp4"

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:                  vUUID,
		ExtID:               live.ID,
		Platform:            "twitch",
		Type:                utils.VodType("live"),
		Title:               live.Title,
		Duration:            1,
		Views:               1,
		Resolution:          lwc.Resolution,
		Processing:          true,
		ThumbnailPath:       fmt.Sprintf("%s/%s-thumbnail.jpg", rootVodPath, fileName),
		WebThumbnailPath:    fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVodPath, fileName),
		VideoPath:           fmt.Sprintf("%s/%s-video.%s", rootVodPath, fileName, videoExtension),
		ChatPath:            chatPath,
		LiveChatPath:        liveChatPath,
		ChatVideoPath:       chatVideoPath,
		LiveChatConvertPath: liveChatConvertPath,
		InfoPath:            fmt.Sprintf("%s/%s-info.json", rootVodPath, fileName),
		StreamedAt:          time.Now(),
		FolderName:          folderName,
		FileName:            fileName,
		// create temporary paths
		TmpVideoDownloadPath:    fmt.Sprintf("/tmp/%s_%s-video.%s", live.ID, vUUID, videoExtension),
		TmpVideoConvertPath:     fmt.Sprintf("/tmp/%s_%s-video-convert.%s", live.ID, vUUID, videoExtension),
		TmpChatDownloadPath:     fmt.Sprintf("/tmp/%s_%s-chat.json", live.ID, vUUID),
		TmpLiveChatDownloadPath: fmt.Sprintf("/tmp/%s_%s-live-chat.json", live.ID, vUUID),
		TmpLiveChatConvertPath:  fmt.Sprintf("/tmp/%s_%s-chat-convert.json", live.ID, vUUID),
		TmpChatRenderPath:       fmt.Sprintf("/tmp/%s_%s-chat.mp4", live.ID, vUUID),
	}

	if viper.GetBool("archive.save_as_hls") {
		vodDTO.TmpVideoHLSPath = fmt.Sprintf("/tmp/%s_%s-video_hls0", live.ID, vUUID)
		vodDTO.VideoHLSPath = fmt.Sprintf("%s/%s-video_hls", rootVodPath, fileName)
		vodDTO.VideoPath = fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", rootVodPath, fileName, live.ID)
	}

	v, err := s.VodService.CreateVod(vodDTO, dbC.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(queue.Queue{LiveArchive: true}, v.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if !lwc.ArchiveChat {
		_, err := q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating queue item: %v", err)
		}

		_, err = v.Update().SetChatPath("").SetChatVideoPath("").Save(context.Background())
		if err != nil {
			return nil, fmt.Errorf("error updating vod: %v", err)
		}

	}

	if !lwc.RenderChat {
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

	wfOptions := client.StartWorkflowOptions{
		ID:        vUUID.String(),
		TaskQueue: "archive",
	}

	input := dto.ArchiveVideoInput{
		VideoID:          live.ID,
		Type:             "live",
		Platform:         "twitch",
		Resolution:       lwc.Resolution,
		DownloadChat:     lwc.ArchiveChat,
		RenderChat:       lwc.RenderChat,
		Vod:              v,
		Channel:          dbC,
		Queue:            q,
		LiveWatchChannel: lwc,
	}

	we, err := temporal.GetTemporalClient().Client.ExecuteWorkflow(context.Background(), wfOptions, workflows.ArchiveLiveVideoWorkflow, input)
	if err != nil {
		log.Error().Err(err).Msg("error starting workflow")
		return nil, fmt.Errorf("error starting workflow: %v", err)
	}

	log.Debug().Msgf("workflow id %s started for live stream %s", we.GetID(), live.ID)

	// set IDs in queue
	_, err = q.Update().SetWorkflowID(we.GetID()).SetWorkflowRunID(we.GetRunID()).Save(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error updating queue item")
		return nil, fmt.Errorf("error updating queue item: %v", err)
	}

	// go s.TaskVodCreateFolder(dbC, v, q, true)

	return &TwitchVodResponse{
		VOD:   v,
		Queue: q,
	}, nil
}
