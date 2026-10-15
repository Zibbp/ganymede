package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/platform"
	"github.com/zibbp/ganymede/internal/storagetemplate"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/utils"
)

const (
	// maxInfoFileSize caps how much of an info file is read. The archiver writes a few hundred
	// kilobytes at most, and the file is whatever is on disk.
	maxInfoFileSize = 8 << 20
	// maxImportedEntries caps chapters and muted segments taken from an info file.
	maxImportedEntries = 5000
	// maxVideoDuration is a day, well above anything a stream produces.
	maxVideoDuration = 24 * 60 * 60
)

// infoFile covers the three shapes the archiver writes to <file name>-info.json: a video
// (platform.VideoInfo), a live stream (platform.LiveStreamInfo) and a clip (platform.ClipInfo).
type infoFile struct {
	ID            string                  `json:"id"`
	UserID        string                  `json:"user_id"`
	UserLogin     string                  `json:"user_login"`
	UserName      string                  `json:"user_name"`
	ChannelID     string                  `json:"channel_id"`
	ChannelName   *string                 `json:"channel_name"`
	VideoID       string                  `json:"video_id"`
	VodOffset     *int                    `json:"vod_offset"`
	Title         string                  `json:"title"`
	URL           string                  `json:"url"`
	Type          string                  `json:"type"`
	CreatedAt     time.Time               `json:"created_at"`
	StartedAt     time.Time               `json:"started_at"`
	Duration      int64                   `json:"duration"`
	ViewCount     int64                   `json:"view_count"`
	ViewerCount   int64                   `json:"viewer_count"`
	Chapters      []chapter.Chapter       `json:"chapters"`
	MutedSegments []platform.MutedSegment `json:"muted_segments"`
}

// isClip reports whether the file is a clip. Only a clip carries channel_id, and a clip that
// is not linked to a video has neither video_id nor vod_offset.
func (f infoFile) isClip() bool { return f.ChannelID != "" }

// isLive reports whether the file describes a live stream, which carries started_at instead of
// created_at and no duration at all.
func (f infoFile) isLive() bool { return !f.isClip() && f.CreatedAt.IsZero() && !f.StartedAt.IsZero() }

// videoFiles maps the files found in a directory onto the path fields of a video.
type videoFiles struct {
	fileName         string
	videoPath        string
	videoHLSPath     string
	thumbnailPath    string
	webThumbnailPath string
	chatPath         string
	chatVideoPath    string
	liveChatPath     string
	liveChatConvert  string
	infoPath         string
	captionPath      string
	hasSprites       bool
}

