package task

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/archive"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/live"
	"github.com/zibbp/ganymede/internal/tasks"
	tasks_client "github.com/zibbp/ganymede/internal/tasks/client"
	tasks_periodic "github.com/zibbp/ganymede/internal/tasks/periodic"
	"github.com/zibbp/ganymede/internal/utils"
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

	case "check_clips":
		task, err := s.RiverClient.Client.Insert(ctx, tasks_periodic.TaskCheckChannelForNewClipsArgs{}, nil)
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
		task, err := s.RiverClient.Client.Insert(ctx, tasks.UpdateStreamVideoIdArgs{Input: tasks.ArchiveVideoInput{QueueId: uuid.Nil}}, nil)
		if err != nil {
			return fmt.Errorf("error inserting task: %v", err)
		}
		log.Info().Str("task_id", fmt.Sprintf("%d", task.Job.ID)).Msgf("task created")

	case "generate_sprite_thumbnails":
		videos, err := s.Store.Client.Vod.Query().Where(vod.SpriteThumbnailsEnabledEQ(false)).All(ctx)
		if err != nil {
			return fmt.Errorf("error getting videos: %v", err)
		}

		for _, video := range videos {
			_, err := s.RiverClient.Client.Insert(ctx, tasks.GenerateSpriteThumbnailArgs{VideoId: video.ID.String()}, nil)
			if err != nil {
				return fmt.Errorf("error inserting task: %v", err)
			}
		}

		log.Info().Msgf("created %d sprite thumbnail tasks", len(videos))

	case "update_video_storage_usage":
		task, err := s.RiverClient.Client.Insert(ctx, tasks.UpdateVideoStorageUsage{}, nil)
		if err != nil {
			return fmt.Errorf("error inserting task: %v", err)
		}
		log.Info().Str("task_id", fmt.Sprintf("%d", task.Job.ID)).Msgf("task created")
	}

	return nil
}

// renameOperation represents a single successful rename (old->new).
type renameOperation struct {
	oldPath string
	newPath string
}

// rollbackRenames will undo the renames in reverse order.
func rollbackRenames(ops []renameOperation) {
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		if err := os.Rename(op.newPath, op.oldPath); err != nil {
			log.Error().Err(err).Msgf("error rolling back rename from %s to %s", op.newPath, op.oldPath)
		} else {
			log.Info().Msgf("rolled back rename from %s to %s", op.newPath, op.oldPath)
		}
	}
}

