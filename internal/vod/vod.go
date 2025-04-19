package vod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/riverqueue/river/rivertype"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	entChapter "github.com/zibbp/ganymede/ent/chapter"
	entMutedSegment "github.com/zibbp/ganymede/ent/mutedsegment"
	"github.com/zibbp/ganymede/ent/playlist"
	"github.com/zibbp/ganymede/ent/predicate"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/cache"
	"github.com/zibbp/ganymede/internal/chat"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	"github.com/zibbp/ganymede/internal/utils"
)

type Service struct {
	Store       *database.Database
	RiverClient *tasks_client.RiverClient
	Platform    platform.Platform
}

func NewService(store *database.Database, riverClient *tasks_client.RiverClient, platform platform.Platform) *Service {
	return &Service{Store: store, RiverClient: riverClient, Platform: platform}
}

type Vod struct {
	ID                      uuid.UUID           `json:"id"`
	ExtID                   string              `json:"ext_id"`
	ExtStreamID             string              `json:"ext_stream_id"`
	ClipExtVodID            string              `json:"clip_ext_vod_id"`
	Platform                utils.VideoPlatform `json:"platform"`
	Type                    utils.VodType       `json:"type"`
	Title                   string              `json:"title"`
	Duration                int                 `json:"duration"`
	ClipVodOffset           int                 `json:"clip_vod_offset"`
	Views                   int                 `json:"views"`
	Resolution              string              `json:"resolution"`
	Processing              bool                `json:"processing"`
	ThumbnailPath           string              `json:"thumbnail_path"`
	WebThumbnailPath        string              `json:"web_thumbnail_path"`
	VideoPath               string              `json:"video_path"`
	VideoHLSPath            string              `json:"video_hls_path"`
	ChatPath                string              `json:"chat_path"`
	LiveChatPath            string              `json:"live_chat_path"`
	LiveChatConvertPath     string              `json:"live_chat_convert_path"`
	ChatVideoPath           string              `json:"chat_video_path"`
	InfoPath                string              `json:"info_path"`
	CaptionPath             string              `json:"caption_path"`
	StreamedAt              time.Time           `json:"streamed_at"`
	UpdatedAt               time.Time           `json:"updated_at"`
	CreatedAt               time.Time           `json:"created_at"`
	FolderName              string              `json:"folder_name"`
	FileName                string              `json:"file_name"`
	Locked                  bool                `json:"locked"`
	TmpVideoDownloadPath    string              `json:"tmp_video_download_path"`
	TmpVideoConvertPath     string              `json:"tmp_video_convert_path"`
	TmpChatDownloadPath     string              `json:"tmp_chat_download_path"`
	TmpLiveChatDownloadPath string              `json:"tmp_live_chat_download_path"`
	TmpLiveChatConvertPath  string              `json:"tmp_live_chat_convert_path"`
	TmpChatRenderPath       string              `json:"tmp_chat_render_path"`
	TmpVideoHLSPath         string              `json:"tmp_video_hls_path"`
}

type Pagination struct {
	Offset     int        `json:"offset"`
	Limit      int        `json:"limit"`
	TotalCount int        `json:"total_count"`
	Pages      int        `json:"pages"`
	Data       []*ent.Vod `json:"data"`
}