// importDirectory turns a directory the scan reported into a video in the database, using the
// info file the archiver wrote next to the media.
func (s *Service) importDirectory(ctx context.Context, videosDir string, dir string) error {
	files, err := collectVideoFiles(dir)
	if err != nil {
		return err
	}
	if files.infoPath == "" {
		return fmt.Errorf("no info file found, the directory cannot be imported")
	}
	if files.videoPath == "" {
		return fmt.Errorf("no video file found, the directory cannot be imported")
	}

	info, err := readInfoFile(files.infoPath)
	if err != nil {
		return err
	}
	if info.ID == "" {
		return fmt.Errorf("info file has no id")
	}
	if info.Title == "" {
		return fmt.Errorf("info file has no title")
	}
	// The archiver only writes info files for the platform it archived from. A file from
	// somewhere else would be imported under the wrong platform and looked up against the wrong
	// API for the rest of its life.
	if info.URL != "" && !isTwitchURL(info.URL) {
		return errors.New("info file does not describe a twitch video")
	}

	exists, err := s.Store.Client.Vod.Query().Where(
		vod.PlatformEQ(utils.PlatformTwitch),
		vod.Or(vod.ExtID(info.ID), vod.ExtStreamID(info.ID)),
	).Exist(ctx)
	if err != nil {
		return fmt.Errorf("error checking for existing video: %w", err)
	}
	if exists {
		return fmt.Errorf("a video with id %s already exists", info.ID)
	}

	channel, err := s.resolveChannel(ctx, videosDir, info)
	if err != nil {
		return err
	}

	videoType, duration, views, streamedAt := videoMetadata(info)

	// A live stream carries no duration, and an old info file may carry none either. The file
	// on disk knows.
	if duration <= 1 {
		if probed, err := exec.GetVideoDuration(ctx, files.videoPath); err == nil && probed > 0 {
			duration = boundedDuration(int64(probed))
		} else if err != nil {
			log.Warn().Err(err).Msgf("error reading the duration of %s", files.videoPath)
		}
	}

	folderName := filepath.Base(dir)
	if channelDir := filepath.Dir(channel.ImagePath); channelDir != "" {
		if rel, err := filepath.Rel(channelDir, dir); err == nil && !isOutside(rel) && rel != "." {
			folderName = rel
		}
	}

	tx, err := s.Store.Client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !isTxDone(err) {
			log.Error().Err(err).Msg("error rolling back import")
		}
	}()

	create := tx.Vod.Create().
		SetChannel(channel).
		SetExtID(info.ID).
		SetPlatform(utils.PlatformTwitch).
		SetType(videoType).
		SetTitle(info.Title).
		SetDuration(duration).
		SetViews(views).
		SetProcessing(false).
		SetStreamedAt(streamedAt).
		SetVideoPath(files.videoPath).
		SetWebThumbnailPath(webThumbnail(files)).
		SetFolderName(folderName).
		SetFileName(files.fileName)

	if files.videoHLSPath != "" {
		create.SetVideoHlsPath(files.videoHLSPath)
	}
	if files.thumbnailPath != "" {
		create.SetThumbnailPath(files.thumbnailPath)
	}
	if files.chatPath != "" {
		create.SetChatPath(files.chatPath)
	}
	if files.chatVideoPath != "" {
		create.SetChatVideoPath(files.chatVideoPath)
	}
	if files.liveChatPath != "" {
		create.SetLiveChatPath(files.liveChatPath)
	}
	if files.liveChatConvert != "" {
		create.SetLiveChatConvertPath(files.liveChatConvert)
	}
	if files.infoPath != "" {
		create.SetInfoPath(files.infoPath)
	}
	if files.captionPath != "" {
		create.SetCaptionPath(files.captionPath)
	}
	if info.isClip() && info.VideoID != "" {
		create.SetClipExtVodID(info.VideoID)
		if info.VodOffset != nil {
			create.SetClipVodOffset(*info.VodOffset)
		}
	}
	if info.isLive() && info.ID != "" {
		// A live archive is later matched to its video by the stream id.
		create.SetExtStreamID(info.ID)
	}

	video, err := create.Save(ctx)
	if err != nil {
		return fmt.Errorf("error creating video: %w", err)
	}

	for _, c := range info.Chapters {
		start, end, ok := boundedRange(c.Start, c.End, duration)
		if !ok {
			continue
		}
		if _, err := tx.Chapter.Create().SetType(c.Type).SetTitle(c.Title).SetStart(start).SetEnd(end).SetVod(video).Save(ctx); err != nil {
			return fmt.Errorf("error creating chapter: %w", err)
		}
	}
	for _, segment := range info.MutedSegments {
		start, end, ok := boundedRange(segment.Offset, segment.Offset+segment.Duration, duration)
		if !ok {
			continue
		}
		if _, err := tx.MutedSegment.Create().SetStart(start).SetEnd(end).SetVod(video).Save(ctx); err != nil {
			return fmt.Errorf("error creating muted segment: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing import: %w", err)
	}

	if files.hasSprites {
		// Sprites on disk without the columns that describe them are of no use, and the job
		// writes both.
		log.Info().Msgf("regenerating sprite thumbnails for imported video %s", video.ID)
	}
	if config.Get().Archive.GenerateSpriteThumbnails {
		if _, err := s.RiverClient.Insert(ctx, tasks.GenerateSpriteThumbnailArgs{VideoId: video.ID.String()}, &river.InsertOpts{}); err != nil {
			log.Warn().Err(err).Msgf("error queueing sprite thumbnails for imported video %s", video.ID)
		}
	}

	return nil
}

