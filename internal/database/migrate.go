package database

import (
	"context"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/utils"
)

// VideosDirMigrate migrates the videos directory if it has changed.
// It will do nothing if the videos directory has not changed.
func (db *Database) VideosDirMigrate(ctx context.Context, videosDir string) error {
	// get latest video from database
	video, err := db.Client.Vod.Query().WithChannel().Limit(1).Order(ent.Desc("created_at")).First(ctx)
	if err != nil {
		// no videos found, likely a new instance. Return gracefully
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil
		} else {
			return err
		}
	}

	// get path of current videos directory
	oldVideoPath := utils.GetPathBefore(video.VideoPath, video.Edges.Channel.Name)
	oldVideoPath = strings.TrimRight(oldVideoPath, "/")

	// check if videos directory has changed
	if oldVideoPath != "" && oldVideoPath != videosDir {
		log.Info().Msg("detected new videos directory; migrating pathes to new directory")

		// update channel paths
		channels, err := db.Client.Channel.Query().All(ctx)
		if err != nil {
			return err
		}
		// replace old path with new path
		for _, c := range channels {
			update := db.Client.Channel.UpdateOne(c)
			update.SetImagePath(strings.Replace(c.ImagePath, oldVideoPath, videosDir, 1))

			if _, err := update.Save(ctx); err != nil {
				return err
			}
		}

		// update video paths
		videos, err := db.Client.Vod.Query().WithChannel().All(ctx)
		if err != nil {
			return err
		}
		// replace old path with new path
		for _, v := range videos {
			update := db.Client.Vod.UpdateOneID(v.ID)
			update.SetThumbnailPath(strings.Replace(v.ThumbnailPath, oldVideoPath, videosDir, 1))
			update.SetWebThumbnailPath(strings.Replace(v.WebThumbnailPath, oldVideoPath, videosDir, 1))
			update.SetVideoPath(strings.Replace(v.VideoPath, oldVideoPath, videosDir, 1))
			update.SetVideoHlsPath(strings.Replace(v.VideoHlsPath, oldVideoPath, videosDir, 1))
			update.SetChatPath(strings.Replace(v.ChatPath, oldVideoPath, videosDir, 1))
			update.SetLiveChatPath(strings.Replace(v.LiveChatPath, oldVideoPath, videosDir, 1))
			update.SetLiveChatConvertPath(strings.Replace(v.LiveChatConvertPath, oldVideoPath, videosDir, 1))
			update.SetChatVideoPath(strings.Replace(v.ChatVideoPath, oldVideoPath, videosDir, 1))
			update.SetInfoPath(strings.Replace(v.InfoPath, oldVideoPath, videosDir, 1))
			update.SetCaptionPath(strings.Replace(v.CaptionPath, oldVideoPath, videosDir, 1))

			if v.SpriteThumbnailsEnabled && len(v.SpriteThumbnailsImages) > 0 {
				var newSpriteThumbs []string
				for _, thumb := range v.SpriteThumbnailsImages {
					newSpriteThumbs = append(newSpriteThumbs, strings.Replace(thumb, oldVideoPath, videosDir, 1))
				}
				update = update.SetSpriteThumbnailsImages(newSpriteThumbs)
			}

			if _, err := update.Save(ctx); err != nil {
				return err
			}
		}

		log.Info().Msg("finished migrating existing video directories")
	}

	return nil
}

// TempDirMigrate migrates the temp directory if it has changed.
// It will do nothing if the temp directory has not changed.
func (db *Database) TempDirMigrate(ctx context.Context, tempDir string) error {
	// get latest video from database
	video, err := db.Client.Vod.Query().WithChannel().Limit(1).Order(ent.Desc("created_at")).First(ctx)
	if err != nil {
		// no videos found, likely a new instance. Return gracefully
		if _, ok := err.(*ent.NotFoundError); ok {
			return nil
		} else {
			return err
		}
	}

	if video.TmpVideoDownloadPath == "" {
		return nil
	}

	// get path of current videos directory
	oldTmpVideoDownloadPath := utils.GetPathBeforePartial(video.TmpVideoDownloadPath, video.ID.String())
	oldTmpVideoDownloadPath = strings.TrimRight(oldTmpVideoDownloadPath, "/")

	// check if videos directory has changed
	if oldTmpVideoDownloadPath != "" && oldTmpVideoDownloadPath != tempDir {
		log.Info().Msg("detected new temp path directory; migrating existing video directories")

		videos, err := db.Client.Vod.Query().WithChannel().All(ctx)
		if err != nil {
			return err
		}

		// replace old path with new path
		for _, v := range videos {
			update := db.Client.Vod.UpdateOneID(v.ID)
			update.SetTmpVideoDownloadPath(strings.Replace(v.TmpVideoDownloadPath, oldTmpVideoDownloadPath, tempDir, 1))
			update.SetTmpVideoConvertPath(strings.Replace(v.TmpVideoConvertPath, oldTmpVideoDownloadPath, tempDir, 1))
			update.SetTmpChatDownloadPath(strings.Replace(v.TmpChatDownloadPath, oldTmpVideoDownloadPath, tempDir, 1))
			update.SetTmpLiveChatDownloadPath(strings.Replace(v.TmpLiveChatDownloadPath, oldTmpVideoDownloadPath, tempDir, 1))
			update.SetTmpLiveChatConvertPath(strings.Replace(v.TmpLiveChatConvertPath, oldTmpVideoDownloadPath, tempDir, 1))
			update.SetTmpChatRenderPath(strings.Replace(v.TmpChatRenderPath, oldTmpVideoDownloadPath, tempDir, 1))
			update.SetTmpVideoHlsPath(strings.Replace(v.TmpVideoHlsPath, oldTmpVideoDownloadPath, tempDir, 1))

			if _, err := update.Save(ctx); err != nil {
				return err
			}
		}

		log.Info().Msg("finished migrating existing temp video directories")
	}

	return nil
}
