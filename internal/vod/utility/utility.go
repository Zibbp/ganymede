package vods_utility

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChapter "github.com/zibbp/ganymede/ent/chapter"
	entMutedSegment "github.com/zibbp/ganymede/ent/mutedsegment"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/utils"
)

// DeleteVod deletes a VOD and its associated files from the database and filesystem.
// This is in a separate package to avoid circular dependencies with the vod service.
func DeleteVod(ctx context.Context, store *database.Database, vodID uuid.UUID, deleteFiles bool) error {

	log.Debug().Msgf("deleting vod %s", vodID)
	// delete vod and queue item
	v, err := store.Client.Vod.Query().Where(vod.ID(vodID)).WithQueue().WithChannel().WithChapters().WithMutedSegments().Only(ctx)
	if err != nil {
		if _, ok := err.(*ent.NotFoundError); ok {
			return fmt.Errorf("vod not found")
		}
		return fmt.Errorf("error deleting vod: %v", err)
	}
	if v.Edges.Queue != nil {
		err = store.Client.Queue.DeleteOneID(v.Edges.Queue.ID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error deleting queue item: %v", err)
		}
	}
	if v.Edges.Chapters != nil {
		_, err = store.Client.Chapter.Delete().Where(entChapter.HasVodWith(vod.ID(vodID))).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error deleting chapters: %v", err)
		}
	}
	if v.Edges.MutedSegments != nil {
		_, err = store.Client.MutedSegment.Delete().Where(entMutedSegment.HasVodWith(vod.ID(vodID))).Exec(ctx)
		if err != nil {
			return fmt.Errorf("error deleting muted segments: %v", err)
		}
	}

	// delete files
	if deleteFiles {
		log.Info().Msgf("deleting files for vod %s", v.ID)

		// Use the videopath for standard videos
		// If HLS video use the path of the HLS directory
		videoPath := v.VideoPath
		if v.VideoHlsPath != "" {
			videoPath = v.VideoHlsPath
		}

		path := filepath.Dir(filepath.Clean(videoPath))

		// Make sure FolderName is present in the path before deleting
		// This is to prevent accidental deletion of unrelated directories
		if v.FolderName != "" {
			if !strings.Contains(path, v.FolderName) {
				log.Warn().Msgf("video folder_name not found in path, cowardly refusing to delete: %s. Delete video without deleting files then manually delete directory", path)
				return fmt.Errorf("video folder_name not found in path, cowardly refusing to delete: %s. Delete video without deleting files then manually delete directory", path)
			}
		}

		log.Info().Msgf("deleting directory %s", path)

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

	err = store.Client.Vod.DeleteOneID(vodID).Exec(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("error deleting vod")
		return fmt.Errorf("error deleting vod: %v", err)
	}
	return nil
}