// StorageMigration migrates video files to a new storage template. Files are moved first and then the database is updated. If any step fails, the operation is rolled back.
func (s *Service) StorageMigration() error {
	// Get all videos from the database.
	videos, err := s.Store.Client.Vod.Query().WithChannel().All(context.Background())
	if err != nil {
		return fmt.Errorf("error getting videos: %v", err)
	}

	// Loop through each video.
	for _, video := range videos {
		// Prepare the template input.
		storageTemplateInput := archive.StorageTemplateInput{
			UUID:    video.ID,
			ID:      video.ExtID,
			Channel: video.Edges.Channel.Name,
			Title:   video.Title,
			Type:    string(video.Type),
			Date:    video.StreamedAt.Format("2006-01-02"),
		}

		// Get folder and file names.
		folderName, err := archive.GetFolderName(video.ID, storageTemplateInput)
		if err != nil {
			log.Error().Err(err).Msgf("error getting folder name for video %s", video.ID)
			continue
		}
		fileName, err := archive.GetFileName(video.ID, storageTemplateInput)
		if err != nil {
			log.Error().Err(err).Msgf("error getting file name for video %s", video.ID)
			continue
		}

		// Extract old root folder (using video path as reference if not using HLS).
		var oldRootFolderPath string
		if video.VideoHlsPath != "" {
			oldRootFolderPath = path.Dir(video.ThumbnailPath)
		} else {
			oldRootFolderPath = path.Dir(video.VideoPath)
		}

		envConfig := config.GetEnvConfig()
		newRootFolderPath := fmt.Sprintf("%s/%s/%s", envConfig.VideosDir, video.Edges.Channel.Name, folderName)

		// We'll record each successful rename here.
		var renames []renameOperation

		// safeRename is a helper to perform the os.Rename and record the operation.
		safeRename := func(oldPath, newPath string) error {
			err := os.Rename(oldPath, newPath)
			if err != nil {
				// If the error is that the target already exists, we consider that acceptable.
				if os.IsExist(err) {
					return nil
				} else if os.IsNotExist(err) {
					// If the file doesn't exist, we'll just skip it. This can happen if the path is in the database yet the file doesn't really exist (e.g. live chat for VODs might be populated in old versions).
					log.Warn().Msgf("file %s does not exist, skipping", oldPath)
					return nil
				}
				return fmt.Errorf("failed to rename %s to %s: %w", oldPath, newPath, err)
			}
			renames = append(renames, renameOperation{oldPath: oldPath, newPath: newPath})
			return nil
		}

		if err := os.MkdirAll(newRootFolderPath, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Perform renames for each file if a path is provided.
		// Video file
		if video.VideoPath != "" {
			ext := path.Ext(video.VideoPath)
			if ext == ".m3u8" {
				oldHlsVideoRootPath := path.Dir(video.VideoPath)
				newHlsVideoRootPath := fmt.Sprintf("%s/%s-video_hls", newRootFolderPath, fileName)
				if err := safeRename(oldHlsVideoRootPath, newHlsVideoRootPath); err != nil {
					log.Error().Err(err).Msgf("error renaming hls video directory for video %s", video.ID)
					rollbackRenames(renames)
					continue
				}
			} else {
				newVideoPath := fmt.Sprintf("%s/%s-video%s", newRootFolderPath, fileName, ext)
				if err := safeRename(video.VideoPath, newVideoPath); err != nil {
					log.Error().Err(err).Msgf("error renaming video file for video %s", video.ID)
					rollbackRenames(renames)
					continue
				}
			}
		}

		// Thumbnail
		if video.ThumbnailPath != "" {
			newPath := fmt.Sprintf("%s/%s-thumbnail%s", newRootFolderPath, fileName, path.Ext(video.ThumbnailPath))
			if err := safeRename(video.ThumbnailPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming thumbnail for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Web Thumbnail
		if video.WebThumbnailPath != "" {
			newPath := fmt.Sprintf("%s/%s-web_thumbnail%s", newRootFolderPath, fileName, path.Ext(video.WebThumbnailPath))
			if err := safeRename(video.WebThumbnailPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming web thumbnail for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Chat file
		if video.ChatPath != "" {
			newPath := fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, path.Ext(video.ChatPath))
			if err := safeRename(video.ChatPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming chat file for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Chat video file
		if video.ChatVideoPath != "" {
			newPath := fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, path.Ext(video.ChatVideoPath))
			if err := safeRename(video.ChatVideoPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming chat video for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Live Chat file
		if video.Type == utils.Live && video.LiveChatPath != "" {
			newPath := fmt.Sprintf("%s/%s-live-chat%s", newRootFolderPath, fileName, path.Ext(video.LiveChatPath))
			if err := safeRename(video.LiveChatPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming live chat for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Live Chat Convert file
		if video.Type == utils.Live && video.LiveChatConvertPath != "" {
			newPath := fmt.Sprintf("%s/%s-live-chat-convert%s", newRootFolderPath, fileName, path.Ext(video.LiveChatConvertPath))
			if err := safeRename(video.LiveChatConvertPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming live chat convert for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Info file
		if video.InfoPath != "" {
			newPath := fmt.Sprintf("%s/%s-info%s", newRootFolderPath, fileName, path.Ext(video.InfoPath))
			if err := safeRename(video.InfoPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming info file for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Caption file
		if video.CaptionPath != "" {
			newPath := fmt.Sprintf("%s/%s-caption%s", newRootFolderPath, fileName, path.Ext(video.CaptionPath))
			if err := safeRename(video.CaptionPath, newPath); err != nil {
				log.Error().Err(err).Msgf("error renaming caption for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Sprite thumbnails directory
		if len(video.SpriteThumbnailsImages) > 0 {
			spriteThumbnailRoot := strings.Split(video.SpriteThumbnailsImages[0], "/sprites")[0]
			oldSpriteThumbnailRootPath := fmt.Sprintf("%s/sprites", spriteThumbnailRoot)
			newSpriteThumbnailRootPath := fmt.Sprintf("%s/sprites", newRootFolderPath)
			if err := safeRename(oldSpriteThumbnailRootPath, newSpriteThumbnailRootPath); err != nil {
				log.Error().Err(err).Msgf("error renaming sprite thumbnails directory for video %s", video.ID)
				rollbackRenames(renames)
				continue
			}
		}

		// Begin a DB transaction to update the file paths.
		tx, err := s.Store.Client.Tx(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("error starting transaction")
			rollbackRenames(renames)
			continue
		}

		// Build the update query with the new paths.
		update := tx.Vod.UpdateOne(video)
		if video.VideoPath != "" {
			ext := path.Ext(video.VideoPath)
			if ext == ".m3u8" {
				newHlsVideoRootPath := fmt.Sprintf("%s/%s-video_hls", newRootFolderPath, fileName)
				update = update.SetVideoPath(fmt.Sprintf("%s/%s-video.m3u8", newHlsVideoRootPath, video.ExtID))
				update = update.SetVideoHlsPath(newHlsVideoRootPath)
			} else {
				update = update.SetVideoPath(fmt.Sprintf("%s/%s-video%s", newRootFolderPath, fileName, ext))
			}
		}
		if video.ThumbnailPath != "" {
			update = update.SetThumbnailPath(fmt.Sprintf("%s/%s-thumbnail%s", newRootFolderPath, fileName, path.Ext(video.ThumbnailPath)))
		}
		if video.WebThumbnailPath != "" {
			update = update.SetWebThumbnailPath(fmt.Sprintf("%s/%s-web_thumbnail%s", newRootFolderPath, fileName, path.Ext(video.WebThumbnailPath)))
		}
		if video.ChatPath != "" {
			update = update.SetChatPath(fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, path.Ext(video.ChatPath)))
		}
		if video.ChatVideoPath != "" {
			update = update.SetChatVideoPath(fmt.Sprintf("%s/%s-chat%s", newRootFolderPath, fileName, path.Ext(video.ChatVideoPath)))
		}
		if video.LiveChatPath != "" {
			update = update.SetLiveChatPath(fmt.Sprintf("%s/%s-live-chat%s", newRootFolderPath, fileName, path.Ext(video.LiveChatPath)))
		}
		if video.LiveChatConvertPath != "" {
			update = update.SetLiveChatConvertPath(fmt.Sprintf("%s/%s-live-chat-convert%s", newRootFolderPath, fileName, path.Ext(video.LiveChatConvertPath)))
		}
		if video.InfoPath != "" {
			update = update.SetInfoPath(fmt.Sprintf("%s/%s-info%s", newRootFolderPath, fileName, path.Ext(video.InfoPath)))
		}
		if video.CaptionPath != "" {
			update = update.SetCaptionPath(fmt.Sprintf("%s/%s-caption%s", newRootFolderPath, fileName, path.Ext(video.CaptionPath)))
		}
		if len(video.SpriteThumbnailsImages) > 0 {
			var newSpriteThumbs []string
			for _, thumb := range video.SpriteThumbnailsImages {
				newSpriteThumbs = append(newSpriteThumbs, fmt.Sprintf("%s/sprites/%s", newRootFolderPath, path.Base(thumb)))
			}
			update = update.SetSpriteThumbnailsImages(newSpriteThumbs)
		}

		// Save the updates.
		if _, err := update.Save(context.Background()); err != nil {
			log.Error().Err(err).Msgf("error updating database paths for video %s", video.ID)
			if err := tx.Rollback(); err != nil {
				log.Error().Err(err).Msg("error rolling back transaction")
			}
			rollbackRenames(renames)
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error().Err(err).Msg("error committing transaction")
			rollbackRenames(renames)
			continue
		}

		log.Info().Msgf("migrated video %s to new storage template", video.ID)

		// Remove old root path if it's empty.
		if err := os.Remove(oldRootFolderPath); err != nil {
			log.Warn().Err(err).Msgf("error removing old root folder '%s' likely files still exist in there", oldRootFolderPath)
		}

	}

	return nil
}
