package archive

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	queue2 "github.com/zibbp/ganymede/ent/queue"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/notification"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
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
	var chatPath string
	var chatVideoPath string
	if chat {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVodPath, fileName)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVodPath, fileName)
	} else {
		chatPath = ""
		chatVideoPath = ""
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

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:               vUUID,
		ExtID:            tVod.ID,
		Platform:         "twitch",
		Type:             utils.VodType(tVod.Type),
		Title:            tVod.Title,
		Duration:         int(parsedDuration.Seconds()),
		Views:            int(tVod.ViewCount),
		Resolution:       quality,
		Processing:       true,
		ThumbnailPath:    fmt.Sprintf("%s/%s-thumbnail.jpg", rootVodPath, fileName),
		WebThumbnailPath: fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVodPath, fileName),
		VideoPath:        fmt.Sprintf("%s/%s-video.mp4", rootVodPath, fileName),
		ChatPath:         chatPath,
		ChatVideoPath:    chatVideoPath,
		InfoPath:         fmt.Sprintf("%s/%s-info.json", rootVodPath, fileName),
		StreamedAt:       parsedDate,
		FolderName:       folderName,
		FileName:         fileName,
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
		q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).SaveX(context.Background())
		v.Update().SetChatPath("").SetChatVideoPath("").SaveX(context.Background())
	}

	// If render chat is disabled update queue
	if !renderChat {
		q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).SaveX(context.Background())
		v.Update().SetChatVideoPath("").SaveX(context.Background())
	}

	// Re-query queue from DB for updated values
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		return nil, fmt.Errorf("error fetching queue item: %v", err)
	}

	// Get max active queue items from config
	maxActiveQueueItems := viper.GetInt("active_queue_items")

	// Get all queue items that are not on hold
	qItems, err := s.Store.Client.Queue.Query().Where(queue2.Processing(true)).Where(queue2.OnHold(false)).All(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error fetching queue items: %v", err)
	}
	if len(qItems)-1 >= maxActiveQueueItems {
		// If there are more than X active items in queue set new queue item to on hold
		log.Debug().Msgf("more than %d active items in queue. setting new queue item %s to on hold", maxActiveQueueItems, q.ID)
		q.Update().SetOnHold(true).SaveX(context.Background())

		return &TwitchVodResponse{
			VOD:   v,
			Queue: q,
		}, nil
	}

	go s.TaskVodCreateFolder(dbC, v, q, true)

	return &TwitchVodResponse{
		VOD:   v,
		Queue: q,
	}, nil
}

func (s *Service) CheckOnHold() {
	// Get max active queue items from config
	maxActiveQueueItems := viper.GetInt("active_queue_items")

	// Get all queue items that are not on hold
	qItems, err := s.Store.Client.Queue.Query().Where(queue2.Processing(true)).Where(queue2.OnHold(false)).All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error fetching queue items")
		return
	}
	if len(qItems) >= maxActiveQueueItems {
		// Do nothing as queue items are still working
		log.Debug().Msgf("more than %d active items in queue. doing nothing", maxActiveQueueItems)
		return
	}
	// Get all queue items that are on hold oldest to newest
	qItems, err = s.Store.Client.Queue.Query().Where(queue2.Processing(true)).Where(queue2.OnHold(true)).WithVod().Order(ent.Asc(queue2.FieldCreatedAt)).All(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error fetching queue items")
		return
	}
	if len(qItems) == 0 {
		// No queue items are on hold
		log.Debug().Msg("no queue items are on hold")
		return
	}
	// Get first queue item
	qItem := qItems[0]

	// Get VOD
	v, err := s.VodService.GetVodWithChannel(qItem.Edges.Vod.ID)
	if err != nil {
		log.Error().Err(err).Msgf("error getting vod: %v", err)
	}

	// Get channel
	dbC, err := s.ChannelService.GetChannelByName(v.Edges.Channel.Name)
	if err != nil {
		log.Error().Err(err).Msgf("error getting channel: %v", err)
	}

	// Get queue item
	q, err := s.QueueService.GetQueueItem(qItem.ID)
	if err != nil {
		log.Error().Err(err).Msgf("error getting queue item: %v", err)
	}

	// Update queue item
	q.Update().SetOnHold(false).SaveX(context.Background())

	// Start queue item
	go s.TaskVodCreateFolder(dbC, v, q, true)

}