type MutedSegment struct {
	ID    string `json:"id"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

func (s *Service) CreateVod(vodDto Vod, cUUID uuid.UUID) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Create().SetID(vodDto.ID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetExtStreamID(vodDto.ExtStreamID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetCaptionPath(vodDto.CaptionPath).SetStreamedAt(vodDto.StreamedAt).SetFolderName(vodDto.FolderName).SetFileName(vodDto.FileName).SetLocked(vodDto.Locked).SetTmpVideoDownloadPath(vodDto.TmpVideoDownloadPath).SetTmpVideoConvertPath(vodDto.TmpVideoConvertPath).SetTmpChatDownloadPath(vodDto.TmpChatDownloadPath).SetTmpLiveChatDownloadPath(vodDto.TmpLiveChatDownloadPath).SetTmpLiveChatConvertPath(vodDto.TmpLiveChatConvertPath).SetTmpChatRenderPath(vodDto.TmpChatRenderPath).SetLiveChatPath(vodDto.LiveChatPath).SetLiveChatConvertPath(vodDto.LiveChatConvertPath).SetVideoHlsPath(vodDto.VideoHLSPath).SetTmpVideoHlsPath(vodDto.TmpVideoHLSPath).SetClipVodOffset(vodDto.ClipVodOffset).SetClipExtVodID(vodDto.ClipExtVodID).Save(context.Background())
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

// GetVodByExternalId gets a VOD by it's external (platform) ID. For more advanced usage use GetVod().
func (s *Service) GetVodByExternalId(ctx context.Context, externalId string) (*ent.Vod, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ExtID(externalId)).Only(ctx)
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

func (s *Service) GetVod(ctx context.Context, vodID uuid.UUID, withChannel bool, withChapters bool, withMutedSegments bool, withQueue bool) (*ent.Vod, error) {
	q := s.Store.Client.Vod.Query()
	q.Where(vod.ID(vodID))

	if withChannel {
		q.WithChannel()
	}
	if withChapters {
		q.WithChapters()
	}
	if withMutedSegments {
		q.WithMutedSegments()
	}
	if withQueue {
		q.WithQueue()
	}

	v, err := q.Only(ctx)
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
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).WithQueue().WithChannel().WithChapters().WithMutedSegments().Only(c.Request().Context())
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
	if v.Edges.Chapters != nil {
		_, err = s.Store.Client.Chapter.Delete().Where(entChapter.HasVodWith(vod.ID(vodID))).Exec(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error deleting chapters: %v", err)
		}
	}
	if v.Edges.MutedSegments != nil {
		_, err = s.Store.Client.MutedSegment.Delete().Where(entMutedSegment.HasVodWith(vod.ID(vodID))).Exec(c.Request().Context())
		if err != nil {
			return fmt.Errorf("error deleting muted segments: %v", err)
		}
	}

	// delete files
	if deleteFiles {
		log.Debug().Msgf("deleting files for vod %s", v.ID)

		path := filepath.Dir(filepath.Clean(v.VideoPath))

		if err := utils.DeleteDirectory(path); err != nil {
			log.Error().Err(err).Msg("error deleting directory")
			return fmt.Errorf("error deleting directory: %v", err)
		}

		// attempt to delete temp files
		if err := utils.DeleteFile(v.TmpVideoDownloadPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug().Msgf("temp file %s does not exist", v.TmpVideoDownloadPath)
			} else {
				return err
			}
		}
		if err := utils.DeleteFile(v.TmpVideoConvertPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug().Msgf("temp file %s does not exist", v.TmpVideoConvertPath)
			} else {
				return err
			}
		}
		if err := utils.DeleteDirectory(v.TmpVideoHlsPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug().Msgf("temp file %s does not exist", v.TmpVideoHlsPath)
			} else {
				return err
			}
		}
		if err := utils.DeleteFile(v.TmpChatDownloadPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug().Msgf("temp file %s does not exist", v.TmpChatDownloadPath)
			} else {
				return err
			}
		}
		if err := utils.DeleteFile(v.TmpChatRenderPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug().Msgf("temp file %s does not exist", v.TmpChatRenderPath)
			} else {
				return err
			}
		}
		if err := utils.DeleteFile(v.TmpLiveChatConvertPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug().Msgf("temp file %s does not exist", v.TmpLiveChatConvertPath)
			} else {
				return err
			}
		}
		if err := utils.DeleteFile(v.TmpLiveChatDownloadPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				log.Debug().Msgf("temp file %s does not exist", v.TmpLiveChatDownloadPath)
			} else {
				return err
			}
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
	v, err := s.Store.Client.Vod.UpdateOneID(vodID).SetChannelID(cUUID).SetExtID(vodDto.ExtID).SetExtID(vodDto.ExtID).SetPlatform(vodDto.Platform).SetType(vodDto.Type).SetTitle(vodDto.Title).SetDuration(vodDto.Duration).SetViews(vodDto.Views).SetResolution(vodDto.Resolution).SetProcessing(vodDto.Processing).SetThumbnailPath(vodDto.ThumbnailPath).SetWebThumbnailPath(vodDto.WebThumbnailPath).SetVideoPath(vodDto.VideoPath).SetChatPath(vodDto.ChatPath).SetChatVideoPath(vodDto.ChatVideoPath).SetInfoPath(vodDto.InfoPath).SetCaptionPath(vodDto.CaptionPath).SetStreamedAt(vodDto.StreamedAt).SetLocked(vodDto.Locked).SetClipVodOffset(vodDto.ClipVodOffset).SetClipExtVodID(vodDto.ClipExtVodID).Save(c.Request().Context())
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

func (s *Service) SearchVods(ctx context.Context, limit int, offset int, types []utils.VodType, predicates []predicate.Vod) (Pagination, error) {

	var pagination Pagination

	queryBuilder := s.Store.Client.Vod.Query().
		Where(vod.Or(predicates...)).
		Order(ent.Desc(vod.FieldStreamedAt)).
		WithChannel().
		Limit(limit).
		Offset(offset)

	if len(types) > 0 {
		queryBuilder = queryBuilder.Where(vod.TypeIn(types...))
	}

	vods, err := queryBuilder.All(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("error searching vods")
		return pagination, fmt.Errorf("error searching vods: %v", err)
	}

	countQuery := s.Store.Client.Vod.Query().Where(vod.Or(predicates...))
	if len(types) > 0 {
		countQuery = countQuery.Where(vod.TypeIn(types...))
	}

	totalCount, err := countQuery.Count(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("error getting total vod count")
		return pagination, fmt.Errorf("error getting total vod count: %v", err)
	}

	pagination.TotalCount = totalCount
	pagination.Limit = limit
	pagination.Offset = offset
	pagination.Pages = int(math.Ceil(float64(totalCount) / float64(limit)))
	pagination.Data = vods

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

func (s *Service) GetVodsPagination(c echo.Context, limit int, offset int, channelId uuid.UUID, types []utils.VodType, playlistId uuid.UUID, isProcessing bool) (Pagination, error) {
	var pagination Pagination

	if channelId != uuid.Nil && playlistId != uuid.Nil {
		return pagination, fmt.Errorf("either channelid or playlistid can be specified, not both")
	}

	// Query builder
	vodQuery := s.Store.Client.Vod.Query()

	// If channel id is not nil
	if channelId != uuid.Nil {
		vodQuery = vodQuery.Where(vod.HasChannelWith(channel.ID(channelId)))
	}

	// If playlist id is not nil
	if playlistId != uuid.Nil {
		vodQuery = vodQuery.Where(vod.HasPlaylistsWith(playlist.ID(playlistId)))
	}

	// If types is not nil
	if len(types) > 0 {
		vodQuery = vodQuery.Where(vod.TypeIn(types...))
	}

	// If processing is true
	if !isProcessing {
		vodQuery = vodQuery.Where(vod.Processing(isProcessing))
	}

	v, err := vodQuery.Order(ent.Desc(vod.FieldStreamedAt)).Limit(limit).Offset(offset).WithChannel().All(c.Request().Context())
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

	// If playlist id is not nil
	if playlistId != uuid.Nil {
		totalCountQuery = totalCountQuery.Where(vod.HasPlaylistsWith(playlist.ID(playlistId)))
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

func (s *Service) GenerateStaticThumbnail(ctx context.Context, videoID uuid.UUID) (*rivertype.JobInsertResult, error) {
	return s.RiverClient.Client.Insert(ctx, tasks.GenerateStaticThumbnailArgs{
		VideoId: videoID.String(),
	}, nil)
}

func (s *Service) GenerateSpriteThumbnails(ctx context.Context, videoID uuid.UUID) (*rivertype.JobInsertResult, error) {
	return s.RiverClient.Client.Insert(ctx, tasks.GenerateSpriteThumbnailArgs{
		VideoId: videoID.String(),
	}, nil)
}

func (s *Service) GetVodClips(ctx context.Context, id uuid.UUID) ([]*ent.Vod, error) {
	video, err := s.Store.Client.Vod.Query().Where(vod.ID(id)).Only(ctx)
	if err != nil {
		return nil, err
	}

	clips, err := s.Store.Client.Vod.Query().Where(vod.ClipExtVodID(video.ExtID)).All(ctx)
	if err != nil {
		return nil, err
	}
	return clips, nil
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
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return nil, fmt.Errorf("error getting vod chat: %v", err)
	}

	var comments []chat.Comment
	cacheData, exists := cache.Cache().Get(v.ID.String())
	if !exists {
		err = loadChatIntoCache(v)
		if err != nil {
			log.Debug().Err(err).Msg("error loading chat into cache")
			return nil, fmt.Errorf("error loading chat into cache: %v", err)
		}
		cacheData, _ = cache.Cache().Get(v.ID.String())
	}
	comments = cacheData.([]chat.Comment)

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
	comments = nil

	defer runtime.GC()

	return &filteredComments, nil
}

func loadChatIntoCache(vod *ent.Vod) error {
	var chatData *chat.ChatNoEmotes
	var comments []chat.Comment

	data, err := utils.ReadChatFile(vod.ChatPath)
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return fmt.Errorf("error getting vod chat: %v", err)
	}
	err = json.Unmarshal(data, &chatData)
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return fmt.Errorf("error getting vod chat: %v", err)
	}

	comments = chatData.Comments
	chatData = nil
	runtime.GC()

	// Sort the comments by their content offset seconds
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].ContentOffsetSeconds < comments[j].ContentOffsetSeconds
	})

	// Set cache
	err = cache.Cache().Set(vod.ID.String(), comments, 10*time.Minute)
	if err != nil {
		log.Debug().Err(err).Msg("error setting cache")
		return fmt.Errorf("error setting cache: %v", err)
	}

	runtime.GC()

	return nil
}

func (s *Service) GetNumberOfVodChatCommentsFromTime(c echo.Context, vodID uuid.UUID, start float64, commentCount int64) (*[]chat.Comment, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(c.Request().Context())
	if err != nil {
		log.Debug().Err(err).Msg("error getting vod chat")
		return nil, fmt.Errorf("error getting vod chat: %v", err)
	}

	var comments []chat.Comment

	cacheData, exists := cache.Cache().Get(v.ID.String())
	if !exists {
		err = loadChatIntoCache(v)
		if err != nil {
			log.Debug().Err(err).Msg("error loading chat into cache")
			return nil, fmt.Errorf("error loading chat into cache: %v", err)
		}
		cacheData, _ = cache.Cache().Get(v.ID.String())
	}
	comments = cacheData.([]chat.Comment)

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
	comments = nil
	defer runtime.GC()

	return &filteredComments, nil

}

func (s *Service) GetChatEmotes(ctx context.Context, vodID uuid.UUID) (*platform.Emotes, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	data, err := utils.ReadChatFile(v.ChatPath)
	if err != nil {
		return nil, fmt.Errorf("error reading chat file: %v", err)
	}
	var chatData *chat.ChatOnlyEmotes
	err = json.Unmarshal(data, &chatData)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling chat data: %v", err)
	}

	defer runtime.GC()

	var emotes platform.Emotes

	// get streamer id from chat
	streamerId, err := getStreamerIdFromInterface(chatData.Streamer.ID)
	if err != nil {
		return nil, err
	}

	switch {
	// check if emotes are embedded in the 'emotes' struct
	case len(chatData.Emotes.FirstParty) > 0 || len(chatData.Emotes.ThirdParty) > 0:
		log.Debug().Str("video_id", v.ID.String()).Msg("chat emotes are embedded in 'emotes' struct")
		// Loop through first party emotes and add them to the emotes slice
		for _, emote := range chatData.Emotes.FirstParty {
			emotes.Emotes = append(emotes.Emotes, platform.Emote{
				ID:     emote.ID,
				Name:   fmt.Sprint(emote.Name),
				URL:    emote.Data,
				Width:  emote.Width,
				Height: emote.Height,
				Type:   "embed",
			})
		}
		// Loop through third party emotes and add them to the emotes slice
		for _, emote := range chatData.Emotes.ThirdParty {
			emotes.Emotes = append(emotes.Emotes, platform.Emote{
				ID:     emote.ID,
				Name:   fmt.Sprint(emote.Name),
				URL:    emote.Data,
				Width:  emote.Width,
				Height: emote.Height,
				Type:   "embed",
			})
		}
	case len(chatData.EmbeddedData.FirstParty) > 0 || len(chatData.EmbeddedData.ThirdParty) > 0:
		log.Debug().Str("video_id", v.ID.String()).Msg("chat emotes are embedded in 'embeddedData' struct")
		// Loop through first party emotes and add them to the emotes slice
		for _, emote := range chatData.EmbeddedData.FirstParty {
			emotes.Emotes = append(emotes.Emotes, platform.Emote{
				ID:     emote.ID,
				Name:   fmt.Sprint(emote.Name),
				URL:    emote.Data,
				Width:  emote.Width,
				Height: emote.Height,
				Type:   "embed",
			})
		}
		// Loop through third party emotes and add them to the emotes slice
		for _, emote := range chatData.EmbeddedData.ThirdParty {
			emotes.Emotes = append(emotes.Emotes, platform.Emote{
				ID:     emote.ID,
				Name:   fmt.Sprint(emote.Name),
				URL:    emote.Data,
				Width:  emote.Width,
				Height: emote.Height,
				Type:   "embed",
			})
		}
		// no embedded emotes; fetch emotes from remote providers
	default:
		log.Debug().Str("video_id", v.ID.String()).Msg("chat emotes are not embedded; fetching emotes from remote providers")

		// get platform global emotes
		globalEmotes, err := s.Platform.GetGlobalEmotes(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting global emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, globalEmotes...)

		// get platform channel emotes
		channelEmotes, err := s.Platform.GetChannelEmotes(ctx, streamerId)
		if err != nil {
			return nil, fmt.Errorf("error getting channel emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, channelEmotes...)

		// get 7tv emotes
		sevenTVGlobalEmotes, err := chat.Get7TVGlobalEmotes(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting 7tv global emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, sevenTVGlobalEmotes...)

		sevenTVChannelEmotes, err := chat.Get7TVChannelEmotes(ctx, streamerId)
		if err != nil {
			return nil, fmt.Errorf("error getting 7tv channel emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, sevenTVChannelEmotes...)

		// get bttv emotes
		bttvGlobalEmotes, err := chat.GetBTTVGlobalEmotes(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting bttv global emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, bttvGlobalEmotes...)

		bttvChannelEmotes, err := chat.GetBTTVChannelEmotes(ctx, streamerId)
		if err != nil {
			return nil, fmt.Errorf("error getting bttv channel emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, bttvChannelEmotes...)

		// get ffz emotes
		ffzGlobalEmotes, err := chat.GetFFZGlobalEmotes(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting ffz global emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, ffzGlobalEmotes...)

		ffzChannelEmotes, err := chat.GetFFZChannelEmotes(ctx, streamerId)
		if err != nil {
			return nil, fmt.Errorf("error getting ffz channel emotes: %v", err)
		}
		emotes.Emotes = append(emotes.Emotes, ffzChannelEmotes...)
	}

	chatData = nil

	defer runtime.GC()
	return &emotes, nil

}

func (s *Service) GetChatBadges(ctx context.Context, vodID uuid.UUID) (*platform.Badges, error) {
	v, err := s.Store.Client.Vod.Query().Where(vod.ID(vodID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}
	data, err := utils.ReadChatFile(v.ChatPath)
	if err != nil {
		return nil, fmt.Errorf("error getting vod chat emotes: %v", err)
	}

	var chatData *chat.ChatOnlyBadges
	var chatDataOld *chat.ChatOnlyBadgesOld
	err = json.Unmarshal(data, &chatData)
	if err != nil {
		// attempt to unmarshal old format
		err = json.Unmarshal(data, &chatDataOld)
		if err != nil {
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

	var badgeResp platform.Badges

	// If emebedded badges
	if len(chatData.EmbeddedData.TwitchBadges) != 0 {
		log.Debug().Str("vod_id", vodID.String()).Msg("Found embedded badges")
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
						badgeResp.Badges = append(badgeResp.Badges, platform.Badge{
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
						badgeResp.Badges = append(badgeResp.Badges, platform.Badge{
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
						badgeResp.Badges = append(badgeResp.Badges, platform.Badge{
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
						badgeResp.Badges = append(badgeResp.Badges, platform.Badge{
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
		log.Debug().Str("vod_id", vodID.String()).Msg("No embedded badges found; fetching from provider")
		// get streamer id from chat
		streamerId, err := getStreamerIdFromInterface(chatData.Streamer.ID)
		if err != nil {
			return nil, err
		}

		twitchBadges, err := s.Platform.GetGlobalBadges(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting twitch global badges: %v", err)
		}
		badgeResp.Badges = append(badgeResp.Badges, twitchBadges...)
		channelBadges, err := s.Platform.GetChannelBadges(ctx, streamerId)
		if err != nil {
			return nil, fmt.Errorf("error getting twitch channel badges: %v", err)
		}
		badgeResp.Badges = append(badgeResp.Badges, channelBadges...)
	}

	chatData = nil
	defer runtime.GC()

	return &badgeResp, nil
}

// LockVod locks or unlocks a VOD
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

// getStreamerIdFromInterface returns the string representation of the streamer id
//
// Older chat files have the streamer ID stored as an int, need to convert to a string
func getStreamerIdFromInterface(id interface{}) (string, error) {
	var streamerId string
	switch i := id.(type) {
	case string:
		streamerId = i
	case int:
		streamerId = strconv.Itoa(i)
	case int64:
		streamerId = strconv.FormatInt(i, 10)
	case float64:
		streamerId = strconv.FormatFloat(i, 'f', -1, 64)
	default:
		return "", fmt.Errorf("unsupported streamer id type: %T", streamerId)
	}
	return streamerId, nil
}

// GetVodChatHistogram returns a histogram of chat messages for a VOD
func (s *Service) GetVodChatHistogram(ctx context.Context, videoId uuid.UUID, resolutionSeconds float64) (map[int]int, error) {
	if resolutionSeconds <= 0 {
		return nil, fmt.Errorf("resolutionSeconds must be greater than 0")
	}

	video, err := s.Store.Client.Vod.Query().Where(vod.ID(videoId)).Only(ctx)
	if err != nil {
		return nil, err
	}

	cacheData, exists := cache.Cache().Get(video.ID.String())
	if !exists {
		err = loadChatIntoCache(video)
		if err != nil {
			log.Debug().Err(err).Msg("error loading chat into cache")
			return nil, fmt.Errorf("error loading chat into cache: %v", err)
		}
		cacheData, _ = cache.Cache().Get(video.ID.String())
	}
	comments := cacheData.([]chat.Comment)

	histogram := make(map[int]int)

	// Populate histogram with bucket start times as keys
	for _, comment := range comments {
		if comment.ContentOffsetSeconds < 0 || comment.ContentOffsetSeconds > float64(video.Duration) {
			continue
		}

		// Calculate the bucket's start time as an integer
		bucketStart := int(math.Floor(comment.ContentOffsetSeconds/resolutionSeconds) * resolutionSeconds)
		histogram[bucketStart]++
	}

	// Convert the histogram to a sorted map
	sortedHistogram := make(map[int]int)
	keys := make([]int, 0, len(histogram))
	for k := range histogram {
		keys = append(keys, k)
	}
	sort.Ints(keys) // Sort the bucket start times

	for _, k := range keys {
		sortedHistogram[k] = histogram[k]
	}

	return sortedHistogram, nil
}
