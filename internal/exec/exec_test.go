package exec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	osExec "os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/zibbp/ganymede/ent"
)

func TestStartArchiveCommand(t *testing.T) {
	t.Parallel()

	cmd := osExec.Command("sh", "-c", "exit 0")
	cmd.SysProcAttr = vodArchiveProcessAttributes()

	done, err := startArchiveCommand(cmd)
	if err != nil {
		t.Fatalf("start archive command: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wait for archive command: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for archive command")
	}
}

// TestStartArchiveCommandWaitsForEscalationCleanup mirrors the cancellation
// path used by DownloadTwitchLiveVideo (SIGTERM to the process group) and
// verifies that completion is not reported while the delayed SIGKILL helper is
// still running, while a descendant that ignores SIGTERM is still cleaned up.
func TestStartArchiveCommandWaitsForEscalationCleanup(t *testing.T) {
	tempDir := t.TempDir()
	ffmpegPIDPath := filepath.Join(tempDir, "ffmpeg.pid")
	descendantPIDPath := filepath.Join(tempDir, "ffmpeg-descendant.pid")
	descendantPath := filepath.Join(tempDir, "ffmpeg-descendant")

	writeExecutable(t, filepath.Join(tempDir, "ffmpeg"), `#!/bin/sh
printf '%s' "$$" > "$1"
trap 'exit 0' TERM
"$2" "$3" &
wait
`)
	writeExecutable(t, descendantPath, `#!/bin/sh
printf '%s' "$$" > "$1"
trap '' TERM
while :; do
	sleep 1
done
`)

	cmd := osExec.Command(filepath.Join(tempDir, "ffmpeg"), ffmpegPIDPath, descendantPath, descendantPIDPath)
	cmd.SysProcAttr = liveArchiveProcessAttributes()

	done, err := startArchiveCommand(cmd)
	if err != nil {
		t.Fatalf("start archive command: %v", err)
	}

	ffmpegPID := waitForPIDFile(t, ffmpegPIDPath)
	descendantPID := waitForPIDFile(t, descendantPIDPath)
	pgid := cmd.Process.Pid
	t.Cleanup(func() {
		killTestProcess(t, -pgid, "live archive process group")
		killTestProcess(t, ffmpegPID, "ffmpeg")
		killTestProcess(t, descendantPID, "ffmpeg descendant")
	})

	if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
		t.Fatalf("send SIGTERM to archive process group: %v", err)
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for archive command to finish after cancellation")
	}

	// The helper sleeps longer than this grace period, so a surviving helper
	// (the bug this guards against) is reliably detected here.
	waitForProcessGroupExit(t, pgid, time.Second)
	waitForProcessExit(t, "ffmpeg", ffmpegPID)
	waitForProcessExit(t, "ffmpeg descendant", descendantPID)
}

func TestLiveArchiveProcessAttributes(t *testing.T) {
	t.Parallel()

	attrs := liveArchiveProcessAttributes()
	if !attrs.Setpgid {
		t.Fatal("live archive process must run in its own process group")
	}
	if attrs.Pdeathsig != syscall.SIGTERM {
		t.Fatalf("parent death signal = %v, want SIGTERM", attrs.Pdeathsig)
	}
}

func TestVodArchiveProcessAttributes(t *testing.T) {
	t.Parallel()

	attrs := vodArchiveProcessAttributes()
	if !attrs.Setpgid {
		t.Fatal("VOD archive process must run in its own process group")
	}
	if attrs.Pdeathsig != syscall.SIGTERM {
		t.Fatalf("parent death signal = %v, want SIGTERM", attrs.Pdeathsig)
	}
}

func TestTwitchVideoDownloadArgsPreferFFmpegForHLS(t *testing.T) {
	t.Parallel()

	args := twitchVideoDownloadArgs(
		"best[height=1080]/best",
		"https://twitch.tv/videos/2838897713",
		"/tmp/video.%(ext)s",
		"--fragment-retries,20",
	)

	ffmpegPreference := -1
	customArgs := -1
	for i, arg := range args {
		switch arg {
		case "--hls-prefer-ffmpeg":
			ffmpegPreference = i
		case "--fragment-retries":
			customArgs = i
		}
	}

	if ffmpegPreference == -1 {
		t.Fatalf("Twitch VOD arguments do not prefer the ffmpeg HLS downloader: %v", args)
	}
	if customArgs == -1 || customArgs < ffmpegPreference {
		t.Fatalf("configured yt-dlp arguments must follow defaults: %v", args)
	}
}

