package archive

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/channel"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
	"strings"
	"time"
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
func (s *Service) ArchiveTwitchChannel(c echo.Context, cName string) (*ent.Channel, error) {
	// Fetch channel from Twitch API
	tChannel, err := s.TwitchService.GetUserByLogin(c, cName)
	if err != nil {
		return nil, fmt.Errorf("error fetching twitch channel: %v", err)
	}

	// Check if channel exists in DB
	cCheck := s.ChannelService.CheckChannelExists(c, tChannel.Login)
	if cCheck == true {
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
		Name:        tChannel.Login,
		DisplayName: tChannel.DisplayName,
		ImagePath:   fmt.Sprintf("/vods/%s/profile.png", tChannel.Login),
	}

	dbC, err := s.ChannelService.CreateChannel(c, channelDTO)
	if err != nil {
		return nil, fmt.Errorf("error creating channel: %v", err)
	}

	return dbC, nil

}

func (s *Service) ArchiveTwitchVod(c echo.Context, vID string, quality string, chat bool) (*TwitchVodResponse, error) {
	// Fetch VOD from Twitch API
	tVod, err := s.TwitchService.GetVodByID(vID)
	if err != nil {
		return nil, fmt.Errorf("error fetching twitch vod: %v", err)
	}
	// Check if vod is already archived
	vCheck, err := s.VodService.CheckVodExists(c, tVod.ID)
	if err != nil {
		return nil, fmt.Errorf("error checking if vod exists: %v", err)
	}
	if vCheck == true {
		return nil, fmt.Errorf("vod already exists")
	}
	// Check if channel exists
	cCheck := s.ChannelService.CheckChannelExists(c, tVod.UserLogin)
	if cCheck == false {
		log.Debug().Msgf("channel does not exist: %s while archiving vod. creating now.", tVod.UserLogin)
		_, err := s.ArchiveTwitchChannel(c, tVod.UserLogin)
		if err != nil {
			return nil, fmt.Errorf("error creating channel: %v", err)
		}
	}
	// Fetch channel
	dbC, err := s.ChannelService.GetChannelByName(c, tVod.UserLogin)
	if err != nil {
		return nil, fmt.Errorf("error fetching channel: %v", err)
	}

	// Generate VOD ID for folder name
	vUUID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("error creating vod uuid: %v", err)
	}

	// Sets
	rootVodPath := fmt.Sprintf("/vods/%s/%s_%s", tVod.UserLogin, tVod.ID, vUUID.String())
	var chatPath string
	var chatVideoPath string
	if chat == true {
		chatPath = fmt.Sprintf("%s/%s-chat.json", rootVodPath, tVod.ID)
		chatVideoPath = fmt.Sprintf("%s/%s-chat.mp4", rootVodPath, tVod.ID)
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
		ThumbnailPath:    fmt.Sprintf("%s/%s-thumbnail.jpg", rootVodPath, tVod.ID),
		WebThumbnailPath: fmt.Sprintf("%s/%s-web_thumbnail.jpg", rootVodPath, tVod.ID),
		VideoPath:        fmt.Sprintf("%s/%s-video.mp4", rootVodPath, tVod.ID),
		ChatPath:         chatPath,
		ChatVideoPath:    chatVideoPath,
		InfoPath:         fmt.Sprintf("%s/%s-info.json", rootVodPath, tVod.ID),
		StreamedAt:       parsedDate,
	}
	v, err := s.VodService.CreateVod(c, vodDTO, dbC.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	// Create queue item
	q, err := s.QueueService.CreateQueueItem(c, queue.Queue{}, v.ID)
	if err != nil {
		return nil, fmt.Errorf("error creating queue item: %v", err)
	}

	// If chat is disabled update queue
	if chat == false {
		q.Update().SetChatProcessing(false).SetTaskChatDownload(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove(utils.Success).SaveX(c.Request().Context())
	}

	go s.TaskVodCreateFolder(dbC, v, q, true)

	return &TwitchVodResponse{
		VOD:   v,
		Queue: q,
	}, nil
}

func (s *Service) TaskVodCreateFolder(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task vod create folder for vod %s", v.ID)
	q.Update().SetTaskVodCreateFolder(utils.Running).SaveX(context.Background())
	// Create folder
	err := utils.CreateFolder(fmt.Sprintf("%s/%s_%s", ch.Name, v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating vod folder")
		q.Update().SetTaskVodCreateFolder(utils.Failed).SaveX(context.Background())
		return
	}
	q.Update().SetTaskVodCreateFolder(utils.Success).SaveX(context.Background())

	if cont == true {
		go s.TaskVodDownloadThumbnail(ch, v, q, true)
	}
}

func (s *Service) TaskVodDownloadThumbnail(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task vod download thumbnail for vod %s", v.ID)
	q.Update().SetTaskVodDownloadThumbnail(utils.Running).SaveX(context.Background())

	// Fetch VOD from Twitch for thumbnails
	tVod, err := s.TwitchService.GetVodByID(v.ExtID)
	if err != nil {
		log.Error().Err(err).Msg("error fetching twitch vod")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		return
	}

	fullResThumbnailUrl := strings.ReplaceAll(tVod.ThumbnailURL, "%{width}", "1920")
	fullResThumbnailUrl = strings.ReplaceAll(fullResThumbnailUrl, "%{height}", "1080")

	webResThumbnailUrl := strings.ReplaceAll(tVod.ThumbnailURL, "%{width}", "640")
	webResThumbnailUrl = strings.ReplaceAll(webResThumbnailUrl, "%{height}", "360")

	// Download full resolution thumbnail
	err = utils.DownloadFile(fullResThumbnailUrl, fmt.Sprintf("%s/%s_%s", ch.Name, v.ExtID, v.ID), fmt.Sprintf("%s-thumbnail.jpg", v.ExtID))
	if err != nil {
		log.Error().Err(err).Msg("error downloading thumbnail")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		return
	}
	// Download web resolution thumbnail
	err = utils.DownloadFile(webResThumbnailUrl, fmt.Sprintf("%s/%s_%s", ch.Name, v.ExtID, v.ID), fmt.Sprintf("%s-web_thumbnail.jpg", v.ExtID))
	if err != nil {
		log.Error().Err(err).Msg("error downloading thumbnail")
		q.Update().SetTaskVodDownloadThumbnail(utils.Failed).SaveX(context.Background())
		return
	}

	q.Update().SetTaskVodDownloadThumbnail(utils.Success).SaveX(context.Background())

	if cont == true {
		go s.TaskVodSaveInfo(ch, v, q, true)
	}
}

func (s *Service) TaskVodSaveInfo(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task vod save info for vod %s", v.ID)
	q.Update().SetTaskVodSaveInfo(utils.Running).SaveX(context.Background())

	// Fetch VOD from Twitch
	tVod, err := s.TwitchService.GetVodByID(v.ExtID)
	if err != nil {
		log.Error().Err(err).Msg("error fetching twitch vod")
		q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
		return
	}

	err = utils.WriteJson(tVod, fmt.Sprintf("%s/%s_%s", ch.Name, v.ExtID, v.ID), fmt.Sprintf("%s-info.json", v.ExtID))
	if err != nil {
		log.Error().Err(err).Msg("error saving info")
		q.Update().SetTaskVodSaveInfo(utils.Failed).SaveX(context.Background())
		return
	}
	q.Update().SetTaskVodSaveInfo(utils.Success).SaveX(context.Background())
	if cont == true {
		go s.TaskVideoDownload(ch, v, q, true)
		//	Check if chat download task is set to success
		if q.TaskChatDownload == utils.Pending {
			go s.TaskChatDownload(ch, v, q, true)
		}
	}
}

func (s *Service) TaskVideoDownload(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task video download for vod %s", v.ID)
	q.Update().SetTaskVideoDownload(utils.Running).SaveX(context.Background())

	err := exec.DownloadTwitchVodVideo(v)
	if err != nil {
		log.Error().Err(err).Msg("error downloading video")
		q.Update().SetTaskVideoDownload(utils.Failed).SaveX(context.Background())
		return
	}

	q.Update().SetTaskVideoDownload(utils.Success).SaveX(context.Background())

	// Always invoke task video move if video download was successful
	go s.TaskVideoMove(ch, v, q, true)

}

func (s *Service) TaskVideoMove(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task video move for vod %s", v.ID)
	q.Update().SetTaskVideoMove(utils.Running).SaveX(context.Background())

	sourcePath := fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID)
	destPath := fmt.Sprintf("/vods/%s/%s_%s/%s-video.mp4", ch.Name, v.ExtID, v.ID, v.ExtID)

	err := utils.MoveFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving video")
		q.Update().SetTaskVideoMove(utils.Failed).SaveX(context.Background())
		return
	}

	q.Update().SetTaskVideoMove(utils.Success).SaveX(context.Background())

	// Set video as complete
	q.Update().SetVideoProcessing(false).SaveX(context.Background())
}

