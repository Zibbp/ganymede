// Package storage reconciles the videos directory with the database: it finds directories
// that hold the files of a video the database does not know about, and offers to delete or
// import them.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/config"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/storagetemplate"
)

// gracePeriod keeps recently written directories out of the findings. A directory that is
// being written to right now is never a leftover.
const gracePeriod = 15 * time.Minute

// videoFileSuffixes are the endings the archiver gives the files of a video. A directory is
// only a finding when it holds at least one of them, so directories that belong to the
// filesystem or to the operator are never offered for deletion.
var videoFileSuffixes = []string{
	"-info.json",
	"-thumbnail.jpg",
	"-web_thumbnail.jpg",
	"-chat.json",
	"-chat.mp4",
	"-chat-convert.json",
	"-live-chat.json",
}

// captionMarker appears in the name of a caption file, whose extension varies.
const captionMarker = "-caption."

// videoFileExtensions are the endings of the video file itself, whose name is built as
// "<file name>-video.<extension>".
var videoFileExtensions = []string{".mp4", ".mkv", ".ts", ".m3u8", ".mov", ".webm"}

// hlsDirectorySuffix and spriteDirectory are the two directories the archiver writes inside a
// video directory.
const (
	hlsDirectorySuffix = "-video_hls"
	spriteDirectory    = "sprites"
)

// Finding is a directory that holds the files of a video the database does not know about.
type Finding struct {
	Path       string
	SizeBytes  int64
	ModifiedAt time.Time
}

// VideoReference is everything about a video that ties it to a directory on disk.
type VideoReference struct {
	Paths          []string
	FolderName     string
	FileName       string
	ChannelFolders []string
}

// ChannelReference is everything about a channel that ties it to a directory on disk.
type ChannelReference struct {
	Folders []string
}

// References is what the database says about the videos directory.
type References struct {
	VideosDir        string
	Videos           []VideoReference
	Channels         []ChannelReference
	OtherDirectories []string
}

// referencedPaths is the reference set resolved against the disk.
//
// protected holds every directory a video could live in, whether or not it is currently there.
// ancestors holds every directory above one of them, because deleting a directory takes
// everything below it. anchors holds the protected directories that are really on disk, and
// channels the channel directories that are really on disk. Only the directories inside those
// two are ever looked at: being next to a video the database and the disk agree on, or inside
// the directory of a channel that still exists, is what makes it safe to call a directory left
// over. Without that, an empty or out of sync database would turn the whole library into a
// list of findings.
type referencedPaths struct {
	protected map[string]struct{}
	ancestors map[string]struct{}
	anchors   map[string]struct{}
	channels  map[string]struct{}
	fileNames map[string]struct{}
	// unresolved holds the channel directories of videos whose files the disk does not confirm.
	// The database and the disk disagree there, so nothing inside them is reported.
	unresolved map[string]struct{}
	// blocked stops the whole scan when a video cannot be tied to any channel directory.
	blocked bool
}

// VideosDirectory returns the cleaned configured videos directory.
func VideosDirectory() string {
	return filepath.Clean(config.GetEnvConfig().VideosDir)
}

// ReferencesFromDatabase reads the videos and channels and turns them into references.
func ReferencesFromDatabase(ctx context.Context, store *database.Database) (References, error) {
	env := config.GetEnvConfig()
	references := References{
		VideosDir:        filepath.Clean(env.VideosDir),
		OtherDirectories: []string{env.TempDir, env.ConfigDir, env.LogsDir},
	}

	videos, err := store.Client.Vod.Query().WithChannel().All(ctx)
	if err != nil {
		return references, fmt.Errorf("error getting videos: %w", err)
	}

	channels, err := store.Client.Channel.Query().All(ctx)
	if err != nil {
		return references, fmt.Errorf("error getting channels: %w", err)
	}

	channelFolders := make(map[string][]string, len(channels))
	for _, channel := range channels {
		folders := channelFolderNames(channel)
		channelFolders[channel.ID.String()] = folders
		references.Channels = append(references.Channels, ChannelReference{Folders: folders})
	}

	for _, video := range videos {
		reference := VideoReference{
			Paths: []string{
				video.VideoPath,
				video.VideoHlsPath,
				video.ThumbnailPath,
				video.WebThumbnailPath,
				video.ChatPath,
				video.LiveChatPath,
				video.LiveChatConvertPath,
				video.ChatVideoPath,
				video.InfoPath,
				video.CaptionPath,
			},
			FolderName: video.FolderName,
			FileName:   video.FileName,
		}
		reference.Paths = append(reference.Paths, video.SpriteThumbnailsImages...)
		if video.Edges.Channel != nil {
			reference.ChannelFolders = channelFolders[video.Edges.Channel.ID.String()]
		}
		references.Videos = append(references.Videos, reference)
	}

	return references, nil
}