func TestAppendYtDlpVideoConfigArgsExcludesOutputOptions(t *testing.T) {
	t.Parallel()

	initial := []string{"-o", "/data/temp/video.%(ext)s"}
	configArgs := strings.Join([]string{
		"  --retries  ", "  10 ", "  ",
		"-o", "/tmp/short-separated",
		"--output", "/tmp/long-separated",
		"-o=/tmp/short-equals",
		"-o/tmp/short-attached",
		"--output=/tmp/long-equals",
		"-P", "/tmp/path-short-separated",
		"--paths", "/tmp/path-long-separated",
		"-P=/tmp/path-short-equals",
		"-P/tmp/path-short-attached",
		"--paths=/tmp/path-long-equals",
		" --fragment-retries", "20 ",
	}, ",")

	got := appendYtDlpVideoConfigArgs(initial, configArgs)
	want := []string{
		"-o", "/data/temp/video.%(ext)s",
		"--retries", "10",
		"--fragment-retries", "20",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("appendYtDlpVideoConfigArgs() = %v, want %v", got, want)
	}
}

func TestPostProcessVideoFFmpegArgsIncludesTitleMetadata(t *testing.T) {
	t.Parallel()

	video := ent.Vod{
		Title:                "A Twitch stream title with spaces",
		TmpVideoDownloadPath: "/tmp/input.ts",
		TmpVideoConvertPath:  "/tmp/output.mp4",
	}

	args := postProcessVideoFFmpegArgs(video, "-c:v copy -c:a copy", true)

	titleMetadataIndex := -1
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-metadata" && args[i+1] == "title="+video.Title {
			titleMetadataIndex = i
			break
		}
	}

	if titleMetadataIndex == -1 {
		t.Fatalf("FFmpeg arguments do not contain title metadata: %v", args)
	}

	// ffmpeg writes to a name of its own; the playlist is only called what the
	// rest of the pipeline looks for once the conversion has been checked. The
	// name is spelled out rather than derived, so that removing the indirection
	// fails here. The video file is named by the segment option next to it.
	wantPlaylist := "/tmp/output.m3u8.part"
	if args[len(args)-1] != wantPlaylist {
		t.Fatalf("last FFmpeg argument = %q, want playlist %q", args[len(args)-1], wantPlaylist)
	}

	segmentFile := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-hls_segment_filename" {
			segmentFile = args[i+1]
		}
	}
	if segmentFile != video.TmpVideoConvertPath {
		t.Fatalf("segment file = %q, want video path %q", segmentFile, video.TmpVideoConvertPath)
	}
}

func TestPlaylistPathForVideo(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"/data/videos/channel/2026-01-01/123-video.mp4": "/data/videos/channel/2026-01-01/123-video.m3u8",
		"/tmp/uuid_uuid-video-convert.mp4":              "/tmp/uuid_uuid-video-convert.m3u8",
		// A directory with a dot in its name must not be mistaken for the
		// extension of an otherwise extensionless file.
		"/data/videos/channel.v2/123-video": "/data/videos/channel.v2/123-video.m3u8",
	}

	for videoPath, want := range tests {
		if got := PlaylistPathForVideo(videoPath); got != want {
			t.Fatalf("PlaylistPathForVideo(%q) = %q, want %q", videoPath, got, want)
		}
	}
}

func TestPostProcessVideoWritesPlaylistNextToVideo(t *testing.T) {
	tmpDir := t.TempDir()
	video := ent.Vod{
		Title:                "playlist next to video",
		TmpVideoDownloadPath: createDummyVideo(t, tmpDir),
		TmpVideoConvertPath:  filepath.Join(tmpDir, "finalized.mp4"),
	}

	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", true, io.Discard); err != nil {
		t.Fatalf("post-process video: %v", err)
	}

	// Every segment addresses a range of the video file, so the player learns
	// the layout from the playlist instead of walking the file.
	playlist, err := os.ReadFile(PlaylistPathForVideo(video.TmpVideoConvertPath))
	if err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	for _, want := range []string{"#EXT-X-MAP:", "#EXT-X-BYTERANGE:", filepath.Base(video.TmpVideoConvertPath)} {
		if !bytes.Contains(playlist, []byte(want)) {
			t.Fatalf("playlist does not contain %q:\n%s", want, playlist)
		}
	}

	// The video file has to stay usable on its own, for downloads and for
	// players that never see the playlist.
	probe, err := ProbeMediaDuration(t.Context(), video.TmpVideoConvertPath)
	if err != nil {
		t.Fatalf("probe finalized video: %v", err)
	}
	if probe.Duration <= 0 {
		t.Fatalf("finalized video reports no duration: %+v", probe)
	}
	decode := osExec.Command("ffmpeg", "-v", "error", "-i", video.TmpVideoConvertPath, "-f", "null", "-")
	if out, err := decode.CombinedOutput(); err != nil {
		t.Fatalf("decode finalized video: %v, output: %s", err, out)
	}
}

