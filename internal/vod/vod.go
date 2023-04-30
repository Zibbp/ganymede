package vod

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/cache"
	"github.com/zibbp/ganymede/internal/chat"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{Store: store}
}

type Vod struct {
	ID               uuid.UUID         `json:"id"`
	ExtID            string            `json:"ext_id"`
	Platform         utils.VodPlatform `json:"platform"`
	Type             utils.VodType     `json:"type"`
	Title            string            `json:"title"`
	Duration         int               `json:"duration"`
	Views            int               `json:"views"`
	Resolution       string            `json:"resolution"`
	Processing       bool              `json:"processing"`
	ThumbnailPath    string            `json:"thumbnail_path"`
	WebThumbnailPath string            `json:"web_thumbnail_path"`
	VideoPath        string            `json:"video_path"`
	ChatPath         string            `json:"chat_path"`
	ChatVideoPath    string            `json:"chat_video_path"`
	InfoPath         string            `json:"info_path"`
	CaptionPath      string            `json:"caption_path"`
	StreamedAt       time.Time         `json:"streamed_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CreatedAt        time.Time         `json:"created_at"`
	FolderName       string            `json:"folder_name"`
	FileName         string            `json:"file_name"`
	Locked           bool              `json:"locked"`
}

type Pagination struct {
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	TotalCount int        `json:"total_count"`
	Pages      int        `json:"pages"`
	Data       []*ent.Vod `json:"data"`
}

func (s *Service) CreateVod(vodDto Vod, cUUID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Create().SetID(vodDto.ID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetCaptionPath(vodDto.CaptionPath).SetStreamedAt(vodDto.StreamedAt).SetFolderName(vodDto.FolderName).SetFileName(vodDto.FileName).SetLocked(vodDto.Locked).Save(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error creating vod")
		if _, ok := err.(*ent.ConstraintError); ok {
			return nil, fmt.Errorf("channel does not exist")
		}
		return nil, fmt.Errorf("error creating vod: %v", err)
	}

	return v, nil
}

func (s *Service) GetVods(c echo.Context) ([]*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().WithChannel().Order(ent.Desc(vod.FieldStreamedAt)).All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vods")
		return nil, fmt.Errorf("error getting vods: %v", err)
	}

	return v, nil
}

func (s *Service) GetVodsByChannel(c echo.Context, cUUID uuid.UUID) ([]*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.HasChannelWith(channel.ID(cUUID))).Order(ent.Desc(vod.FieldStreamedAt)).All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vods by channel")
		return nil, fmt.Errorf("error getting vods by channel: %v", err)
	}

	return v, nil
}

func (s *Service) GetVod(vodID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod")
		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("vod not found")
		}
		return nil, fmt.Errorf("error getting vod: %v", err)
	}

	return v, nil
}

func (s *Service) GetVodWithChannel(vodID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).WithChannel().Only(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod")
		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("vod not found")
		}
		return nil, fmt.Errorf("error getting vod: %v", err)
	}

	return v, nil
}

func (s *Service) DeleteVod(c echo.Context, vodID uuid.UUID, deleteFiles bool) error {
	log.Debug().Msgf("deleting vod %s", vodID)
	// delete vod and queue item
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).WithQueue().WithChannel().Only(c.Request().Context())
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("vod not found")
		}
		return fmt.Errorf("error deleting vod: %v", err)
	}
	if v.Edges.Queue != nil {
		err = s.Store.Client.Queue.DeleteOneID(v.Edges.Queue.ID).Exec(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error deleting queue item: %v", err)
		}
	}

	// delete files
	if deleteFiles {
		log.Debug().Msgf("deleting files for vod %s", v.ID)
		path := fmt.Sprintf("/vods/%s/%s", v.Edges.Channel.Name, v.FolderName)
		err := utils.DeleteFolder(path)
		if err != nil {
			log.Debug().Err(err).Msg("error deleting files")
			return err
		}
	}

	err = s.Store.Client.Vod.DeleteOneID(vodID).Exec(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error deleting vod")
		return fmt.Errorf("error deleting vod: %v", err)
	}
	return nil
}

func (s *Service) UpdateVod(c echo.Context, vodID uuid.UUID, vodDto Vod, cUUID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.UpdateOneID(vodID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetCaptionPath(vodDto.CaptionPath).SetStreamedAt(vodDto.StreamedAt).SetLocked(vodDto.Locked).Save(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error updating vod")

		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil, fmt.Errorf("vod not found")
		}
		return nil, fmt.Errorf("error updating vod: %v", err)
	}

	return v, nil
}

func (s *Service) CheckVodExists(extID string) (bool, error) {
	_, err := s.Store.Client.Vod.Query().Where(vod.ExtID(extID)).Only(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("error checking vod exists")

		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("error checking vod exists: %v", err)
	}

	return true, nil
}

func (s *Service) SearchVods(c echo.Context, term string, limit int, offset int) (Pagination, error) {

	var pagination Pagination

	v, err := s.Store.Client.Vod.Query().Where(vod.TitleContainsFold(term)).Order(ent.Desc(vod.FieldStreamedAt)).Limit(limit).Offset(offset).All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error searching vods")
		return pagination, fmt.Errorf("error searching vods: %v", err)
	}

	totalCount, err := s.Store.Client.Vod.Query().Where(vod.TitleContainsFold(term)).Count(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting total vod count")
		return pagination, fmt.Errorf("error getting total vod count: %v", err)
	}

	pagination.TotalCount = totalCount
	pagination.Limit = limit
	pagination.Offset = offset
	pagination.Pages = int(math.Ceil(float64(totalCount) / float64(limit)))
	pagination.Data = v

	return pagination, nil
}

func (s *Service) GetVodPlaylists(c echo.Context, vodID uuid.UUID) ([]*ent.Playlist, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).WithPlaylists().Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod playlists")
		return nil, fmt.Errorf("error getting vod playlists: %v", err)
	}

	return v.Edges.Playlists, nil
}

func (s *Service) GetVodsPagination(c echo.Context, limit int, offset int, channelId uuid.UUID, types []utils.VodType) (Pagination, error) {
	var pagination Pagination

	// Query builder
	vodQuery := s.Store.Client.Vod.Query()

	// If channel id is not nil
	if channelId != uuid.Nil {
		vodQuery = vodQuery.Where(vod.HasChannelWith(channel.ID(channelId)))
	}

	// If types is not nil
	if len(types) > 0 {
		vodQuery = vodQuery.Where(vod.TypeIn(types...))
	}

	v, err := vodQuery.Order(ent.Desc(vod.FieldStreamedAt)).Limit(limit).Offset(offset).All(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vods")
		return pagination, fmt.Errorf("error getting vods: %v", err)
	}

	// Get total count
	// Amount will differ depending on types supplied
	totalCountQuery := s.Store.Client.Vod.Query()

	// If channel id is not nil
	if channelId != uuid.Nil {
		totalCountQuery = totalCountQuery.Where(vod.HasChannelWith(channel.ID(channelId)))
	}

	// If types is not nil
	if len(types) > 0 {
		totalCountQuery = totalCountQuery.Where(vod.TypeIn(types...))
	}

	totalCount, err := totalCountQuery.Count(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting total vod count")
		return pagination, fmt.Errorf("error getting total vod count: %v", err)
	}

	pagination.Limit = limit
	pagination.Offset = offset
	pagination.TotalCount = totalCount
	pagination.Pages = int(math.Ceil(float64(totalCount) / float64(limit)))
	pagination.Data = v

	return pagination, nil
}

func (s *Service) GetUserIdFromChat(c echo.Context, vodID uuid.UUID) (*int64, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod")
		return nil, fmt.Errorf("error getting vod: %v", err)
	}
	data, err := utils.ReadChatFile(v.ChatPath)
	if err != nil {
		log.Debug().Err(err).Msg("error reading chat file")
		return nil, fmt.Errorf("error reading chat file: %v", err)
	}
	var chatData *chat.ChatNoEmotes
	err = json.Unmarshal(data, &chatData)
	if err != nil {
		log.Debug().Err(err).Msg("error unmarshalling chat data")
		return nil, fmt.Errorf("error unmarshalling chat data: %v", err)
	}
	// Older chat files have the streamer ID stored as a string, need to convert to an int64
	var sID int64
	switch streamerChatId := chatData.Streamer.ID.(type) {
	case string:
		sID, err = strconv.ParseInt(streamerChatId, 10, 64)
		if err != nil {
			log.Debug().Err(err).Msg("error parsing streamer chat id")
			return nil, fmt.Errorf("error parsing streamer chat id: %v", err)
		}
	case float64:
		sID = int64(streamerChatId)
	}
	if sID == 0 {
		return nil, fmt.Errorf("error getting streamer id from chat")
	}

	return &sID, nil

}

func (s *Service) GetVodChatComments(c echo.Context, vodID uuid.UUID, start float64, end float64) (*[]chat.Comment, error) {
	envDeployment := os.Getenv("ENV")

	if envDeployment == "development" {
		utils.PrintMemUsage()
	}

	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return nil, fmt.Errorf("error getting vod chat: %v", err)
	}

	var chatData *chat.ChatNoEmotes
	var comments []chat.Comment
	cacheData, exists := cache.Cache().Get(v.ID.String())
	if exists {
		comments = cacheData.([]chat.Comment)
	} else {
		data, err := utils.ReadChatFile(v.ChatPath)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}
		err = json.Unmarshal(data, &chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}

		comments = chatData.Comments
		chatData = nil
		runtime.GC()

		// Sort the comments by their content offset seconds
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].ContentOffsetSeconds < comments[j].ContentOffsetSeconds
		})

		// Set cache
		err = cache.Cache().Set(v.ID.String(), comments, 10*time.Minute)
		if err != nil {
			log.Debug().Err(err).Msg("error setting cache")
			return nil, fmt.Errorf("error setting cache: %v", err)
		}

		runtime.GC()

	}

	// Reset the cache
	err = cache.Cache().Set(v.ID.String(), comments, 10*time.Minute)
	if err != nil {
		log.Debug().Err(err).Msg("error setting cache")
		return nil, fmt.Errorf("error setting cache: %v", err)
	}

	var filteredComments []chat.Comment

	// Use binary search to find the index of the first comment with an offset greater than the specified offset
	// This is much faster than iterating through the entire slice
	i := sort.Search(len(comments), func(i int) bool { return comments[i].ContentOffsetSeconds >= start })

	// Iterate through the comments starting at the index found above
	// Stop when we reach the end of the slice or the offset is greater than the specified end offset
	for i < len(comments) && comments[i].ContentOffsetSeconds <= end {
		filteredComments = append(filteredComments, comments[i])
		i++
	}

	// Cleanup
	chatData = nil
	comments = nil

	defer runtime.GC()

	if envDeployment == "development" {
		utils.PrintMemUsage()
	}

	return &filteredComments, nil
}

func (s *Service) GetNumberOfVodChatCommentsFromTime(c echo.Context, vodID uuid.UUID, start float64, commentCount int64) (*[]chat.Comment, error) {
	envDeployment := os.Getenv("ENV")

	if envDeployment == "development" {
		utils.PrintMemUsage()
	}

	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return nil, fmt.Errorf("error getting vod chat: %v", err)
	}

	var chatData *chat.ChatNoEmotes
	var comments []chat.Comment

	cacheData, exists := cache.Cache().Get(v.ID.String())

	if exists {
		comments = cacheData.([]chat.Comment)
	} else {
		data, err := utils.ReadChatFile(v.ChatPath)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}
		err = json.Unmarshal(data, &chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}

		comments = chatData.Comments
		chatData = nil
		runtime.GC()

		// Sort the comments by their content offset seconds
		sort.Slice(comments, func(i, j int) bool {
			return comments[i].ContentOffsetSeconds < comments[j].ContentOffsetSeconds
		})

		err = cache.Cache().Set(v.ID.String(), comments, 10*time.Minute)
		if err != nil {
			log.Debug().Err(err).Msg("error setting cache")
			return nil, fmt.Errorf("error setting cache: %v", err)
		}

		runtime.GC()

	}

	// Reset the cache
	err = cache.Cache().Set(v.ID.String(), comments, 10*time.Minute)
	if err != nil {
		log.Debug().Err(err).Msg("error setting cache")
		return nil, fmt.Errorf("error setting cache: %v", err)
	}

	var filteredComments []chat.Comment

	// Use binary search to find the index of the first comment with an offset greater than the specified offset
	// This is much faster than iterating through the entire slice
	i := sort.Search(len(comments), func(i int) bool { return comments[i].ContentOffsetSeconds >= start })

	// Iterate backwards from the index found above to get the last commentCount comments before the start time
	for j := i; len(filteredComments) < int(commentCount); j-- {
		if j < 0 || j >= len(comments) {
			break
		}
		comment := comments[j]
		filteredComments = append(filteredComments, comment)
	}

	// Check if the index is less than the number of comments we want to return
	if i-int(commentCount) >= 0 {
		filteredComments = comments[i-int(commentCount) : i]
	}

	// Cleanup
	chatData = nil
	comments = nil
	defer runtime.GC()

	if envDeployment == "development" {
		utils.PrintMemUsage()
	}

	return &filteredComments, nil

}

func (s *Service) GetVodChatEmotes(c echo.Context, vodID uuid.UUID) (*chat.GanymedeEmotes, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat emotes")
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}
	data, err := utils.ReadChatFile(v.ChatPath)
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat emotes")
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}
	var chatData *chat.ChatOnlyEmotes
	err = json.Unmarshal(data, &chatData)
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat emotes")
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}

	defer runtime.GC()

	var ganymedeEmotes chat.GanymedeEmotes

	switch {
	case len(chatData.Emotes.FirstParty) > 0 && len(chatData.Emotes.ThirdParty) > 0:
		log.Debug().Msgf("VOD %s chat playback embedded emotes found in 'emotes'", vodID)
		for _, emote := range chatData.Emotes.FirstParty {
			var ganymedeEmote chat.GanymedeEmote
			ganymedeEmote.Name = fmt.Sprint(emote.Name)
			ganymedeEmote.ID = emote.ID
			ganymedeEmote.URL = emote.Data
			ganymedeEmote.Type = "embed"
			ganymedeEmote.Width = emote.Width
			ganymedeEmote.Height = emote.Height
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, ganymedeEmote)
		}
		// Loop through third party emotes
		for _, emote := range chatData.Emotes.ThirdParty {
			var ganymedeEmote chat.GanymedeEmote
			ganymedeEmote.Name = fmt.Sprint(emote.Name)
			ganymedeEmote.ID = emote.ID
			ganymedeEmote.URL = emote.Data
			ganymedeEmote.Type = "embed"
			ganymedeEmote.Width = emote.Width
			ganymedeEmote.Height = emote.Height
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, ganymedeEmote)
		}
	case len(chatData.EmbeddedData.FirstParty) > 0 && len(chatData.EmbeddedData.ThirdParty) > 0:
		log.Debug().Msgf("VOD %s chat playback embedded emotes found in 'emebeddedData'", vodID)
		for _, emote := range chatData.EmbeddedData.FirstParty {
			var ganymedeEmote chat.GanymedeEmote
			ganymedeEmote.Name = fmt.Sprint(emote.Name)
			ganymedeEmote.ID = emote.ID
			ganymedeEmote.URL = emote.Data
			ganymedeEmote.Type = "embed"
			ganymedeEmote.Width = emote.Width
			ganymedeEmote.Height = emote.Height
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, ganymedeEmote)
		}
		// Loop through third party emotes
		for _, emote := range chatData.EmbeddedData.ThirdParty {
			var ganymedeEmote chat.GanymedeEmote
			ganymedeEmote.Name = fmt.Sprint(emote.Name)
			ganymedeEmote.ID = emote.ID
			ganymedeEmote.URL = emote.Data
			ganymedeEmote.Type = "embed"
			ganymedeEmote.Width = emote.Width
			ganymedeEmote.Height = emote.Height
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, ganymedeEmote)
		}
	default:
		log.Debug().Msgf("VOD %s chat playback embedded emotes not found, fetching emotes from providers", vodID)

		twitchGlobalEmotes, err := chat.GetTwitchGlobalEmotes()
		if err != nil {
			log.Debug().Err(err).Msg("error getting twitch global emotes")
			return nil, fmt.Errorf("error getting twitch global emotes: %v", err)
		}

		// Older chat files have the streamer ID stored as a string, need to convert to an int64
		var sID int64
		switch streamerChatId := chatData.Streamer.ID.(type) {
		case string:
			sID, err = strconv.ParseInt(streamerChatId, 10, 64)
			if err != nil {
				log.Debug().Err(err).Msg("error parsing streamer chat id")
				return nil, fmt.Errorf("error parsing streamer chat id: %v", err)
			}
		case float64:
			sID = int64(streamerChatId)
		}

		twitchChannelEmotes, err := chat.GetTwitchChannelEmotes(sID)
		if err != nil {
			log.Debug().Err(err).Msg("error getting twitch channel emotes")
			return nil, fmt.Errorf("error getting twitch channel emotes: %v", err)
		}
		sevenTVGlobalEmotes, err := chat.Get7TVGlobalEmotes()
		if err != nil {
			log.Debug().Err(err).Msg("error getting 7tv global emotes")
			return nil, fmt.Errorf("error getting 7tv global emotes: %v", err)
		}
		sevenTVChannelEmotes, err := chat.Get7TVChannelEmotes(sID)
		if err != nil {
			log.Debug().Err(err).Msg("error getting 7tv channel emotes")
			return nil, fmt.Errorf("error getting 7tv channel emotes: %v", err)
		}
		bttvGlobalEmotes, err := chat.GetBTTVGlobalEmotes()
		if err != nil {
			log.Debug().Err(err).Msg("error getting bttv global emotes")
			return nil, fmt.Errorf("error getting bttv global emotes: %v", err)
		}
		bttvChannelEmotes, err := chat.GetBTTVChannelEmotes(sID)
		if err != nil {
			log.Debug().Err(err).Msg("error getting bttv channel emotes")
			return nil, fmt.Errorf("error getting bttv channel emotes: %v", err)
		}
		ffzGlobalEmotes, err := chat.GetFFZGlobalEmotes()
		if err != nil {
			log.Debug().Err(err).Msg("error getting ffz global emotes")
			return nil, fmt.Errorf("error getting ffz global emotes: %v", err)
		}
		ffzChannelEmotes, err := chat.GetFFZChannelEmotes(sID)
		if err != nil {
			log.Debug().Err(err).Msg("error getting ffz channel emotes")
			return nil, fmt.Errorf("error getting ffz channel emotes: %v", err)
		}

		// Loop through twitch global emotes
		for _, emote := range twitchGlobalEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}
		// Loop through twitch channel emotes
		for _, emote := range twitchChannelEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}
		// Loop through 7tv global emotes
		for _, emote := range sevenTVGlobalEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}
		// Loop through 7tv channel emotes
		for _, emote := range sevenTVChannelEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}
		// Loop through bttv global emotes
		for _, emote := range bttvGlobalEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}
		// Loop through bttv channel emotes
		for _, emote := range bttvChannelEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}
		// Loop through ffz global emotes
		for _, emote := range ffzGlobalEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}
		// Loop through ffz channel emotes
		for _, emote := range ffzChannelEmotes {
			ganymedeEmotes.Emotes = append(ganymedeEmotes.Emotes, *emote)
		}

	}

	chatData = nil

	defer runtime.GC()
	return &ganymedeEmotes, nil

}

func (s *Service) GetVodChatBadges(c echo.Context, vodID uuid.UUID) (*chat.GanymedeBadges, error) {
	envDeployment := os.Getenv("ENV")

	if envDeployment == "development" {
		utils.PrintMemUsage()
	}

	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat emotes")
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}
	data, err := utils.ReadChatFile(v.ChatPath)
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat emotes")
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}

	var chatData *chat.ChatOnlyBadges
	var chatDataOld *chat.ChatOnlyBadgesOld
	err = json.Unmarshal(data, &chatData)
	if err != nil {
		// attempt to unmarshal old format
		err = json.Unmarshal(data, &chatDataOld)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat emotes")
			return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
		}
	}

	// convert old format to new format
	if chatDataOld != nil {
		for _, badge := range chatDataOld.EmbeddedData.TwitchBadges {
			var tmpBadges = make(map[string]chat.ChatTwitchBadgeVersion)
			for v, imgData := range badge.Versions {
				chatTwitchBadgeVersion := chat.ChatTwitchBadgeVersion{
					Title:       badge.Name,
					Description: fmt.Sprintf("%s-%s", badge.Name, v),
					Bytes:       imgData,
				}
				tmpBadges[v] = chatTwitchBadgeVersion
			}
			chatData.EmbeddedData.TwitchBadges = append(chatData.EmbeddedData.TwitchBadges, chat.ChatTwitchBadge{
				Name:     badge.Name,
				Versions: tmpBadges,
			})
		}
	}

	var badgeResp chat.GanymedeBadges

	// If emebedded badges
	if len(chatData.EmbeddedData.TwitchBadges) != 0 {
		log.Debug().Msgf("VOD %s chat playback embedded badges found", vodID)
		// Emebedded badges have duplicate arrays for each of the below
		// So we need to check if we have already added the badge to the response
		// To ensure we use the channel's badge and not the global one
		subscriberBadgesSet := false
		bitsBadgesSet := false
		subGiftBadgesSet := false

		for _, badge := range chatData.EmbeddedData.TwitchBadges {

			if badge.Name == "subscriber" && !subscriberBadgesSet {
				empty := false
				for v, imgData := range badge.Versions {
					// check if empty
					if imgData.Title == "" {
						empty = true
					} else {
						badgeResp.Badges = append(badgeResp.Badges, chat.GanymedeBadge{
							Name:       badge.Name,
							Version:    v,
							Title:      fmt.Sprintf("%s %s", badge.Name, v),
							ImageUrl1X: fmt.Sprintf("data:image/png;base64,%s", imgData.Bytes),
						})
					}
				}
				if empty {
					continue
				} else {
					subscriberBadgesSet = true
					continue
				}
			}

			if badge.Name == "bits" && !bitsBadgesSet {
				empty := false
				for v, imgData := range badge.Versions {
					if imgData.Title == "" {
						empty = true
					} else {
						badgeResp.Badges = append(badgeResp.Badges, chat.GanymedeBadge{
							Name:       badge.Name,
							Version:    v,
							Title:      fmt.Sprintf("%s %s", badge.Name, v),
							ImageUrl1X: fmt.Sprintf("data:image/png;base64,%s", imgData.Bytes),
						})
					}
				}
				if empty {
					continue
				} else {
					bitsBadgesSet = true
					continue
				}
			}
			if badge.Name == "sub-gifter" && !subGiftBadgesSet {
				empty := false
				for v, imgData := range badge.Versions {
					if imgData.Title == "" {
						empty = true
					} else {
						badgeResp.Badges = append(badgeResp.Badges, chat.GanymedeBadge{
							Name:       badge.Name,
							Version:    v,
							Title:      fmt.Sprintf("%s %s", badge.Name, v),
							ImageUrl1X: fmt.Sprintf("data:image/png;base64,%s", imgData.Bytes),
						})
					}
				}
				if empty {
					continue
				} else {
					subGiftBadgesSet = true
					continue
				}
			}

			if badge.Name != "subscriber" && badge.Name != "bits" && badge.Name != "sub-gifter" {
				for v, imgData := range badge.Versions {
					if imgData.Title == "" {
					} else {
						badgeResp.Badges = append(badgeResp.Badges, chat.GanymedeBadge{
							Name:       badge.Name,
							Version:    v,
							Title:      fmt.Sprintf("%s %s", badge.Name, v),
							ImageUrl1X: fmt.Sprintf("data:image/png;base64,%s", imgData.Bytes),
						})
					}
					break
				}
			}

		}

	} else {
		log.Debug().Msgf("VOD %s chat playback embedded badges not found, fetching badges from providers", vodID)
		// Older chat files have the streamer ID stored as a string, need to convert to an int64
		var sID int64
		switch streamerChatId := chatData.Streamer.ID.(type) {
		case string:
			sID, err = strconv.ParseInt(streamerChatId, 10, 64)
			if err != nil {
				log.Debug().Err(err).Msg("error parsing streamer chat id")
				return nil, fmt.Errorf("error parsing streamer chat id: %v", err)
			}
		case float64:
			sID = int64(streamerChatId)
		}

		twitchBadges, err := chat.GetTwitchGlobalBadges()
		if err != nil {
			log.Error().Err(err).Msg("error getting twitch global badges")
			return nil, fmt.Errorf("error getting twitch global badges: %v", err)
		}
		channelBadges, err := chat.GetTwitchChannelBadges(sID)
		if err != nil {
			log.Error().Err(err).Msg("error getting twitch channel badges")
			return nil, fmt.Errorf("error getting twitch channel badges: %v", err)
		}

		// Loop through twitch global badges
		badgeResp.Badges = append(badgeResp.Badges, twitchBadges.Badges...)

		// Loop through twitch channel badges

		badgeResp.Badges = append(badgeResp.Badges, channelBadges.Badges...)

		twitchBadges = nil
		channelBadges = nil

	}

	chatData = nil
	defer runtime.GC()

	if envDeployment == "development" {
		utils.PrintMemUsage()
	}

	return &badgeResp, nil

}

func (s *Service) LockVod(c echo.Context, vID uuid.UUID, status bool) error {
	_, err := s.Store.Client.Vod.UpdateOneID(vID).SetLocked(status).Save(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error updating vod")

		// if vod not found
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("vod not found")
		}
		return fmt.Errorf("error updating vod: %v", err)
	}

	return nil
}