func (s *Service) ArchiveTwitchLive(lwc *ent.Live, ts twitch.Live) (*TwitchVodResponse, error) {
	// Check if channel exists
	cCheck := s.ChannelService.CheckChannelExists(ts.UserLogin)
	if !cCheck {
		log.Debug().Msgf("channel does not exist: %s while archiving live stream. creating now.", ts.UserLogin)
		_, err := s.ArchiveTwitchChannel(ts.UserLogin)
		if err != nil {
			return nil, fmt.Errorf("error creating channel: %v", err)
		}
	}
	// Fetch channel
	dbC, err := s.ChannelService.GetChannelByName(ts.UserLogin)
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
		ID:        ts.ID,
		UserLogin: ts.UserLogin,
		Title:     ts.Title,
		Type:      "live",
		CreatedAt: ts.StartedAt,
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
	rootVodPath := fmt.Sprintf("/vods/%s/%s", ts.UserLogin, folderName)
	var chatPath string
	var chatVideoPath string
	if lwc.ArchiveChat {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVodPath, fileName)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVodPath, fileName)
	} else {
		chatPath = ""
		chatVideoPath = ""
	}

	// Create VOD in DB
	vodDTO := vod.Vod{
		ID:               vUUID,
		ExtID:            ts.ID,
		Platform:         "twitch",
		Type:             utils.VodType("live"),
		Title:            ts.Title,
		Duration:         1,
		Views:            1,
		Resolution:       lwc.Resolution,
		Processing:       true,
		ThumbnailPath:    fmt.Sprintf("%s/%s-thumbnail.jpg", rootVodPath, fileName),
		WebThumbnailPath: fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVodPath, fileName),
		VideoPath:        fmt.Sprintf("%s/%s-video.mp4", rootVodPath, fileName),
		ChatPath:         chatPath,
		ChatVideoPath:    chatVideoPath,
		InfoPath:         fmt.Sprintf("%s/%s-info.json", rootVodPath, fileName),
		StreamedAt:       time.Now(),
		FolderName:       folderName,
		FileName:         fileName,
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
		q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).SaveX(context.Background())
		v.Update().SetChatPath("").SetChatVideoPath("").SaveX(context.Background())
	}

	if !lwc.RenderChat {
		q.Update().SetTaskChatRender(utils.Success).SetRenderChat(false).SaveX(context.Background())
		v.Update().SetChatVideoPath("").SaveX(context.Background())
	}

	// Re-query queue from DB for updated values
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		return nil, fmt.Errorf("error fetching queue item: %v", err)
	}

	go s.TaskVodCreateFolder(dbC, v, q, true)

	return &TwitchVodResponse{
		VOD:   v,
		Queue: q,
	}, nil
}

func (s *Service) RestartTask(c echo.Context, qID uuid.UUID, task string, cont bool) error {
	q, err := s.QueueService.GetQueueItem(qID)
	if err != nil {
		return err
	}
	v, err := s.VodService.GetVodWithChannel(q.Edges.Vod.ID)
	if err != nil {
		return err
	}
	ch, err := s.ChannelService.GetChannel(v.Edges.Channel.ID)
	if err != nil {
		return err
	}

	log.Debug().Msgf("restarting task: %s for queue id: continue: ", task)

	switch task {
	case "vod_create_folder":
		go s.TaskVodCreateFolder(ch, v, q, cont)
	case "vod_download_thumbnail":
		if q.LiveArchive {
			go s.TaskVodDownloadLiveThumbnail(ch, v, q, cont)
		} else {
			go s.TaskVodDownloadThumbnail(ch, v, q, cont)
		}
	case "vod_save_info":
		if q.LiveArchive {
			err = s.TaskVodSaveLiveInfo(ch, v, q, cont)
			if err != nil {
				log.Error().Err(err).Msg("error saving live info")
				q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
				s.TaskError(ch, v, q, "vod_save_info")
				return err
			}
			q.Update().SetTaskVodSaveInfo(utils.Success).SaveX(context.Background())
		} else {
			go s.TaskVodSaveInfo(ch, v, q, cont)
		}
	case "video_download":
		go s.TaskVideoDownload(ch, v, q, cont)
	case "video_convert":
		go s.TaskVideoConvert(ch, v, q, cont)
	case "video_move":
		go s.TaskVideoMove(ch, v, q, cont)
	case "chat_convert":
		go s.TaskChatConvertRestart(ch, v, q, cont)
	case "chat_download":
		go s.TaskChatDownload(ch, v, q, cont)
	case "chat_render":
		go s.TaskChatRender(ch, v, q, cont)
	case "chat_move":
		if q.LiveArchive {
			go s.TaskLiveChatMove(ch, v, q, cont)
		} else {
			go s.TaskChatMove(ch, v, q, cont)
		}
	default:
		return fmt.Errorf("task not found")
	}

	return nil
}