func TestNormalizedPostProcessVideoFFmpegArgsResetEachStream(t *testing.T) {
	t.Parallel()

	video := ent.Vod{
		TmpVideoDownloadPath: "/tmp/input.ts",
		TmpVideoConvertPath:  "/tmp/output.mp4",
	}
	args := normalizedPostProcessVideoFFmpegArgs(video, "-c:v copy -c:a copy", true)

	// The audio filter has to survive the repair pass. The fragmented muxer
	// rejects the ADTS AAC a live archive downloads and, unlike the MP4 muxer,
	// does not insert the filter itself, so dropping it fails the conversion
	// outright.
	want := map[string]string{
		"-bsf:v": "setts=ts=TS-STARTDTS",
		"-bsf:a": "aac_adtstoasc,setts=ts=TS-STARTDTS",
	}
	got := map[string]string{}
	for i := 0; i < len(args)-1; i++ {
		if _, ok := want[args[i]]; ok {
			got[args[i]] = args[i+1]
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized bitstream filters = %v, want %v: %v", got, want, args)
	}
}

func TestPostProcessVideoFFmpegArgsKeepTheAudioBitstreamFilter(t *testing.T) {
	t.Parallel()

	video := ent.Vod{
		TmpVideoDownloadPath: "/tmp/input.ts",
		TmpVideoConvertPath:  "/tmp/output.mp4",
	}

	for _, args := range [][]string{
		postProcessVideoFFmpegArgs(video, "-c:v copy -c:a copy", true),
		postProcessVideoFFmpegArgs(video, "-c:v copy -c:a copy", false),
	} {
		found := false
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-bsf:a" && args[i+1] == "aac_adtstoasc" {
				found = true
			}
		}
		if !found {
			t.Fatalf("FFmpeg arguments do not convert ADTS audio: %v", args)
		}
	}
}

func TestPostProcessVideoPointsPlaylistAtArchivedName(t *testing.T) {
	tmpDir := t.TempDir()
	video := ent.Vod{
		Title:                "playlist names the archived file",
		TmpVideoDownloadPath: createDummyVideo(t, tmpDir),
		// The names deliberately differ the way they do in an archive: ffmpeg
		// writes the temporary name into the playlist, but the player only ever
		// sees the file under its final name.
		TmpVideoConvertPath: filepath.Join(tmpDir, "1234_uuid-video-convert.mp4"),
		VideoPath:           filepath.Join(tmpDir, "archived", "channel_2026-01-01-video.mp4"),
	}

	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", true, io.Discard); err != nil {
		t.Fatalf("post-process video: %v", err)
	}

	playlist, err := os.ReadFile(PlaylistPathForVideo(video.TmpVideoConvertPath))
	if err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	if bytes.Contains(playlist, []byte(filepath.Base(video.TmpVideoConvertPath))) {
		t.Fatalf("playlist still references the temporary file:\n%s", playlist)
	}
	if !bytes.Contains(playlist, []byte(filepath.Base(video.VideoPath))) {
		t.Fatalf("playlist does not reference the archived file:\n%s", playlist)
	}

	// Renaming the video next to the playlist has to be enough to play it.
	archivedVideo := video.VideoPath
	if err := os.MkdirAll(filepath.Dir(archivedVideo), 0o755); err != nil {
		t.Fatalf("create archive directory: %v", err)
	}
	if err := os.Rename(video.TmpVideoConvertPath, archivedVideo); err != nil {
		t.Fatalf("move video: %v", err)
	}
	if err := os.Rename(PlaylistPathForVideo(video.TmpVideoConvertPath), PlaylistPathForVideo(archivedVideo)); err != nil {
		t.Fatalf("move playlist: %v", err)
	}
	decode := osExec.Command("ffmpeg", "-v", "error", "-i", PlaylistPathForVideo(archivedVideo), "-t", "1", "-f", "null", "-")
	if out, err := decode.CombinedOutput(); err != nil {
		t.Fatalf("play archived playlist: %v, output: %s", err, out)
	}
}

func TestPostProcessVideoFailsWhenNoPlaylistWasWritten(t *testing.T) {
	tmpDir := t.TempDir()
	video := ent.Vod{
		TmpVideoDownloadPath: createDummyVideo(t, tmpDir),
		TmpVideoConvertPath:  filepath.Join(tmpDir, "finalized.mp4"),
	}

	// Configured arguments that switch the muxer make ffmpeg exit successfully
	// without ever writing the pair. A truncated fragmented file still probes as
	// valid media, so the missing playlist is what has to be noticed.
	err := postProcessVideo(t.Context(), video, "-f mp4", true, io.Discard)
	if err == nil {
		t.Fatal("post-processing reported success without producing a playlist")
	}
	if !strings.Contains(err.Error(), "playlist") {
		t.Fatalf("error does not mention the missing playlist: %v", err)
	}
}

