package task

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/auth"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/twitch"
)

type Service struct {
	Store          *database.Database
	LiveService    *live.Service
	ArchiveService *archive.Service
}

func NewService(store *database.Database, liveService *live.Service, archiveService *archive.Service) *Service {
	return &Service{Store: store, LiveService: liveService, ArchiveService: archiveService}
}

func (s *Service) StartTask(c echo.Context, task string) error {
	log.Info().Msgf("Manually starting task %s", task)

	switch task {
	case "check_live":
		err := s.LiveService.Check()
		if err != nil {
			return fmt.Errorf("error checking live: %v", err)
		}

	case "check_vod":
		go s.LiveService.CheckVodWatchedChannels()

	case "get_jwks":
		err := auth.FetchJWKS()
		if err != nil {
			return fmt.Errorf("error fetching jwks: %v", err)
		}

	case "twitch_auth":
		err := twitch.Authenticate()
		if err != nil {
			return fmt.Errorf("error authenticating twitch: %v", err)
		}

	case "queue_hold_check":
		go s.ArchiveService.CheckOnHold()

	case "storage_migration":
		go s.StorageMigration()

	}

	return nil
}

func (s *Service) StorageMigration() error {
	// Get all videos in db
	videos, err := s.Store.Client.Vod.Query().WithChannel().All(context.Background())
	if err != nil {
		return fmt.Errorf("error getting videos: %v", err)
	}

	// Loop through videos and move them to new storage
	for _, video := range videos {

		// Populate templates
		vDto := twitch.Vod{
			ID:        video.ExtID,
			UserLogin: video.Edges.Channel.Name,
			Title:     video.Title,
			Type:      string(video.Type),
			CreatedAt: video.StreamedAt.Format(time.RFC3339),
		}
		folderName, err := archive.GetFolderName(video.ID, vDto)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting folder name for video %s", video.ID)
			continue
		}
		fileName, err := archive.GetFileName(video.ID, vDto)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting file name for video %s", video.ID)
			continue
		}

		// Extract parts of path
		// Use video path as that will always be available
		tmpRootFolder := strings.SplitN(video.VideoPath, "/", 6)[0:4]
		// Add array of strings together seperated by /
		oldRootFolderPath := strings.Join(tmpRootFolder, "/")

		newRootFolderPath := fmt.Sprintf("/vods/%s/%s", video.Edges.Channel.Name, folderName)

		// Rename files first
		// Video
		if video.VideoPath != "" {
			ext := path.Ext(video.VideoPath)
			if ext == ".m3u8" {
				parts := strings.Split(video.VideoPath, "/")
				path := strings.Join(parts[:len(parts)-1], "/")
				err := os.Rename(path, fmt.Sprintf("%s/%s-video_hls", oldRootFolderPath, fileName))
				if err != nil {
					if os.IsExist(err) {
					} else {
						log.Error().Err(err).Msgf("Error renaming %s to %s. Skipping this video", path, fmt.Sprintf("%s/%s-video_hls", oldRootFolderPath, fileName))
						continue
					}
				}
				video.Update().SetVideoPath(fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", newRootFolderPath, fileName, video.ExtID)).Save(context.Background())
			} else {
				err := os.Rename(video.VideoPath, fmt.Sprintf("%s/%s-video%s", oldRootFolderPath, fileName, ext))
				if err != nil {
					if os.IsExist(err) {
					} else {
						log.Error().Err(err).Msgf("Error renaming %s to %s. Skipping this video", video.VideoPath, fmt.Sprintf("%s/%s-video_hls", oldRootFolderPath, fileName))
						continue
					}
				}
				video.Update().SetVideoPath(fmt.Sprintf("%s/%s-video%s", newRootFolderPath, fileName, ext)).Save(context.Background())
			}
		}
		// Thumbnail
		if video.ThumbnailPath != "" {
			ext := path.Ext(video.ThumbnailPath)
			err := os.Rename(video.ThumbnailPath, fmt.Sprintf("%s/%s-thumbnail%s", oldRootFolderPath, fileName, ext))
			if err != nil {
				if os.IsExist(err) {
				} else {
					log.Error().Err(err).Msgf("Error renaming %s to %s.", video.ThumbnailPath, fmt.Sprintf("%s/%s-thumbnail%s", oldRootFolderPath, fileName, ext))
				}

			}
			video.Update().SetThumbnailPath(fmt.Sprintf("%s/%s-thumbnail%s", newRootFolderPath, fileName, ext)).Save(context.Background())
		}
		// Web Thumbnail
		if video.WebThumbnailPath != "" {
			ext := path.Ext(video.WebThumbnailPath)
			err := os.Rename(video.WebThumbnailPath, fmt.Sprintf("%s/%s-web_thumbnail%s", oldRootFolderPath, fileName, ext))
			if err != nil {
				if os.IsExist(err) {
				} else {
					log.Error().Err(err).Msgf("Error renaming %s to %s.", video.WebThumbnailPath, fmt.Sprintf("%s/%s-web_thumbnail%s", oldRootFolderPath, fileName, ext))
				}

			}
			video.Update().SetWebThumbnailPath(fmt.Sprintf("%s/%s-web_thumbnail%s", newRootFolderPath, fileName, ext)).Save(context.Background())
		}
		// Chat Path
		if video.ChatPath != "" {
			ext := path.Ext(video.ChatPath)
			err := os.Rename(video.ChatPath, fmt.Sprintf("%s/%s-chat%s", oldRootFolderPath, fileName, ext))
			if err != nil {
				if os.IsExist(err) {
				} else {
					log.Error().Err(err).Msgf("Error renaming %s to %s.", video.ChatPath, fmt.Sprintf("%s/%s-chat%s", oldRootFolderPath, fileName, ext))
				}

			}
			video.Update().SetChatPath(fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, ext)).Save(context.Background())
		}
		// Chat Video Path
		if video.ChatVideoPath != "" {
			ext := path.Ext(video.ChatVideoPath)
			err := os.Rename(video.ChatVideoPath, fmt.Sprintf("%s/%s-chat%s", oldRootFolderPath, fileName, ext))
			if err != nil {
				if os.IsExist(err) {
				} else {
					log.Error().Err(err).Msgf("Error renaming %s to %s.", video.ChatVideoPath, fmt.Sprintf("%s/%s-chat%s", oldRootFolderPath, fileName, ext))
				}

			}
			video.Update().SetChatVideoPath(fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, ext)).Save(context.Background())
		}
		// Info Path
		if video.InfoPath != "" {
			ext := path.Ext(video.InfoPath)
			err := os.Rename(video.InfoPath, fmt.Sprintf("%s/%s-info%s", oldRootFolderPath, fileName, ext))
			if err != nil {
				if os.IsExist(err) {
				} else {
					log.Error().Err(err).Msgf("Error renaming %s to %s.", video.ChatVideoPath, fmt.Sprintf("%s/%s-info%s", oldRootFolderPath, fileName, ext))
				}

			}
			video.Update().SetInfoPath(fmt.Sprintf("%s/%s-info%s", newRootFolderPath, fileName, ext)).Save(context.Background())
		}

		// Rename root video folder
		err = os.Rename(oldRootFolderPath, newRootFolderPath)
		if err != nil {
			if os.IsExist(err) {

			} else {
				log.Error().Err(err).Msgf("Error renaming %s to %s.", oldRootFolderPath, newRootFolderPath)
			}

		}

		log.Info().Msgf("Migrated video %s to new storage template", video.ID)

	}

	return nil
}
