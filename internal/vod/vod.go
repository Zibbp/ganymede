package vod

import (
	"context"
	"fmt"
	gojson "github.com/goccy/go-json"
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
	"math"
	"strconv"
	"time"
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
	StreamedAt       time.Time         `json:"streamed_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	CreatedAt        time.Time         `json:"created_at"`
}

type Pagination struct {
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	TotalCount int        `json:"total_count"`
	Pages      int        `json:"pages"`
	Data       []*ent.Vod `json:"data"`
}

func (s *Service) CreateVod(vodDto Vod, cUUID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Create().SetID(vodDto.ID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetStreamedAt(vodDto.StreamedAt).Save(context.Background())
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

func (s *Service) DeleteVod(c echo.Context, vodID uuid.UUID) error {
	// delete vod and queue item
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).WithQueue().Only(c.Request().Context())
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

	err = s.Store.Client.Vod.DeleteOneID(vodID).Exec(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error deleting vod")
		return fmt.Errorf("error deleting vod: %v", err)
	}
	return nil
}

func (s *Service) UpdateVod(c echo.Context, vodID uuid.UUID, vodDto Vod, cUUID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.UpdateOneID(vodID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetStreamedAt(vodDto.StreamedAt).Save(c.Request().Context())
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

func (s *Service) GetVodsPagination(c echo.Context, limit int, offset int, channelId uuid.UUID) (Pagination, error) {

	if channelId == uuid.Nil {
		var pagination Pagination

		v, err := s.Store.Client.Vod.Query().Order(ent.Desc(vod.FieldStreamedAt)).Limit(limit).Offset(offset).All(c.Request().Context())
		if err != nil {
			log.Debug().Err(err).Msg("error getting vods")
			return pagination, fmt.Errorf("error getting vods: %v", err)
		}

		totalCount, err := s.Store.Client.Vod.Query().Count(c.Request().Context())
		if err != nil {
			log.Debug().Err(err).Msg("error getting vods count")
			return pagination, fmt.Errorf("error getting vods count: %v", err)
		}

		pagination.Limit = limit
		pagination.Offset = offset
		pagination.TotalCount = totalCount
		pagination.Pages = int(math.Ceil(float64(totalCount) / float64(limit)))
		pagination.Data = v

		return pagination, nil

	} else {
		var pagination Pagination

		v, err := s.Store.Client.Vod.Query().Where(vod.HasChannelWith(channel.ID(channelId))).Order(ent.Desc(vod.FieldStreamedAt)).Limit(limit).Offset(offset).All(c.Request().Context())
		if err != nil {
			log.Debug().Err(err).Msg("error getting vods")
			return pagination, fmt.Errorf("error getting vods: %v", err)
		}

		totalCount, err := s.Store.Client.Vod.Query().Where(vod.HasChannelWith(channel.ID(channelId))).Count(c.Request().Context())
		if err != nil {
			log.Debug().Err(err).Msg("error getting vods count")
			return pagination, fmt.Errorf("error getting vods count: %v", err)
		}

		pagination.Limit = limit
		pagination.Offset = offset
		pagination.TotalCount = totalCount
		pagination.Pages = int(math.Ceil(float64(totalCount) / float64(limit)))
		pagination.Data = v

		return pagination, nil
	}

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
	err = gojson.Unmarshal(data, &chatData)
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
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return nil, fmt.Errorf("error getting vod chat: %v", err)
	}

	var chatData *chat.ChatNoEmotes
	cacheData, exists := cache.Cache().Get(fmt.Sprintf("%s", v.ID))
	if exists {
		err := gojson.Unmarshal(cacheData.([]byte), &chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error unmarshalling chat data")
			return nil, fmt.Errorf("error unmarshalling chat data: %v", err)
		}
		log.Debug().Msgf("Using chat cache for VOD %s", v.ID)
	} else {
		data, err := utils.ReadChatFile(v.ChatPath)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}
		err = gojson.Unmarshal(data, &chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}

		data, err = gojson.Marshal(chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error marshalling chat data")
			return nil, fmt.Errorf("error marshalling chat data: %v", err)
		}
		err = cache.Cache().Set(fmt.Sprintf("%s", v.ID), data, 30*time.Minute)
		if err != nil {
			log.Debug().Err(err).Msg("error setting cache")
			return nil, fmt.Errorf("error setting cache: %v", err)
		}
		log.Debug().Msgf("Set chat cache for VOD %s", v.ID)
		data = nil
	}

	var filteredComments []chat.Comment
	for _, message := range chatData.Comments {
		if message.ContentOffsetSeconds >= start && message.ContentOffsetSeconds <= end {
			filteredComments = append(filteredComments, message)
		}
	}

	chatData = nil
	cacheData = nil

	return &filteredComments, nil
}

func (s *Service) GetNumberOfVodChatCommentsFromTime(c echo.Context, vodID uuid.UUID, start float64, commentCount int64) (*[]chat.Comment, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return nil, fmt.Errorf("error getting vod chat: %v", err)
	}

	var chatData *chat.ChatNoEmotes
	cacheData, exists := cache.Cache().Get(fmt.Sprintf("%s", v.ID))
	if exists {
		err := gojson.Unmarshal(cacheData.([]byte), &chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error unmarshalling chat data")
			return nil, fmt.Errorf("error unmarshalling chat data: %v", err)
		}
		log.Debug().Msgf("Using chat cache for VOD %s", v.ID)
	} else {
		data, err := utils.ReadChatFile(v.ChatPath)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}
		err = gojson.Unmarshal(data, &chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error getting vod chat")
			return nil, fmt.Errorf("error getting vod chat: %v", err)
		}

		data, err = gojson.Marshal(chatData)
		if err != nil {
			log.Debug().Err(err).Msg("error marshalling chat data")
			return nil, fmt.Errorf("error marshalling chat data: %v", err)
		}
		err = cache.Cache().Set(fmt.Sprintf("%s", v.ID), data, 30*time.Minute)
		if err != nil {
			log.Debug().Err(err).Msg("error setting cache")
			return nil, fmt.Errorf("error setting cache: %v", err)
		}
		log.Debug().Msgf("Set chat cache for VOD %s", v.ID)
		data = nil
	}

	var filteredComments []chat.Comment
	for _, message := range chatData.Comments {
		if message.ContentOffsetSeconds <= start {
			filteredComments = append(filteredComments, message)
		}
	}

	count := len(filteredComments)
	// Count to int64
	var i int64
	i = int64(count)
	if i < commentCount {
		return nil, nil
	}

	chatData = nil
	cacheData = nil

	filteredComments = filteredComments[i-commentCount : i]

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
	err = gojson.Unmarshal(data, &chatData)
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat emotes")
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}

	if len(chatData.Emotes.FirstParty) > 0 && len(chatData.Emotes.ThirdParty) > 0 {
		log.Debug().Msgf("detected embedded emotes for vod %s", vodID)
		var ganymedeEmotes chat.GanymedeEmotes
		// Loop through first party emotes
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
		chatData = nil
		data = nil
		return &ganymedeEmotes, nil
	} else {
		// Embedded emotes not found, fetch emotes from the providers
		var ganymedeEmotes chat.GanymedeEmotes
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
		chatData = nil
		return &ganymedeEmotes, nil
	}
}

func (s *Service) GetVodChatBadges(c echo.Context, vodID uuid.UUID) (*chat.BadgeResp, error) {
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
	var chatData *chat.ChatNoEmotes
	err = gojson.Unmarshal(data, &chatData)
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat emotes")
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
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

	var badgeResp chat.BadgeResp
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
	for _, badge := range twitchBadges.Badges {
		badgeResp.Badges = append(badgeResp.Badges, badge)
	}
	// Loop through twitch channel badges
	for _, badge := range channelBadges.Badges {
		badgeResp.Badges = append(badgeResp.Badges, badge)
	}

	chatData = nil
	data = nil

	return &badgeResp, nil
}