func TestPostProcessVideoDiscardsTheRemainsOfAFailedRun(t *testing.T) {
	tmpDir := t.TempDir()
	video := ent.Vod{
		TmpVideoDownloadPath: createDummyVideo(t, tmpDir),
		TmpVideoConvertPath:  filepath.Join(tmpDir, "finalized.mp4"),
	}

	// A first run leaves a finished pair behind.
	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", true, io.Discard); err != nil {
		t.Fatalf("post-process video: %v", err)
	}

	// ffmpeg can fail part way through and still close its output, leaving a
	// truncated video beside a playlist that looks complete. Nothing may be
	// left that a retry could mistake for a finished conversion.
	if err := postProcessVideo(t.Context(), video, "-bsf:v nonexistent_filter", true, io.Discard); err == nil {
		t.Fatal("post-processing reported success with a broken configuration")
	}

	for _, path := range []string{video.TmpVideoConvertPath, PlaylistPathForVideo(video.TmpVideoConvertPath)} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s survived a failed conversion: %v", path, err)
		}
	}
}

func TestValidatePlaylistRejectsAnUnfinishedPlaylist(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	tests := map[string]string{
		// ffmpeg closes a playlist even when it was interrupted, so the closing
		// tag is what tells the two apart.
		"unfinished":     "#EXTM3U\n#EXT-X-VERSION:7\n#EXTINF:10.0,\n",
		"not a playlist": "\x00\x00\x00 ftypiso5",
		"empty":          "",
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			playlistPath := filepath.Join(tmpDir, name+".m3u8")
			if err := os.WriteFile(playlistPath, []byte(contents), 0o644); err != nil {
				t.Fatalf("write playlist: %v", err)
			}
			if err := ValidatePlaylist(playlistPath); err == nil {
				t.Fatalf("%q was accepted as a complete playlist", contents)
			}
		})
	}

	complete := filepath.Join(tmpDir, "complete.m3u8")
	if err := os.WriteFile(complete, []byte("#EXTM3U\n#EXTINF:10.0,\nvideo.mp4\n#EXT-X-ENDLIST\n"), 0o644); err != nil {
		t.Fatalf("write playlist: %v", err)
	}
	if err := ValidatePlaylist(complete); err != nil {
		t.Fatalf("complete playlist rejected: %v", err)
	}
}

func TestRewritePlaylistMediaNameOnlyTouchesReferences(t *testing.T) {
	t.Parallel()

	playlistPath := filepath.Join(t.TempDir(), "video.m3u8")
	before := `#EXTM3U
#EXT-X-MAP:URI="old-video.mp4",BYTERANGE="1402@0"
#EXTINF:10.0,
#EXT-X-BYTERANGE:3359464@1402
old-video.mp4
#EXT-X-ENDLIST
`
	if err := os.WriteFile(playlistPath, []byte(before), 0o644); err != nil {
		t.Fatalf("write playlist: %v", err)
	}

	if err := RewritePlaylistMediaName(playlistPath, "old-video.mp4", "new-video.mp4"); err != nil {
		t.Fatalf("rewrite playlist: %v", err)
	}

	after, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	if bytes.Contains(after, []byte("old-video.mp4")) {
		t.Fatalf("playlist still references the old name:\n%s", after)
	}
	if bytes.Count(after, []byte("new-video.mp4")) != 2 {
		t.Fatalf("playlist does not name the video twice:\n%s", after)
	}
	if !bytes.Contains(after, []byte(`BYTERANGE="1402@0"`)) || !bytes.Contains(after, []byte("#EXT-X-ENDLIST")) {
		t.Fatalf("rewrite changed more than the references:\n%s", after)
	}

	// A playlist that names something else does not belong to this video, and
	// quietly writing the new name into it would produce one that parses while
	// describing nothing.
	if err := RewritePlaylistMediaName(playlistPath, "old-video.mp4", "other-video.mp4"); err == nil {
		t.Fatal("a playlist naming a different video was rewritten")
	}
	unchanged, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("read playlist: %v", err)
	}
	if !bytes.Equal(after, unchanged) {
		t.Fatal("the rejected rewrite changed the playlist")
	}

	// The name reaches a file browsers parse as a playlist and comes from a
	// field that can be edited through the API.
	for _, name := range []string{"a\nb.mp4", "a\"b.mp4", "a#b.mp4"} {
		if err := RewritePlaylistMediaName(playlistPath, "new-video.mp4", name); err == nil {
			t.Fatalf("%q was written into a playlist", name)
		}
	}
}