// channelFolderNames returns the directory names a channel can live under: the name the
// template resolves to today and the login name the rest of the code falls back to.
func channelFolderNames(channel *ent.Channel) []string {
	folders := []string{channel.Name}

	folderName, err := storagetemplate.GetChannelFolderName(storagetemplate.ChannelTemplateInput{
		ChannelName:        channel.Name,
		ChannelID:          channel.ExtID,
		ChannelDisplayName: channel.DisplayName,
	})
	if err != nil {
		log.Warn().Err(err).Msgf("error resolving channel folder template for channel %s", channel.Name)
		return folders
	}

	return append(folders, folderName)
}

// Reconcile returns the findings for the supplied references, with sizes. It fails rather than
// return an incomplete result: the caller replaces the stored findings with what it gets back,
// and a short list would delete findings that are still there.
func Reconcile(references References) ([]Finding, error) {
	referenced := resolve(references)

	paths, err := findOrphanedDirectories(references.VideosDir, referenced)
	if err != nil {
		return nil, err
	}

	findings := []Finding{}
	for _, path := range paths {
		finding, err := newFinding(path)
		if err != nil {
			return nil, err
		}
		findings = append(findings, finding)
	}

	return findings, nil
}

// resolve turns the references into the sets the scan works with.
func resolve(references References) referencedPaths {
	videosDir := references.VideosDir
	referenced := referencedPaths{
		protected:  make(map[string]struct{}),
		ancestors:  make(map[string]struct{}),
		anchors:    make(map[string]struct{}),
		channels:   make(map[string]struct{}),
		fileNames:  make(map[string]struct{}),
		unresolved: make(map[string]struct{}),
	}

	for _, dir := range references.OtherDirectories {
		if path, ok := insideVideosDir(dir, videosDir); ok {
			referenced.protected[path] = struct{}{}
			referenced.addAncestors(path)
		}
	}

	for _, channel := range references.Channels {
		for _, folder := range channel.Folders {
			if folder == "" {
				continue
			}
			dir, ok := insideVideosDir(filepath.Join(videosDir, folder), videosDir)
			if !ok {
				continue
			}
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				continue
			}
			// os.Stat follows symlinks, so a channel directory can point anywhere. Scanning it
			// would walk, report and delete outside the videos directory.
			if !staysInsideVideosDir(dir, videosDir) {
				log.Warn().Msgf("channel directory %s leaves the videos directory, skipping it", dir)
				continue
			}
			// The login name and the name the template resolves to can be the same directory on
			// a case insensitive share, and it must not be scanned twice.
			if referenced.hasChannelDirectory(dir, info) {
				continue
			}
			referenced.channels[dir] = struct{}{}
			referenced.addAncestors(dir)
		}
	}

	for _, video := range references.Videos {
		if video.FileName != "" {
			referenced.fileNames[video.FileName] = struct{}{}
		}

		resolved := false

		for _, path := range video.Paths {
			if path == "" {
				continue
			}

			dir, ok := insideVideosDir(filepath.Dir(path), videosDir)
			if !ok {
				continue
			}
			if referenced.add(dir) {
				resolved = true
			}

			// A playlist lives in a subdirectory of the video's own directory.
			if strings.EqualFold(filepath.Ext(path), ".m3u8") {
				if parent, ok := insideVideosDir(filepath.Dir(dir), videosDir); ok {
					referenced.add(parent)
				}
			}
		}

		// The directory the video would live in today, so that a video whose stored paths no
		// longer resolve, for example after an incomplete storage migration, still shields it.
		if video.FolderName != "" {
			for _, channelFolder := range video.ChannelFolders {
				if channelFolder == "" {
					continue
				}
				if dir, ok := insideVideosDir(filepath.Join(videosDir, channelFolder, video.FolderName), videosDir); ok {
					if referenced.add(dir) {
						resolved = true
					}
				}
			}
		}

		if resolved {
			continue
		}

		// Nothing on disk confirms this video. Its paths may simply be stale, after an
		// interrupted path migration or because it predates the folder columns, and then its
		// files are somewhere in its channel directory under a name nothing here knows. Leave
		// that channel alone rather than call its videos leftovers.
		if len(video.ChannelFolders) == 0 {
			referenced.blocked = true
			continue
		}
		for _, channelFolder := range video.ChannelFolders {
			if channelFolder == "" {
				continue
			}
			if dir, ok := insideVideosDir(filepath.Join(videosDir, channelFolder), videosDir); ok {
				referenced.unresolved[dir] = struct{}{}
			}
		}
	}

	return referenced
}