func (s *Service) TaskVodCreateFolder(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task vod create folder for vod %s", v.ID)
	q.Update().SetTaskVodCreateFolder(utils.Running).SaveX(context.Background())
	// Create folder
	err := utils.CreateFolder(fmt.Sprintf("%s/%s", ch.Name, v.FolderName))
	if err != nil {
		log.Error().Err(err).Msg("error creating vod folder")
		q.Update().SetTaskVodCreateFolder(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_create_folder")
		return
	}
	q.Update().SetTaskVodCreateFolder(utils.Success).SaveX(context.Background())

	if cont {
		if q.LiveArchive {
			go s.TaskVodDownloadLiveThumbnail(ch, v, q, true)
		} else {
			go s.TaskVodDownloadThumbnail(ch, v, q, true)
		}

	}
}

func (s *Service) TaskVodDownloadLiveThumbnail(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task vod download thumbnail for live stream %s", v.ID)
	_, err := q.Update().SetTaskVodDownloadThumbnail(utils.Running).Save(context.Background())
	if err != nil {
		if ent.IsNotFound(err) {
			log.Warn().Msgf("queue item not found while updating task vod download thumbnail for live stream %s", v.ID)
			return
		}
		log.Error().Err(err).Msgf("error updating task vod download thumbnail for live stream %s", v.ID)
		return
	}

	// Fetch Stream from Twitch for thumbnails
	stream, err := s.TwitchService.GetStreams(fmt.Sprintf("?user_login=%s", ch.Name))
	if err != nil {
		log.Error().Err(err).Msg("error fetching twitch stream")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_download_thumbnail")
		return
	}
	if len(stream.Data) == 0 {
		log.Error().Msg("no stream found")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_download_thumbnail")
		if cont {
			// Refresh thumbnails for live stream after 30 minutes
			go s.RefreshLiveThumbnails(ch, v, q)
			// Proceed with task
			err = s.TaskVodSaveLiveInfo(ch, v, q, true)
			if err != nil {
				log.Error().Err(err).Msg("error saving live info")
				q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
				s.TaskError(ch, v, q, "vod_save_info")
			}
		}
		return
	}
	tVod := stream.Data[0]
	fullResThumbnailUrl := strings.ReplaceAll(tVod.ThumbnailURL, "{width}", "1920")
	fullResThumbnailUrl = strings.ReplaceAll(fullResThumbnailUrl, "{height}", "1080")

	webResThumbnailUrl := strings.ReplaceAll(tVod.ThumbnailURL, "{width}", "640")
	webResThumbnailUrl = strings.ReplaceAll(webResThumbnailUrl, "{height}", "360")

	// Download full resolution thumbnail
	err = utils.DownloadFile(fullResThumbnailUrl, fmt.Sprintf("%s/%s", ch.Name, v.FolderName), fmt.Sprintf("%s-thumbnail.jpg", v.FileName))
	if err != nil {
		log.Error().Err(err).Msg("error downloading thumbnail")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_download_thumbnail")
	}
	// Download web resolution thumbnail
	err = utils.DownloadFile(webResThumbnailUrl, fmt.Sprintf("%s/%s", ch.Name, v.FolderName), fmt.Sprintf("%s-web_thumbnail.jpg", v.FileName))
	if err != nil {
		log.Error().Err(err).Msg("error downloading thumbnail")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_download_thumbnail")
	}

	if err == nil {
		q.Update().SetTaskVodDownloadThumbnail(utils.Success).SaveX(context.Background())
	}

	if cont {
		// Refresh thumbnails for live stream after 30 minutes
		go s.RefreshLiveThumbnails(ch, v, q)
		// Proceed with task
		err = s.TaskVodSaveLiveInfo(ch, v, q, true)
		if err != nil {
			log.Error().Err(err).Msg("error saving live info")
			q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "vod_save_info")
		}
	}
}