func TestPostProcessVideoWithoutPlaylistWritesAProgressiveVideo(t *testing.T) {
	tmpDir := t.TempDir()
	video := ent.Vod{
		Title:                "playlists turned off",
		TmpVideoDownloadPath: createDummyVideo(t, tmpDir),
		TmpVideoConvertPath:  filepath.Join(tmpDir, "finalized.mp4"),
		VideoPath:            filepath.Join(tmpDir, "archived-video.mp4"),
	}

	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", false, io.Discard); err != nil {
		t.Fatalf("post-process video: %v", err)
	}

	// Installations that do not want the extra file keep exactly what they had
	// before: one progressive video and nothing beside it.
	contents, err := os.ReadFile(video.TmpVideoConvertPath)
	if err != nil {
		t.Fatalf("read post-processed video: %v", err)
	}
	if bytes.Contains(contents, []byte("moof")) {
		t.Fatal("video is fragmented although playlists are turned off")
	}
	for _, path := range []string{PlaylistPathForVideo(video.TmpVideoConvertPath), PendingPlaylistPath(video.TmpVideoConvertPath)} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s was written although playlists are turned off: %v", path, err)
		}
	}

	probe, err := ProbeMediaDuration(t.Context(), video.TmpVideoConvertPath)
	if err != nil {
		t.Fatalf("probe post-processed video: %v", err)
	}
	if probe.Duration <= 0 {
		t.Fatalf("post-processed video reports no duration: %+v", probe)
	}
}

func TestPostProcessVideoRemovesAPlaylistFromAnEarlierRun(t *testing.T) {
	tmpDir := t.TempDir()
	video := ent.Vod{
		TmpVideoDownloadPath: createDummyVideo(t, tmpDir),
		TmpVideoConvertPath:  filepath.Join(tmpDir, "finalized.mp4"),
	}

	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", true, io.Discard); err != nil {
		t.Fatalf("post-process video with a playlist: %v", err)
	}

	// Turning playlists off and converting again must not leave the old
	// playlist behind: it describes a layout the new video does not have, and a
	// playlist that parses is indistinguishable from a correct one, so no
	// player could fall back.
	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", false, io.Discard); err != nil {
		t.Fatalf("post-process video without a playlist: %v", err)
	}

	for _, path := range []string{PlaylistPathForVideo(video.TmpVideoConvertPath), PendingPlaylistPath(video.TmpVideoConvertPath)} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s survived a conversion without playlists: %v", path, err)
		}
	}
}

func TestValidatePlaylistCoversVideo(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	videoPath := filepath.Join(tmpDir, "video.mp4")
	if err := os.WriteFile(videoPath, make([]byte, 5000), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	write := func(t *testing.T, name, contents string) string {
		t.Helper()
		playlistPath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(playlistPath, []byte(contents), 0o644); err != nil {
			t.Fatalf("write playlist: %v", err)
		}
		return playlistPath
	}

	matching := write(t, "matching.m3u8", `#EXTM3U
#EXT-X-MAP:URI="video.mp4",BYTERANGE="1000@0"
#EXT-X-BYTERANGE:2000@1000
video.mp4
#EXT-X-BYTERANGE:2000@3000
video.mp4
#EXT-X-ENDLIST
`)
	if err := ValidatePlaylistCoversVideo(matching, videoPath); err != nil {
		t.Fatalf("matching playlist rejected: %v", err)
	}

	// A video that was rewritten, truncated or replaced after the playlist was
	// written leaves the offsets addressing something else.
	tests := map[string]string{
		"short": `#EXTM3U
#EXT-X-MAP:URI="video.mp4",BYTERANGE="1000@0"
#EXT-X-BYTERANGE:2000@1000
video.mp4
#EXT-X-ENDLIST
`,
		"long": `#EXTM3U
#EXT-X-MAP:URI="video.mp4",BYTERANGE="1000@0"
#EXT-X-BYTERANGE:9000@1000
video.mp4
#EXT-X-ENDLIST
`,
		"unreadable": `#EXTM3U
#EXT-X-BYTERANGE:notanumber@0
video.mp4
#EXT-X-ENDLIST
`,
	}
	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := ValidatePlaylistCoversVideo(write(t, name+".m3u8", contents), videoPath); err == nil {
				t.Fatal("a playlist that does not match the video was accepted")
			}
		})
	}
}

func TestValidatePlaylistNamesVideo(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	playlistPath := filepath.Join(tmpDir, "video.m3u8")
	if err := os.WriteFile(playlistPath, []byte(`#EXTM3U
#EXT-X-MAP:URI="channel-video.mp4",BYTERANGE="1000@0"
#EXT-X-BYTERANGE:2000@1000
channel-video.mp4
#EXT-X-ENDLIST
`), 0o644); err != nil {
		t.Fatalf("write playlist: %v", err)
	}

	if err := ValidatePlaylistNamesVideo(playlistPath, filepath.Join(tmpDir, "channel-video.mp4")); err != nil {
		t.Fatalf("playlist naming the video rejected: %v", err)
	}

	// A rename elsewhere in the pipeline moves the pair together, so the byte
	// ranges still fit while the name no longer does.
	if err := ValidatePlaylistNamesVideo(playlistPath, filepath.Join(tmpDir, "renamed-video.mp4")); err == nil {
		t.Fatal("playlist naming another video was accepted")
	}
}

