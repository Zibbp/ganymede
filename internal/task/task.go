package task

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	tasks_periodic "github.com/zibbp/ganymede/internal/tasks/periodic"
)

type Service struct {
	Store       *database.Database
	LiveService *live.Service
	RiverClient *tasks_client.RiverClient
}

func NewService(store *database.Database, liveService *live.Service, riverClient *tasks_client.RiverClient) *Service {
	return &Service{Store: store, LiveService: liveService, RiverClient: riverClient}
}

func (s *Service) StartTask(ctx context.Context, task string) error {
	log.Info().Msgf("manually starting task %s", task)

	switch task {
	case "check_live":
		err := s.LiveService.Check(ctx)
		if err != nil {
			return fmt.Errorf("error checking live: %v", err)
		}

	case "check_vod":
		task, err := s.RiverClient.Client.Insert(ctx, tasks_periodic.CheckChannelsForNewVideosArgs{}, nil)
		if err != nil {
			return fmt.Errorf("error inserting task: %v", err)
		}
		log.Info().Str("task_id", fmt.Sprintf("%d", task.Job.ID)).Msgf("task created")

	case "get_jwks":
		task, err := s.RiverClient.Client.Insert(ctx, tasks_periodic.FetchJWKSArgs{}, nil)
		if err != nil {
			return fmt.Errorf("error inserting task: %v", err)
		}
		log.Info().Str("task_id", fmt.Sprintf("%d", task.Job.ID)).Msgf("task created")

	case "storage_migration":
		go func() {
			err := s.StorageMigration()
			if err != nil {
				log.Error().Err(err).Msg("error migrating storage")
			}
		}()

	case "prune_videos":
		task, err := s.RiverClient.Client.Insert(ctx, tasks_periodic.PruneVideosArgs{}, nil)
		if err != nil {
			return fmt.Errorf("error inserting task: %v", err)
		}
		log.Info().Str("task_id", fmt.Sprintf("%d", task.Job.ID)).Msgf("task created")

	case "save_chapters":
		task, err := s.RiverClient.Client.Insert(ctx, tasks_periodic.SaveVideoChaptersArgs{}, nil)
		if err != nil {
			return fmt.Errorf("error inserting task: %v", err)
		}
		log.Info().Str("task_id", fmt.Sprintf("%d", task.Job.ID)).Msgf("task created")

	case "update_stream_vod_ids":
		task, err := s.RiverClient.Client.Insert(ctx, tasks_periodic.UpdateLivestreamVodIdsArgs{}, nil)
		if err != nil {
			return fmt.Errorf("error inserting task: %v", err)
		}
		log.Info().Str("task_id", fmt.Sprintf("%d", task.Job.ID)).Msgf("task created")

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
		storageTemplateInput := archive.StorageTemplateInput{
			UUID:    video.ID,
			ID:      video.ExtID,
			Channel: video.Edges.Channel.Name,
			Title:   video.Title,
			Type:    string(video.Type),
			Date:    video.CreatedAt.Format("2006-01-02"),
		}
		folderName, err := archive.GetFolderName(video.ID, storageTemplateInput)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting folder name for video %s", video.ID)
			continue
		}
		fileName, err := archive.GetFileName(video.ID, storageTemplateInput)
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
				_, err = video.Update().SetVideoPath(fmt.Sprintf("%s/%s-video_hls/%s-video.m3u8", newRootFolderPath, fileName, video.ExtID)).Save(context.Background())
				if err != nil {
					log.Error().Err(err).Msgf("Error updating video path for video %s", video.ID)
					continue
				}
			} else {
				err := os.Rename(video.VideoPath, fmt.Sprintf("%s/%s-video%s", oldRootFolderPath, fileName, ext))
				if err != nil {
					if os.IsExist(err) {
					} else {
						log.Error().Err(err).Msgf("Error renaming %s to %s. Skipping this video", video.VideoPath, fmt.Sprintf("%s/%s-video_hls", oldRootFolderPath, fileName))
						continue
					}
				}
				_, err = video.Update().SetVideoPath(fmt.Sprintf("%s/%s-video%s", newRootFolderPath, fileName, ext)).Save(context.Background())
				if err != nil {
					log.Error().Err(err).Msgf("Error updating video path for video %s", video.ID)
					continue
				}
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
			_, err = video.Update().SetThumbnailPath(fmt.Sprintf("%s/%s-thumbnail%s", newRootFolderPath, fileName, ext)).Save(context.Background())
			if err != nil {
				log.Error().Err(err).Msgf("Error updating thumbnail path for video %s", video.ID)
				continue
			}
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
			_, err = video.Update().SetWebThumbnailPath(fmt.Sprintf("%s/%s-web_thumbnail%s", newRootFolderPath, fileName, ext)).Save(context.Background())
			if err != nil {
				log.Error().Err(err).Msgf("Error updating web thumbnail path for video %s", video.ID)
				continue
			}
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
			_, err = video.Update().SetChatPath(fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, ext)).Save(context.Background())
			if err != nil {
				log.Error().Err(err).Msgf("Error updating chat path for video %s", video.ID)
				continue
			}
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
			_, err = video.Update().SetChatVideoPath(fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, ext)).Save(context.Background())
			if err != nil {
				log.Error().Err(err).Msgf("Error updating chat video path for video %s", video.ID)
				continue
			}
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
			_, err = video.Update().SetInfoPath(fmt.Sprintf("%s/%s-info%s", newRootFolderPath, fileName, ext)).Save(context.Background())
			if err != nil {
				log.Error().Err(err).Msgf("Error updating info path for video %s", video.ID)
				continue
			}
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