func (s *Service) RefreshLiveThumbnails(ch *ent.Channel, v *ent.Vod, q *ent.Queue) {
	log.Debug().Msg("refresh live thumbnails called...sleeping for 20 minutes")
	time.Sleep(20 * time.Minute)
	log.Debug().Msg("refresh live thumbnails sleep done")
	go s.TaskVodDownloadLiveThumbnail(ch, v, q, false)
}

func (s *Service) TaskVodDownloadThumbnail(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task vod download thumbnail for vod %s", v.ID)
	q.Update().SetTaskVodDownloadThumbnail(utils.Running).SaveX(context.Background())

	// Fetch VOD from Twitch for thumbnails
	tVod, err := s.TwitchService.GetVodByID(v.ExtID)
	if err != nil {
		log.Error().Err(err).Msg("error fetching twitch vod")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_download_thumbnail")
		return
	}

	fullResThumbnailUrl := strings.ReplaceAll(tVod.ThumbnailURL, "%{width}", "1920")
	fullResThumbnailUrl = strings.ReplaceAll(fullResThumbnailUrl, "%{height}", "1080")

	webResThumbnailUrl := strings.ReplaceAll(tVod.ThumbnailURL, "%{width}", "640")
	webResThumbnailUrl = strings.ReplaceAll(webResThumbnailUrl, "%{height}", "360")

	// Download full resolution thumbnail
	err = utils.DownloadFile(fullResThumbnailUrl, fmt.Sprintf("%s/%s", ch.Name, v.FolderName), fmt.Sprintf("%s-thumbnail.jpg", v.FileName))
	if err != nil {
		log.Error().Err(err).Msg("error downloading thumbnail")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_download_thumbnail")
		return
	}
	// Download web resolution thumbnail
	err = utils.DownloadFile(webResThumbnailUrl, fmt.Sprintf("%s/%s", ch.Name, v.FolderName), fmt.Sprintf("%s-web_thumbnail.jpg", v.FileName))
	if err != nil {
		log.Error().Err(err).Msg("error downloading thumbnail")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_download_thumbnail")
		return
	}

	q.Update().SetTaskVodDownloadThumbnail(utils.Success).SaveX(context.Background())

	if cont {
		go s.TaskVodSaveInfo(ch, v, q, true)
	}
}

func (s *Service) TaskVodSaveLiveInfo(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) error {
	log.Debug().Msgf("starting task vod save info for vod %s", v.ID)
	q.Update().SetTaskVodSaveInfo(utils.Running).SaveX(context.Background())

	// Fetch VOD from Twitch
	// Fetch Stream from Twitch for thumbnails
	stream, err := s.TwitchService.GetStreams(fmt.Sprintf("?user_login=%s", ch.Name))
	if err != nil {
		log.Error().Err(err).Msg("error fetching twitch vod")
		q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_save_info")
		return err
	}

	if len(stream.Data) == 0 {
		return fmt.Errorf("stream data is empty")
	}

	tVod := stream.Data[0]

	err = utils.WriteJson(tVod, fmt.Sprintf("%s/%s", ch.Name, v.FolderName), fmt.Sprintf("%s-info.json", v.FileName))
	if err != nil {
		log.Error().Err(err).Msg("error saving info")
		q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_save_info")
		return err
	}
	q.Update().SetTaskVodSaveInfo(utils.Success).SaveX(context.Background())
	if cont {

		busC := make(chan bool)

		startChatDownloadChannel := make(chan bool)

		go s.TaskLiveVideoDownload(ch, v, q, true, busC, startChatDownloadChannel)
		//	Check if chat download task is set to success
		if q.TaskChatDownload == utils.Pending {
			go s.TaskLiveChatDownload(ch, v, q, true, busC, startChatDownloadChannel, true)
		}
	}
	return nil
}

func (s *Service) TaskVodSaveInfo(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task vod save info for vod %s", v.ID)
	q.Update().SetTaskVodSaveInfo(utils.Running).SaveX(context.Background())

	// Fetch VOD from Twitch
	tVod, err := s.TwitchService.GetVodByID(v.ExtID)
	if err != nil {
		log.Error().Err(err).Msg("error fetching twitch vod")
		q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_save_info")
		return
	}

	err = utils.WriteJson(tVod, fmt.Sprintf("%s/%s", ch.Name, v.FolderName), fmt.Sprintf("%s-info.json", v.FileName))
	if err != nil {
		log.Error().Err(err).Msg("error saving info")
		q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "vod_save_info")
		return
	}
	q.Update().SetTaskVodSaveInfo(utils.Success).SaveX(context.Background())
	if cont {
		go s.TaskVideoDownload(ch, v, q, true)
		//	Check if chat download task is set to success
		if q.TaskChatDownload == utils.Pending {
			go s.TaskChatDownload(ch, v, q, true)
		}
	}
}