// collectVideoFiles maps the entries of a directory onto the files the archiver writes.
func collectVideoFiles(dir string) (videoFiles, error) {
	var files videoFiles
	// The names the media is built from. The archiver gives every file of a video the same
	// prefix, so one that disagrees with the info file, or with the other kind of media,
	// belongs to a different video.
	var flatPrefix, hlsPrefix string
	var flatVideoPath, hlsPlaylistPath string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return files, fmt.Errorf("error reading directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		if entry.IsDir() {
			switch {
			case name == spriteDirectory:
				files.hasSprites = true
			case strings.HasSuffix(name, hlsDirectorySuffix):
				if files.videoHLSPath != "" {
					return files, errors.New("the directory holds more than one hls directory")
				}
				files.videoHLSPath = path
				hlsPrefix = strings.TrimSuffix(name, hlsDirectorySuffix)
				hlsPlaylistPath = hlsPlaylist(path)
			}
			continue
		}

		switch {
		case strings.HasSuffix(name, "-info.json"):
			if files.infoPath != "" {
				return files, errors.New("the directory holds more than one info file")
			}
			files.infoPath = path
			files.fileName = strings.TrimSuffix(name, "-info.json")
		case strings.HasSuffix(name, "-thumbnail.jpg"):
			files.thumbnailPath = path
		case strings.HasSuffix(name, "-web_thumbnail.jpg"):
			files.webThumbnailPath = path
		// the live chat file also ends with -chat.json, so it has to be matched first
		case strings.HasSuffix(name, "-live-chat.json"):
			files.liveChatPath = path
		case strings.HasSuffix(name, "-chat-convert.json"):
			files.liveChatConvert = path
		case strings.HasSuffix(name, "-chat.json"):
			files.chatPath = path
		case strings.HasSuffix(name, "-chat.mp4"):
			files.chatVideoPath = path
		case strings.Contains(name, captionMarker):
			files.captionPath = path
		default:
			for _, extension := range videoFileExtensions {
				if strings.HasSuffix(name, "-video"+extension) {
					if flatVideoPath != "" {
						return files, errors.New("the directory holds more than one video file")
					}
					flatVideoPath = path
					flatPrefix = strings.TrimSuffix(name, "-video"+extension)
					break
				}
			}
		}
	}

	// A video file and an hls directory of two different videos are two videos in one directory,
	// whichever of them the import would end up using.
	if flatPrefix != "" && hlsPrefix != "" && flatPrefix != hlsPrefix {
		return files, errors.New("the directory holds more than one video file")
	}

	// An empty hls directory must not take the place of a video file that is there.
	files.videoPath = hlsPlaylistPath
	if files.videoPath == "" {
		files.videoPath = flatVideoPath
	}

	videoPrefix := hlsPrefix
	if videoPrefix == "" {
		videoPrefix = flatPrefix
	}
	if files.fileName == "" {
		files.fileName = videoPrefix
	} else if videoPrefix != "" && videoPrefix != files.fileName {
		return files, errors.New("the info file and the video file belong to different videos")
	}

	return files, nil
}

// hlsPlaylist returns the playlist inside an HLS directory.
func hlsPlaylist(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".m3u8") {
			return filepath.Join(dir, entry.Name())
		}
	}
	return ""
}

func readInfoFile(path string) (infoFile, error) {
	var info infoFile

	file, err := os.Open(path)
	if err != nil {
		return info, fmt.Errorf("error reading info file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Warn().Err(err).Msgf("error closing info file %s", path)
		}
	}()

	data, err := io.ReadAll(io.LimitReader(file, maxInfoFileSize+1))
	if err != nil {
		return info, fmt.Errorf("error reading info file: %w", err)
	}
	if len(data) > maxInfoFileSize {
		return info, fmt.Errorf("info file is larger than %d bytes", maxInfoFileSize)
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return info, fmt.Errorf("error parsing info file: %w", err)
	}
	if len(info.Chapters) > maxImportedEntries || len(info.MutedSegments) > maxImportedEntries {
		return info, fmt.Errorf("info file holds more than %d chapters or muted segments", maxImportedEntries)
	}

	return info, nil
}