// hasChannelDirectory reports whether the same directory was already recorded under another
// name.
func (r *referencedPaths) hasChannelDirectory(dir string, info os.FileInfo) bool {
	if _, ok := r.channels[dir]; ok {
		return true
	}
	for known := range r.channels {
		if knownInfo, err := os.Stat(known); err == nil && os.SameFile(info, knownInfo) {
			return true
		}
	}
	return false
}

// isUnresolved reports whether the directory is in, or is, a channel directory the database and
// the disk disagree about.
func (r *referencedPaths) isUnresolved(path string) bool {
	for dir := range r.unresolved {
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// add marks a directory as off limits, and as an anchor when it is really on disk. It reports
// whether the directory is there.
func (r *referencedPaths) add(dir string) bool {
	r.protected[dir] = struct{}{}
	r.addAncestors(dir)

	if _, ok := r.anchors[dir]; ok {
		return true
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		r.anchors[dir] = struct{}{}
		return true
	}
	return false
}

func (r *referencedPaths) addAncestors(dir string) {
	for parent := filepath.Dir(dir); parent != "." && parent != string(filepath.Separator); parent = filepath.Dir(parent) {
		if _, ok := r.ancestors[parent]; ok {
			return
		}
		r.ancestors[parent] = struct{}{}
	}
}

// findOrphanedDirectories returns the directories that hold video files but belong to no video
// in the database. Only the directories next to an anchor, or inside a channel directory, are
// considered.
func findOrphanedDirectories(videosDir string, referenced referencedPaths) ([]string, error) {
	if referenced.blocked {
		return nil, errors.New("a video in the database cannot be tied to a channel directory, refusing to scan")
	}

	regions := make([]string, 0, len(referenced.anchors)+len(referenced.channels))
	seen := make(map[string]struct{}, len(referenced.anchors)+len(referenced.channels))

	addRegion := func(region string) {
		// The videos directory itself is never a region: its entries are channel directories,
		// filesystem directories such as lost+found, and whatever else the operator keeps there.
		if region == videosDir {
			return
		}
		// A protected directory is never a region either. A video anchors its own directory
		// through its sprites and its HLS segments, and nothing inside a video is a leftover.
		if _, ok := referenced.protected[region]; ok {
			return
		}
		// The database and the disk disagree somewhere in this channel.
		if referenced.isUnresolved(region) {
			return
		}
		if _, ok := seen[region]; ok {
			return
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}

	for anchor := range referenced.anchors {
		addRegion(filepath.Dir(anchor))
	}
	// A channel that still exists anchors its own directory, so the leftovers of a channel
	// whose videos were all deleted are found as well. This needs at least one video that the
	// database and the disk agree on, otherwise a database restored empty next to a full
	// library would report every video in it.
	if len(referenced.anchors) > 0 {
		for channel := range referenced.channels {
			addRegion(channel)
		}
	}

	sort.Strings(regions)

	orphans := []string{}

	for _, region := range regions {
		entries, err := os.ReadDir(region)
		if err != nil {
			return nil, fmt.Errorf("error reading directory %s: %w", region, err)
		}

		for _, entry := range entries {
			// Files and symlinks are never reported or deleted.
			if !entry.IsDir() {
				continue
			}

			path := filepath.Join(region, entry.Name())

			if _, ok := referenced.protected[path]; ok {
				continue
			}
			// Something referenced lives below this directory, and deleting it would take that
			// with it.
			if _, ok := referenced.ancestors[path]; ok {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}

			if !utf8.ValidString(path) {
				log.Warn().Msgf("skipping directory with an invalid name: %q", path)
				continue
			}

			orphan, err := isOrphanedDirectory(path, referenced.fileNames)
			if err != nil {
				return nil, fmt.Errorf("error reading directory %s: %w", path, err)
			}
			if orphan {
				orphans = append(orphans, path)
			}
		}
	}

	return orphans, nil
}

// isOrphanedDirectory applies the checks that look at the directory itself: it holds video
// files, none of them belong to a known video, and nothing in it was written recently.
func isOrphanedDirectory(path string, fileNames map[string]struct{}) (bool, error) {
	holdsVideo, err := holdsVideoFiles(path)
	if err != nil || !holdsVideo {
		return false, err
	}

	known, err := holdsKnownVideoFiles(path, fileNames)
	if err != nil || known {
		return false, err
	}

	recent, err := writtenRecently(path)
	if err != nil || recent {
		return false, err
	}

	return true, nil
}

// holdsVideoFiles reports whether the directory holds files the archiver would have written.
// Directory names are not considered, apart from the two directories the archiver writes: a
// filesystem cache such as @eaDir names its entries after the media files and would otherwise
// look like a video directory.
func holdsVideoFiles(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			if strings.HasSuffix(name, hlsDirectorySuffix) || name == spriteDirectory {
				return true, nil
			}
			continue
		}

		if strings.Contains(name, captionMarker) {
			return true, nil
		}
		for _, suffix := range videoFileSuffixes {
			if strings.HasSuffix(name, suffix) {
				return true, nil
			}
		}
		for _, extension := range videoFileExtensions {
			if strings.HasSuffix(name, "-video"+extension) {
				return true, nil
			}
		}
	}

	return false, nil
}

// holdsKnownVideoFiles reports whether the directory holds files that belong to a video in the
// database. The files of a video keep their names when they move, so a video whose directory
// the database no longer knows about, after a storage migration that did not finish or a
// directory an operator renamed, is recognised by them.
func holdsKnownVideoFiles(path string, fileNames map[string]struct{}) (bool, error) {
	if len(fileNames) == 0 {
		return false, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		name := entry.Name()
		for fileName := range fileNames {
			if strings.HasPrefix(name, fileName+"-") {
				return true, nil
			}
		}
	}

	return false, nil
}

// writtenRecently reports whether anything in the directory was written within the grace
// period. A timestamp in the future is not treated as recent, so a directory with a broken
// modification time cannot hide forever.
func writtenRecently(path string) (bool, error) {
	recent := false

	err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		age := time.Since(info.ModTime())
		if age >= 0 && age < gracePeriod {
			recent = true
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	return recent, nil
}

// staysInsideVideosDir reports whether the directory is still inside the videos directory once
// every symlink in both paths is resolved.
func staysInsideVideosDir(dir string, videosDir string) bool {
	resolvedRoot, err := filepath.EvalSymlinks(videosDir)
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return false
	}
	_, ok := insideVideosDir(resolved, resolvedRoot)
	return ok
}

// insideVideosDir returns the cleaned directory if it is inside the videos directory. Anything
// outside can never collide with the scan.
func insideVideosDir(dir string, videosDir string) (string, bool) {
	if dir == "" || dir == "." {
		return "", false
	}

	dir = filepath.Clean(dir)

	rel, err := filepath.Rel(videosDir, dir)
	if err != nil || rel == "." || isOutside(rel) {
		return "", false
	}

	return dir, true
}

// isOutside reports whether a relative path leaves the directory it was resolved against. A
// directory may legitimately be named "..something", so only a bare ".." and a ".." segment
// count.
func isOutside(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func newFinding(path string) (Finding, error) {
	var finding Finding

	info, err := os.Stat(path)
	if err != nil {
		return finding, fmt.Errorf("error reading directory %s: %w", path, err)
	}

	size, err := directorySize(path)
	if err != nil {
		return finding, fmt.Errorf("error getting size of directory %s: %w", path, err)
	}

	finding.Path = path
	finding.SizeBytes = size
	finding.ModifiedAt = info.ModTime()

	return finding, nil
}

func directorySize(path string) (int64, error) {
	var size int64

	err := filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}