func TestPostProcessVideoNormalizesAnomalousContainerTimeline(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTimestampOffsetMedia(t, tmpDir, "mp4")
	outputPath := filepath.Join(tmpDir, "normalized.mp4")
	video := ent.Vod{
		Title:                "timestamp normalization regression",
		TmpVideoDownloadPath: inputPath,
		TmpVideoConvertPath:  outputPath,
		// Set so the repair pass also exercises pointing the playlist at the
		// archived name, which is where the two steps meet.
		VideoPath: filepath.Join(tmpDir, "archived-video.mp4"),
	}

	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", true, io.Discard); err != nil {
		t.Fatalf("post-process timestamp-offset media: %v", err)
	}

	probe, err := ProbeMediaDuration(t.Context(), outputPath)
	if err != nil {
		t.Fatalf("probe normalized output: %v", err)
	}
	if probe.HasTimestampAnomaly() {
		t.Fatalf("output still has a timestamp anomaly: format=%f stream=%f", probe.FormatDuration, probe.LongestStreamDuration)
	}
	if probe.FormatDuration < 2 || probe.FormatDuration > 4 {
		t.Fatalf("normalized format duration = %f, want about 3 seconds", probe.FormatDuration)
	}
	decode := osExec.Command("ffmpeg", "-v", "error", "-i", outputPath, "-f", "null", "-")
	if out, err := decode.CombinedOutput(); err != nil {
		t.Fatalf("decode normalized output: %v, output: %s", err, out)
	}
}