func (s *Service) TaskLiveVideoDownload(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool, busC chan bool, startChatDownloadChannel chan bool) {
	log.Debug().Msgf("starting task video download for live stream %s", v.ID)
	q.Update().SetTaskVideoDownload(utils.Running).SaveX(context.Background())

	err := exec.DownloadTwitchLiveVideo(v, ch, startChatDownloadChannel)
	if err != nil {
		log.Error().Err(err).Msg("error downloading live video")
		q.Update().SetTaskVideoDownload(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "video_download")
		return
	}

	// Send kill command to chat download
	if q.TaskChatDownload != utils.Success {
		busC <- true
	}

	q.Update().SetTaskVideoDownload(utils.Success).SaveX(context.Background())

	// Set live watch channel to not live
	live, err := ch.QueryLive().Only(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("error getting live")
	}
	if err == nil {
		live.Update().SetIsLive(false).SaveX(context.Background())
	}

	// Update video duration with duration from video
	duration, err := exec.GetVideoDuration(fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error getting video duration")

	}
	if err == nil {
		v.Update().SetDuration(duration).SaveX(context.Background())
	}

	//Always invoke task video convert if video download was successful
	go s.TaskVideoConvert(ch, v, q, true)

}

func (s *Service) TaskVideoDownload(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task video download for vod %s", v.ID)
	q.Update().SetTaskVideoDownload(utils.Running).SaveX(context.Background())

	err := exec.DownloadTwitchVodVideo(v)
	if err != nil {
		log.Error().Err(err).Msg("error downloading video")
		q.Update().SetTaskVideoDownload(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "video_download")
		return
	}

	q.Update().SetTaskVideoDownload(utils.Success).SaveX(context.Background())

	// Always invoke task video convert if video download was successful
	go s.TaskVideoConvert(ch, v, q, true)

}

func (s *Service) TaskVideoConvert(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task video convert for vod %s", v.ID)
	q.Update().SetTaskVideoConvert(utils.Running).SaveX(context.Background())

	err := exec.ConvertTwitchVodVideo(v)
	if err != nil {
		log.Error().Err(err).Msg("error converting video")
		q.Update().SetTaskVideoConvert(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "video_convert")
		return
	}

	// Check if video should be saved as hls
	if viper.GetBool("archive.save_as_hls") {
		err = exec.ConvertToHLS(v)
		if err != nil {
			log.Error().Err(err).Msg("error converting video to hls")
			q.Update().SetTaskVideoConvert(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "video_convert")
			return
		}
	}

	q.Update().SetTaskVideoConvert(utils.Success).SaveX(context.Background())

	// Always invoke task video move if video convert was successful
	go s.TaskVideoMove(ch, v, q, true)
}

func (s *Service) TaskVideoMove(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task video move for vod %s", v.ID)
	q.Update().SetTaskVideoMove(utils.Running).SaveX(context.Background())

	// Check if video is saved as HLS
	if viper.GetBool("archive.save_as_hls") {
		// Move HLS folder to vod folder
		sourcePath := fmt.Sprintf("/tmp/%s_%s-video_hls0", v.ExtID, v.ID)
		destPath := fmt.Sprintf("/vods/%s/%s/%s-video_hls", ch.Name, v.FolderName, v.FileName)
		err := utils.MoveFolder(sourcePath, destPath)
		if err != nil {
			log.Error().Err(err).Msg("error moving video hls directory")
			q.Update().SetTaskVideoMove(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "video_move")
			return
		}
		// Update video path to hls path
		v.Update().SetVideoPath(fmt.Sprintf("/vods/%s/%s/%s-video_hls/%s-video.m3u8", ch.Name, v.FolderName, v.FileName, v.ExtID)).SaveX(context.Background())
	} else {
		sourcePath := fmt.Sprintf("/tmp/%s_%s-video-convert.mp4", v.ExtID, v.ID)
		destPath := fmt.Sprintf("/vods/%s/%s/%s-video.mp4", ch.Name, v.FolderName, v.FileName)

		err := utils.MoveFile(sourcePath, destPath)
		if err != nil {
			log.Error().Err(err).Msg("error moving video")
			q.Update().SetTaskVideoMove(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "video_move")
			return
		}
	}

	// Clean up files
	// Delete source file
	err := utils.DeleteFile(fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID))
	if err != nil {
		log.Info().Err(err).Msgf("error deleting source file for vod %s", v.ID)
	}
	// Ensure the converted file is deleted
	err = utils.DeleteFile(fmt.Sprintf("/tmp/%s_%s-video-convert.mp4", v.ExtID, v.ID))
	if err != nil {
		log.Debug().Msgf("error deleting converted file for vod %s", v.ID)
	}

	q.Update().SetTaskVideoMove(utils.Success).SaveX(context.Background())

	// Set video as complete
	q.Update().SetVideoProcessing(false).SaveX(context.Background())

	// Check if all task are done
	if q.LiveArchive {
		go s.CheckIfLiveTasksAreDone(ch, v, q)
	} else {
		go s.CheckIfTasksAreDone(ch, v, q)
	}
}