// videoMetadata derives type, duration in seconds, views and stream date from the info file,
// whose shape depends on what was archived.
func videoMetadata(info infoFile) (utils.VodType, int, int, time.Time) {
	switch {
	case info.isClip():
		// A clip stores its duration in seconds.
		return utils.Clip, boundedDuration(info.Duration), int(info.ViewCount), info.CreatedAt
	case info.isLive():
		// A live stream has no duration in its info file.
		return utils.Live, 1, int(info.ViewerCount), info.StartedAt
	default:
		// A video stores its duration as a Go duration in nanoseconds.
		videoType := utils.Archive
		switch utils.VodType(info.Type) {
		case utils.Highlight, utils.Upload, utils.Archive:
			videoType = utils.VodType(info.Type)
		}
		return videoType, boundedDuration(int64(time.Duration(info.Duration).Seconds())), int(info.ViewCount), info.CreatedAt
	}
}

// isTwitchURL reports whether the url belongs to twitch, by host rather than by text: a host
// such as twitch.tv.example.com must not pass.
func isTwitchURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "twitch.tv" || strings.HasSuffix(host, ".twitch.tv")
}

// webThumbnail returns the thumbnail the video card needs, which is a required column, falling
// back to the regular thumbnail.
func webThumbnail(files videoFiles) string {
	if files.webThumbnailPath != "" {
		return files.webThumbnailPath
	}
	return files.thumbnailPath
}

// boundedRange keeps a chapter or a muted segment inside the video it belongs to.
func boundedRange(start int, end int, duration int) (int, int, bool) {
	if start < 0 {
		start = 0
	}
	if end > duration {
		end = duration
	}
	if start >= end {
		return 0, 0, false
	}
	return start, end, true
}

// boundedDuration keeps a duration read from a file within what a video can be.
func boundedDuration(seconds int64) int {
	if seconds < 1 {
		return 1
	}
	if seconds > maxVideoDuration {
		return maxVideoDuration
	}
	return int(seconds)
}

// resolveChannel finds the channel of the video, creating it the way the archiver does when
// it is not in the database.
func (s *Service) resolveChannel(ctx context.Context, videosDir string, info infoFile) (*ent.Channel, error) {
	login := info.UserLogin
	if info.isClip() && info.ChannelName != nil {
		login = strings.ToLower(*info.ChannelName)
	}

	if login != "" {
		existing, err := s.Store.Client.Channel.Query().Where(entChannel.Name(login)).Only(ctx)
		if err == nil {
			return existing, nil
		}
		if !ent.IsNotFound(err) {
			return nil, fmt.Errorf("error getting channel: %w", err)
		}
	}

	extID := info.UserID
	if info.isClip() {
		extID = info.ChannelID
	}
	if extID != "" {
		existing, err := s.Store.Client.Channel.Query().Where(entChannel.ExtID(extID)).Only(ctx)
		if err == nil {
			return existing, nil
		}
		if !ent.IsNotFound(err) {
			return nil, fmt.Errorf("error getting channel: %w", err)
		}
	}

	if login == "" && extID == "" {
		return nil, fmt.Errorf("info file names no channel")
	}

	var platformChannel *platform.ChannelInfo
	var err error
	if login != "" {
		platformChannel, err = s.Platform.GetChannel(ctx, &login, nil)
	} else {
		platformChannel, err = s.Platform.GetChannel(ctx, nil, &extID)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting channel from platform: %w", err)
	}

	folderName, err := storagetemplate.GetChannelFolderName(storagetemplate.ChannelTemplateInput{
		ChannelName:        platformChannel.Login,
		ChannelID:          platformChannel.ID,
		ChannelDisplayName: platformChannel.DisplayName,
	})
	if err != nil {
		log.Warn().Err(err).Msg("error resolving channel folder template, falling back to channel login name")
		folderName = platformChannel.Login
	}

	channelDir := filepath.Join(videosDir, folderName)
	if err := utils.CreateDirectory(channelDir); err != nil {
		return nil, fmt.Errorf("error creating channel directory: %w", err)
	}
	imagePath := filepath.Join(channelDir, "profile.png")
	if err := utils.DownloadFile(ctx, platformChannel.ProfileImageURL, imagePath); err != nil {
		log.Warn().Err(err).Msg("error downloading channel profile image")
	}

	created, err := s.Store.Client.Channel.Create().
		SetExtID(platformChannel.ID).
		SetName(platformChannel.Login).
		SetDisplayName(platformChannel.DisplayName).
		SetImagePath(imagePath).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating channel: %w", err)
	}

	return created, nil
}