func (s *Service) TaskChatDownload(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task chat download for vod %s", v.ID)
	q.Update().SetTaskChatDownload(utils.Running).SaveX(context.Background())

	err := exec.DownloadTwitchVodChat(v)
	if err != nil {
		log.Error().Err(err).Msg("error downloading chat")
		q.Update().SetTaskChatDownload(utils.Failed).SaveX(context.Background())
		return
	}

	q.Update().SetTaskChatDownload(utils.Success).SaveX(context.Background())

	if cont == true {
		go s.TaskChatRender(ch, v, q, true)
	}
}

func (s *Service) TaskChatRender(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task chat render for vod %s", v.ID)
	q.Update().SetTaskChatRender(utils.Running).SaveX(context.Background())

	err := exec.RenderTwitchVodChat(v)
	if err != nil {
		log.Error().Err(err).Msg("error rendering chat")
		q.Update().SetTaskChatRender(utils.Failed).SaveX(context.Background())
		return
	}

	q.Update().SetTaskChatRender(utils.Success).SaveX(context.Background())

	// Always move chat if render was successful
	go s.TaskChatMove(ch, v, q, true)

}

func (s *Service) TaskChatMove(ch *ent.Channel, v *ent.Vod, q *ent.Queue, cont bool) {
	log.Debug().Msgf("starting task chat move for vod %s", v.ID)
	q.Update().SetTaskChatMove(utils.Running).SaveX(context.Background())

	// Chat JSON
	sourcePath := fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID)
	destPath := fmt.Sprintf("/vods/%s/%s_%s/%s-chat.json", ch.Name, v.ExtID, v.ID, v.ExtID)

	err := utils.MoveFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving chat")
		q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
		return
	}
	// Chat Video
	sourcePath = fmt.Sprintf("/tmp/%s_%s-chat.mp4", v.ExtID, v.ID)
	destPath = fmt.Sprintf("/vods/%s/%s_%s/%s-chat.mp4", ch.Name, v.ExtID, v.ID, v.ExtID)

	err = utils.MoveFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error moving chat")
		q.Update().SetTaskChatMove(utils.Failed).SaveX(context.Background())
		return
	}

	q.Update().SetTaskChatMove(utils.Success).SaveX(context.Background())

	// Set chat as complete
	q.Update().SetChatProcessing(false).SaveX(context.Background())
}