func (s *Service) TaskChatDownload(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task chat download for vod %s", v.ID)
	q.Update().SetTaskChatDownload(utils.Running).SaveX(context.Background())

	err := exec.DownloadTwitchVodChat(v)
	if err != nil {
		log.Error().Err(err).Msg("error downloading chat")
		q.Update().SetTaskChatDownload(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_download")
		return
	}

	q.Update().SetTaskChatDownload(utils.Success).SaveX(context.Background())

	// copy chat json
	sourcePath := fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-chat.json", ch.Name, v.FolderName, v.FileName)

	err = utils.CopyFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error copying chat")
	}

	if cont {
		go s.TaskChatRender(ch, v, q, true)
	}
}

func (s *Service) TaskLiveChatDownload(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool, busC chan bool, startChatDownloadChannel chan bool, waitForVideo bool) {
	log.Debug().Msgf("starting task chat download for live stream %s", v.ID)
	q.Update().SetTaskChatDownload(utils.Running).SaveX(context.Background())

	err := exec.DownloadTwitchLiveChat(v, ch, q, busC, startChatDownloadChannel, waitForVideo)
	if err != nil {
		log.Error().Err(err).Msg("error downloading live chat")
		q.Update().SetTaskChatDownload(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_download")
		return
	}

	q.Update().SetTaskChatDownload(utils.Success).SaveX(context.Background())

	// copy live chat
	sourcePath := fmt.Sprintf("/tmp/%s_%s-live-chat.json", v.ExtID, v.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-live-chat.json", ch.Name, v.FolderName, v.FileName)

	err = utils.CopyFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving live chat")
	}

	// Always convert live chat to vod chat
	go s.TaskLiveChatConvert(ch, v, q, true)

}

func (s *Service) TaskChatConvertRestart(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	// Check if chat file exists
	chatPath := fmt.Sprintf("/tmp/%s_%s-live-chat.json", v.ExtID, v.ID)
	if !utils.FileExists(chatPath) {
		storageChatPath := fmt.Sprintf("/tmp/%s_%s-live-chat.json", v.ExtID, v.ID)
		if utils.FileExists(storageChatPath) {
			err := utils.CopyFile(storageChatPath, chatPath)
			if err != nil {
				log.Error().Err(err).Msg("error copying live chat")
			}
		} else {
			log.Error().Msgf("chat file does not exist %s", chatPath)
			q.Update().SetTaskChatConvert(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "chat_convert")
			return
		}
	}

	go s.TaskLiveChatConvert(ch, v, q, cont)
}

func (s *Service) TaskLiveChatConvert(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task chat convert for vod %s", v.ID)
	q.Update().SetTaskChatConvert(utils.Running).SaveX(context.Background())

	// Check if chat file exists
	chatPath := fmt.Sprintf("/tmp/%s_%s-live-chat.json", v.ExtID, v.ID)
	if !utils.FileExists(chatPath) {
		log.Debug().Msgf("chat file does not exist %s - this means there were no chat messages - setting chat to complete", chatPath)
		// Set queue chat task to complete
		q.Update().SetChatProcessing(false).SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).SaveX(context.Background())
		// Set VOD chat to empty
		v.Update().SetChatPath("").SetChatVideoPath("").SaveX(context.Background())
		// Check if all task are done
		go s.CheckIfLiveTasksAreDone(ch, v, q)
		return
	}

	// Fetch streamer from Twitch API for their user ID
	streamer, err := twitch.API.GetUserByLogin(ch.Name)
	if err != nil {
		log.Error().Err(err).Msg("error getting streamer from Twitch API")
		q.Update().SetTaskChatConvert(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_convert")
		return
	}
	cID, err := strconv.Atoi(streamer.ID)
	if err != nil {
		log.Error().Err(err).Msg("error converting streamer ID to int")
		q.Update().SetTaskChatConvert(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_convert")
		return
	}

	// Get queue item (refresh)
	q, err = s.QueueService.GetQueueItem(q.ID)
	if err != nil {
		log.Error().Err(err).Msg("error getting queue item")
		q.Update().SetTaskChatConvert(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_convert")
		return
	}

	err = utils.ConvertTwitchLiveChatToVodChat(fmt.Sprintf("/tmp/%s_%s-live-chat.json", v.ExtID, v.ID), ch.Name, v.ID.String(), v.ExtID, cID, q.ChatStart)
	if err != nil {
		log.Error().Err(err).Msg("error converting chat")
		q.Update().SetTaskChatConvert(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_convert")
		log.Info().Msgf("livestream chat task failed - setting vod to processed so it can be viewed")
		v.Update().SetProcessing(false).SaveX(context.Background())
		return
	}

	// TwitchDownloader "chatupdate"
	// Embeds emotes and badges into the chat file
	err = exec.TwitchChatUpdate(v)
	if err != nil {
		log.Error().Err(err).Msg("error updating chat")
		q.Update().SetTaskChatConvert(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_convert")
		log.Info().Msgf("livestream chat task failed - setting vod to processed so it can be viewed")
		v.Update().SetProcessing(false).SaveX(context.Background())
		return
	}

	q.Update().SetTaskChatConvert(utils.Success).SaveX(context.Background())

	// copy converted chat
	sourcePath := fmt.Sprintf("/tmp/%s_%s-chat-convert.json", v.ExtID, v.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-chat-convert.json", ch.Name, v.FolderName, v.FileName)

	err = utils.CopyFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error copying chat convert")
	}

	// Always render chat
	go s.TaskChatRender(ch, v, q, true)
}

func (s *Service) TaskChatRender(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	var renderContinue bool
	if !q.RenderChat {
		log.Debug().Msgf("skipping chat render for vod %s as it is disabled", v.ID)
		renderContinue = true
	} else {
		log.Debug().Msgf("starting task chat render for vod %s", v.ID)
		q.Update().SetTaskChatRender(utils.Running).SaveX(context.Background())

		err, rCont := exec.RenderTwitchVodChat(v, q)
		if err != nil {
			log.Error().Err(err).Msg("error rendering chat")
			q.Update().SetTaskChatRender(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "chat_render")
			if q.LiveArchive {
				log.Info().Msgf("livestream chat task failed - setting vod to processed so it can be viewed")
				v.Update().SetProcessing(false).SaveX(context.Background())
			}
			return
		}
		renderContinue = rCont

		q.Update().SetTaskChatRender(utils.Success).SaveX(context.Background())
	}

	// Always move chat if render was successful
	if renderContinue {
		if q.LiveArchive {
			go s.TaskLiveChatMove(ch, v, q, true)
		} else {
			go s.TaskChatMove(ch, v, q, true)
		}
	} else {
		// Check if all task are done
		go s.CheckIfTasksAreDone(ch, v, q)
	}
}

func (s *Service) TaskChatMove(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task chat move for vod %s", v.ID)
	q.Update().SetTaskChatMove(utils.Running).SaveX(context.Background())

	// Chat JSON
	sourcePath := fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-chat.json", ch.Name, v.FolderName, v.FileName)

	err := utils.MoveFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving chat")
		q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_move")
		return
	}
	if q.RenderChat {
		// Chat Video
		sourcePath = fmt.Sprintf("/tmp/%s_%s-chat.mp4", v.ExtID, v.ID)
		destPath = fmt.Sprintf("/vods/%s/%s/%s-chat.mp4", ch.Name, v.FolderName, v.FileName)

		err = utils.MoveFile(sourcePath, destPath)
		if err != nil {
			log.Error().Err(err).Msg("error moving chat")
			q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "chat_move")
			return
		}
	}

	q.Update().SetTaskChatMove(utils.Success).SaveX(context.Background())

	// Set chat as complete
	q.Update().SetChatProcessing(false).SaveX(context.Background())

	// Check if all task are done
	go s.CheckIfTasksAreDone(ch, v, q)
}

func (s *Service) TaskLiveChatMove(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task chat move for live stream %s", v.ID)
	q.Update().SetTaskChatMove(utils.Running).SaveX(context.Background())

	// live chat JSON
	sourcePath := fmt.Sprintf("/tmp/%s_%s-live-chat.json", v.ExtID, v.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-live-chat.json", ch.Name, v.FolderName, v.FileName)

	err := utils.MoveFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving live chat")
		q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_move")
		return
	}

	// converted chat JSON
	sourcePath = fmt.Sprintf("/tmp/%s_%s-chat-convert.json", v.ExtID, v.ID)
	destPath = fmt.Sprintf("/vods/%s/%s/%s-chat-convert.json", ch.Name, v.FolderName, v.FileName)

	err = utils.MoveFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving chat convert")
		q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_move")
		return
	}

	// parsed chat JSON
	sourcePath = fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID)
	destPath = fmt.Sprintf("/vods/%s/%s/%s-chat.json", ch.Name, v.FolderName, v.FileName)

	err = utils.MoveFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving live parsed chat")
		q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
		s.TaskError(ch, v, q, "chat_move")
		return
	}

	if q.RenderChat {
		// Chat Video
		sourcePath = fmt.Sprintf("/tmp/%s_%s-chat.mp4", v.ExtID, v.ID)
		destPath = fmt.Sprintf("/vods/%s/%s/%s-chat.mp4", ch.Name, v.FolderName, v.FileName)

		err = utils.MoveFile(sourcePath, destPath)
		if err != nil {
			log.Error().Err(err).Msg("error moving chat")
			q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
			s.TaskError(ch, v, q, "chat_move")
			return
		}
	}

	q.Update().SetTaskChatMove(utils.Success).SaveX(context.Background())

	// Set chat as complete
	q.Update().SetChatProcessing(false).SaveX(context.Background())

	// Check if all task are done
	go s.CheckIfLiveTasksAreDone(ch, v, q)
}

