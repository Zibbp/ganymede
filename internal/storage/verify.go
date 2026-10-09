package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
)

// VerifyFinding checks, right before an action, that a directory is still what the scan
// reported: inside the videos directory, a real directory, holding video files, not written
// to recently, and with no video in the database referencing it or anything below it. It is
// the targeted counterpart of the scan and does not walk the library.
func VerifyFinding(ctx context.Context, store *database.Database, videosDir string, path string) error {
	cleaned := filepath.Clean(path)

	if !filepath.IsAbs(cleaned) {
		return errors.New("path is not absolute")
	}
	if _, ok := insideVideosDir(cleaned, videosDir); !ok {
		return errors.New("path is outside of the videos directory")
	}
	// The entries of the videos directory are never findings.
	if filepath.Dir(cleaned) == videosDir {
		return errors.New("path is a top level directory")
	}

	// Another configured data directory is not a leftover, wherever the operator put it.
	env := config.GetEnvConfig()
	for _, dir := range []string{env.TempDir, env.ConfigDir, env.LogsDir} {
		if dir == "" {
			continue
		}
		dir = filepath.Clean(dir)
		if cleaned == dir || strings.HasPrefix(dir, cleaned+string(filepath.Separator)) || strings.HasPrefix(cleaned, dir+string(filepath.Separator)) {
			return errors.New("path is inside a configured data directory")
		}
	}

	// Lstat so that a symlink is never followed.
	info, err := os.Lstat(cleaned)
	if err != nil {
		return fmt.Errorf("error reading path: %w", err)
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}

	// Anything a video stores a path to, in or below this directory, protects it. A column can
	// also hold the directory itself, video_hls_path does.
	prefix := cleaned + string(filepath.Separator)
	referenced, err := store.Client.Vod.Query().Where(vod.Or(
		vod.VideoHlsPathEQ(cleaned),
		vod.VideoPathHasPrefix(prefix),
		vod.VideoHlsPathHasPrefix(prefix),
		vod.ThumbnailPathHasPrefix(prefix),
		vod.WebThumbnailPathHasPrefix(prefix),
		vod.ChatPathHasPrefix(prefix),
		vod.LiveChatPathHasPrefix(prefix),
		vod.LiveChatConvertPathHasPrefix(prefix),
		vod.ChatVideoPathHasPrefix(prefix),
		vod.InfoPathHasPrefix(prefix),
		vod.CaptionPathHasPrefix(prefix),
	)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("error checking videos: %w", err)
	}
	if referenced {
		return errors.New("a video in the database references this directory")
	}

	// Nothing inside the directory of a video is a leftover, and the scan never looks there.
	if err := checkAncestors(ctx, store, videosDir, cleaned); err != nil {
		return err
	}

	fileNames, err := knownFileNames(ctx, store)
	if err != nil {
		return err
	}

	orphan, err := isOrphanedDirectory(cleaned, fileNames)
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}
	if !orphan {
		return errors.New("directory is no longer a finding")
	}

	return nil
}

// checkAncestors refuses a directory that sits inside the directory of a video: the scan never
// enters one, so a path that claims to be a finding in there did not come from it.
func checkAncestors(ctx context.Context, store *database.Database, videosDir string, path string) error {
	for ancestor := filepath.Dir(path); len(ancestor) > len(videosDir); ancestor = filepath.Dir(ancestor) {
		holdsVideo, err := holdsVideoFiles(ancestor)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", ancestor, err)
		}
		if !holdsVideo {
			continue
		}

		prefix := ancestor + string(filepath.Separator)
		referenced, err := store.Client.Vod.Query().Where(vod.Or(
			vod.VideoPathHasPrefix(prefix),
			vod.VideoHlsPathHasPrefix(prefix),
			vod.ThumbnailPathHasPrefix(prefix),
			vod.WebThumbnailPathHasPrefix(prefix),
			vod.InfoPathHasPrefix(prefix),
		)).Exist(ctx)
		if err != nil {
			return fmt.Errorf("error checking videos: %w", err)
		}
		if referenced {
			return errors.New("path is inside the directory of a video")
		}
	}

	return nil
}

// knownFileNames returns the file name prefix of every video in the database.
func knownFileNames(ctx context.Context, store *database.Database) (map[string]struct{}, error) {
	var rows []struct {
		FileName string `json:"file_name"`
	}
	if err := store.Client.Vod.Query().Select(vod.FieldFileName).Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("error getting file names: %w", err)
	}

	fileNames := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.FileName != "" {
			fileNames[row.FileName] = struct{}{}
		}
	}
	return fileNames, nil
}