func TestVodArchiveProcessGroupExitsAfterWorkerHardCrash(t *testing.T) {
	tempDir := t.TempDir()
	ytDlpPIDPath := filepath.Join(tempDir, "yt-dlp.pid")
	ffmpegPIDPath := filepath.Join(tempDir, "ffmpeg.pid")

	writeExecutable(t, filepath.Join(tempDir, "yt-dlp"), `#!/bin/sh
printf '%s' "$$" > "$1"
ffmpeg "$2" &
wait
`)
	writeExecutable(t, filepath.Join(tempDir, "ffmpeg"), `#!/bin/sh
printf '%s' "$$" > "$1"
while :; do
	sleep 1
done
`)

	worker := osExec.Command(os.Args[0], "-test.run=^TestVodArchiveWorkerHelper$")
	worker.Env = append(os.Environ(),
		"GANYMEDE_ARCHIVE_WORKER_HELPER=1",
		"GANYMEDE_ARCHIVE_TEST_DIR="+tempDir,
		"PATH="+tempDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	if err := worker.Start(); err != nil {
		t.Fatalf("start worker helper: %v", err)
	}
	t.Cleanup(func() {
		if worker.Process != nil {
			_ = worker.Process.Kill()
		}
		_ = worker.Wait()
	})

	ytDlpPID := waitForPIDFile(t, ytDlpPIDPath)
	ffmpegPID := waitForPIDFile(t, ffmpegPIDPath)
	processGroupID, err := syscall.Getpgid(ytDlpPID)
	if err != nil {
		t.Fatalf("get archive process group: %v", err)
	}
	t.Cleanup(func() {
		killTestProcess(t, -processGroupID, "archive process group")
		killTestProcess(t, ytDlpPID, "yt-dlp")
		killTestProcess(t, ffmpegPID, "ffmpeg")
	})

	if err := worker.Process.Kill(); err != nil {
		t.Fatalf("hard-crash worker helper: %v", err)
	}
	if err := worker.Wait(); err == nil {
		t.Fatal("hard-crashed worker helper exited successfully")
	}

	waitForProcessExit(t, "yt-dlp", ytDlpPID)
	waitForProcessExit(t, "ffmpeg", ffmpegPID)
}

func TestLiveArchiveProcessGroupExitsAfterWorkerHardCrash(t *testing.T) {
	tempDir := t.TempDir()
	ffmpegPIDPath := filepath.Join(tempDir, "ffmpeg.pid")
	descendantPIDPath := filepath.Join(tempDir, "ffmpeg-descendant.pid")

	writeExecutable(t, filepath.Join(tempDir, "ffmpeg"), `#!/bin/sh
printf '%s' "$$" > "$1"
ffmpeg-descendant "$2" &
wait
`)
	writeExecutable(t, filepath.Join(tempDir, "ffmpeg-descendant"), `#!/bin/sh
printf '%s' "$$" > "$1"
while :; do
	sleep 1
done
`)

	worker := osExec.Command(os.Args[0], "-test.run=^TestLiveArchiveWorkerHelper$")
	worker.Env = append(os.Environ(),
		"GANYMEDE_LIVE_ARCHIVE_WORKER_HELPER=1",
		"GANYMEDE_ARCHIVE_TEST_DIR="+tempDir,
		"PATH="+tempDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	if err := worker.Start(); err != nil {
		t.Fatalf("start live worker helper: %v", err)
	}
	t.Cleanup(func() {
		if worker.Process != nil {
			_ = worker.Process.Kill()
		}
		_ = worker.Wait()
	})

	ffmpegPID := waitForPIDFile(t, ffmpegPIDPath)
	descendantPID := waitForPIDFile(t, descendantPIDPath)
	processGroupID, err := syscall.Getpgid(ffmpegPID)
	if err != nil {
		t.Fatalf("get live archive process group: %v", err)
	}
	t.Cleanup(func() {
		killTestProcess(t, -processGroupID, "live archive process group")
		killTestProcess(t, ffmpegPID, "ffmpeg")
		killTestProcess(t, descendantPID, "ffmpeg descendant")
	})

	if err := worker.Process.Kill(); err != nil {
		t.Fatalf("hard-crash live worker helper: %v", err)
	}
	if err := worker.Wait(); err == nil {
		t.Fatal("hard-crashed live worker helper exited successfully")
	}

	waitForProcessExit(t, "ffmpeg", ffmpegPID)
	waitForProcessExit(t, "ffmpeg descendant", descendantPID)
}

// TestLiveArchiveProcessGroupEscalatesForTermIgnoringFFmpeg covers the crash
// recovery path where ffmpeg ignores the first SIGTERM while blocked on a
// network read. The forwarding shim must escalate to SIGKILL so the capture
// process group does not outlive a crashed worker.
func TestLiveArchiveProcessGroupEscalatesForTermIgnoringFFmpeg(t *testing.T) {
	tempDir := t.TempDir()
	ffmpegPIDPath := filepath.Join(tempDir, "ffmpeg.pid")
	descendantPIDPath := filepath.Join(tempDir, "ffmpeg-descendant.pid")

	writeExecutable(t, filepath.Join(tempDir, "ffmpeg"), `#!/bin/sh
printf '%s' "$$" > "$1"
trap '' TERM
ffmpeg-descendant "$2" &
wait
`)
	writeExecutable(t, filepath.Join(tempDir, "ffmpeg-descendant"), `#!/bin/sh
printf '%s' "$$" > "$1"
trap '' TERM
while :; do
	sleep 1
done
`)

	worker := osExec.Command(os.Args[0], "-test.run=^TestLiveArchiveWorkerHelper$")
	worker.Env = append(os.Environ(),
		"GANYMEDE_LIVE_ARCHIVE_WORKER_HELPER=1",
		"GANYMEDE_ARCHIVE_TEST_DIR="+tempDir,
		"PATH="+tempDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	if err := worker.Start(); err != nil {
		t.Fatalf("start live worker helper: %v", err)
	}
	t.Cleanup(func() {
		if worker.Process != nil {
			_ = worker.Process.Kill()
		}
		_ = worker.Wait()
	})

	ffmpegPID := waitForPIDFile(t, ffmpegPIDPath)
	descendantPID := waitForPIDFile(t, descendantPIDPath)
	processGroupID, err := syscall.Getpgid(ffmpegPID)
	if err != nil {
		t.Fatalf("get live archive process group: %v", err)
	}
	t.Cleanup(func() {
		killTestProcess(t, -processGroupID, "live archive process group")
		killTestProcess(t, ffmpegPID, "ffmpeg")
		killTestProcess(t, descendantPID, "ffmpeg descendant")
	})

	if err := worker.Process.Kill(); err != nil {
		t.Fatalf("hard-crash live worker helper: %v", err)
	}
	if err := worker.Wait(); err == nil {
		t.Fatal("hard-crashed live worker helper exited successfully")
	}

	waitForProcessExit(t, "ffmpeg", ffmpegPID)
	waitForProcessExit(t, "ffmpeg descendant", descendantPID)
}

func TestVodArchiveWorkerHelper(t *testing.T) {
	if os.Getenv("GANYMEDE_ARCHIVE_WORKER_HELPER") != "1" {
		return
	}

	tempDir := os.Getenv("GANYMEDE_ARCHIVE_TEST_DIR")
	cmd := osExec.Command(
		filepath.Join(tempDir, "yt-dlp"),
		filepath.Join(tempDir, "yt-dlp.pid"),
		filepath.Join(tempDir, "ffmpeg.pid"),
	)
	cmd.SysProcAttr = vodArchiveProcessAttributes()

	done, err := startArchiveCommand(cmd)
	if err != nil {
		t.Fatalf("start archive command: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("wait for archive command: %v", err)
	}
}

func TestLiveArchiveWorkerHelper(t *testing.T) {
	if os.Getenv("GANYMEDE_LIVE_ARCHIVE_WORKER_HELPER") != "1" {
		return
	}

	tempDir := os.Getenv("GANYMEDE_ARCHIVE_TEST_DIR")
	cmd := osExec.Command(
		filepath.Join(tempDir, "ffmpeg"),
		filepath.Join(tempDir, "ffmpeg.pid"),
		filepath.Join(tempDir, "ffmpeg-descendant.pid"),
	)
	cmd.SysProcAttr = liveArchiveProcessAttributes()

	done, err := startArchiveCommand(cmd)
	if err != nil {
		t.Fatalf("start live archive command: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("wait for live archive command: %v", err)
	}
}

func writeExecutable(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func waitForPIDFile(t *testing.T, path string) int {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		contents, err := os.ReadFile(path)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(contents)))
			if err != nil {
				t.Fatalf("parse PID from %s: %v", path, err)
			}
			return pid
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("read PID file %s: %v", path, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for PID file %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForProcessExit(t *testing.T, name string, pid int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) {
			return
		}
		if err != nil {
			t.Fatalf("read %s process state: %v", name, err)
		}
		fields := strings.Fields(string(stat))
		if len(fields) >= 3 && fields[2] == "Z" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s process %d remained after worker hard crash", name, pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func killTestProcess(t *testing.T, pid int, name string) {
	t.Helper()

	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		t.Errorf("clean up %s: %v", name, err)
	}
}

// waitForProcessGroupExit waits until the process group has no live (non
// zombie) members. Zombies are ignored because the test process is not their
// parent and reaping is deferred to init.
func waitForProcessGroupExit(t *testing.T, pgid int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if !processGroupHasLiveProcesses(pgid) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("process group %d still has live members after archive completion", pgid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func processGroupHasLiveProcesses(pgid int) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		contents := string(stat)
		closeParen := strings.LastIndex(contents, ")")
		if closeParen == -1 || closeParen+2 >= len(contents) {
			continue
		}
		// Remainder layout: state ppid pgrp ...
		fields := strings.Fields(contents[closeParen+1:])
		if len(fields) < 3 || fields[0] == "Z" {
			continue
		}
		group, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		if group == pgid {
			return true
		}
	}
	return false
}

func Test_extractSharedChatArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "empty",
			in:   nil,
			want: nil,
		},
		{
			name: "no shared flags",
			in:   []string{"-h", "1440", "-w", "340", "--font", "Inter"},
			want: nil,
		},
		{
			name: "equals form",
			in:   []string{"-h", "1440", "--stv=false", "--font", "Inter"},
			want: []string{"--stv=false"},
		},
		{
			name: "space form",
			in:   []string{"--bttv", "false", "-h", "1440"},
			want: []string{"--bttv", "false"},
		},
		{
			name: "all three providers mixed forms",
			in:   []string{"--framerate", "30", "--bttv=true", "--ffz", "false", "--stv=false"},
			want: []string{"--bttv=true", "--ffz", "false", "--stv=false"},
		},
		{
			name: "temp-path space form",
			in:   []string{"-h", "1440", "--temp-path", "/var/cache/td"},
			want: []string{"--temp-path", "/var/cache/td"},
		},
		{
			name: "temp-path equals form",
			in:   []string{"--temp-path=/var/cache/td", "--font", "Inter"},
			want: []string{"--temp-path=/var/cache/td"},
		},
		{
			name: "trailing flag without value",
			in:   []string{"--stv"},
			want: []string{"--stv"},
		},
		{
			name: "bare boolean does not swallow following flag",
			in:   []string{"--stv", "--temp-path", "/var/cache/td"},
			want: []string{"--stv", "--temp-path", "/var/cache/td"},
		},
		{
			name: "does not match prefix-only flags",
			in:   []string{"--stvthing", "--bttvfoo=1", "--temp-pathish"},
			want: nil,
		},
		{
			name: "collision is intentionally not forwarded",
			in:   []string{"--collision", "rename", "--stv=false"},
			want: []string{"--stv=false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSharedChatArgs(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractSharedChatArgs(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func Test_appendFFmpegLiveOutputStreamArgs(t *testing.T) {
	tests := []struct {
		name      string
		audioOnly bool
		want      []string
	}{
		{
			name:      "all streams",
			audioOnly: false,
			want:      []string{"-map", "0", "-dn", "-ignore_unknown", "-c", "copy"},
		},
		{
			name:      "audio only",
			audioOnly: true,
			want:      []string{"-map", "0:a", "-dn", "-ignore_unknown", "-c", "copy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendFFmpegLiveOutputStreamArgs(nil, tt.audioOnly)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendFFmpegLiveOutputStreamArgs(nil, %t) = %v, want %v", tt.audioOnly, got, tt.want)
			}
		})
	}
}