func (s *Service) CheckIfTasksAreDone(ch *ent.Channel, v *ent.Vod, qO *ent.Queue) {
	q, err := s.QueueService.ArchiveGetQueueItem(qO.ID)
	if err != nil {
		log.Error().Err(err).Msg("error getting queue item")
		return
	}
	if q.TaskVideoDownload == utils.Success && q.TaskVideoConvert == utils.Success && q.TaskVideoMove == utils.Success && q.TaskChatDownload == utils.Success && q.TaskChatRender == utils.Success && q.TaskChatMove == utils.Success {
		log.Debug().Msgf("all task for vod %s are done", v.ID)
		q.Update().SetVideoProcessing(false).SetChatProcessing(false).SetProcessing(false).SaveX(context.Background())
		v.Update().SetProcessing(false).SaveX(context.Background())
		// Send webhook
		go notification.SendVideoArchiveSuccessNotification(ch, v, q)
		// Start next queue item if there is one
		go s.CheckOnHold()

	}
}

func (s *Service) CheckIfLiveTasksAreDone(ch *ent.Channel, v *ent.Vod, qO *ent.Queue) {
	q, err := s.QueueService.ArchiveGetQueueItem(qO.ID)
	if err != nil {
		log.Error().Err(err).Msg("error getting queue item")
		return
	}
	if q.TaskVideoDownload == utils.Success && q.TaskVideoConvert == utils.Success && q.TaskVideoMove == utils.Success && q.TaskChatDownload == utils.Success && q.TaskChatConvert == utils.Success && q.TaskChatRender == utils.Success && q.TaskChatMove == utils.Success {
		log.Debug().Msgf("all task for live stream %s are done", v.ID)
		q.Update().SetVideoProcessing(false).SetChatProcessing(false).SetProcessing(false).SaveX(context.Background())

		v.Update().SetProcessing(false).SaveX(context.Background())
		// Send webhook
		go notification.SendLiveArchiveSuccessNotification(ch, v, q)
	}
}

func (s *Service) TaskError(ch *ent.Channel, v *ent.Vod, q *ent.Queue, task string) {
	go notification.SendErrorNotification(ch, v, q, task)
}
