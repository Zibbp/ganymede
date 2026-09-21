package exec

import (
	"context"
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

// TestStartArchiveCommandGracefulTermExitsPromptly mirrors the user-stop path
// used by DownloadTwitchLiveVideo (SIGTERM to the process group) and verifies
// the shim reports completion as soon as ffmpeg exits instead of waiting out
// the SIGKILL backstop, so post-processing sees a fully flushed playlist.
func TestStartArchiveCommandGracefulTermExitsPromptly(t *testing.T) {
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
trap 'exit 0' TERM
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

	// Both fakes exit on SIGTERM, so everything must be gone quickly. If the
	// shim waited out its SIGKILL backstop instead of reaping the helper,
	// the deadlines below catch it.
	waitForProcessGroupExit(t, pgid, 5*time.Second)
	waitForProcessExit(t, "ffmpeg", ffmpegPID)
	waitForProcessExit(t, "ffmpeg descendant", descendantPID)
}

// TestArchiveProcessForwarderKillsGroupOnParentDeathSignal covers the worker
// crash path: the kernel delivers Pdeathsig (SIGUSR1) to the shim, which must
// kill the whole group immediately even when ffmpeg ignores SIGTERM while
// blocked on a network read.
func TestArchiveProcessForwarderKillsGroupOnParentDeathSignal(t *testing.T) {
	tempDir := t.TempDir()
	ffmpegPIDPath := filepath.Join(tempDir, "ffmpeg.pid")
	descendantPIDPath := filepath.Join(tempDir, "ffmpeg-descendant.pid")
	descendantPath := filepath.Join(tempDir, "ffmpeg-descendant")

	writeExecutable(t, filepath.Join(tempDir, "ffmpeg"), `#!/bin/sh
printf '%s' "$$" > "$1"
trap '' TERM
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
	shimPID := cmd.Process.Pid
	t.Cleanup(func() {
		killTestProcess(t, -pgid, "live archive process group")
		killTestProcess(t, ffmpegPID, "ffmpeg")
		killTestProcess(t, descendantPID, "ffmpeg descendant")
	})

	// SIGUSR1 targets only the shim (like Pdeathsig), which must SIGKILL
	// the group without any grace period.
	if err := syscall.Kill(shimPID, syscall.SIGUSR1); err != nil {
		t.Fatalf("send SIGUSR1 to archive process forwarder: %v", err)
	}

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for archive command to finish after parent-death signal")
	}

	waitForProcessGroupExit(t, pgid, 5*time.Second)
	waitForProcessExit(t, "ffmpeg", ffmpegPID)
	waitForProcessExit(t, "ffmpeg descendant", descendantPID)
}

func TestLiveArchiveProcessAttributes(t *testing.T) {
	t.Parallel()

	attrs := liveArchiveProcessAttributes()
	if !attrs.Setpgid {
		t.Fatal("live archive process must run in its own process group")
	}
	if attrs.Pdeathsig != syscall.SIGUSR1 {
		t.Fatalf("parent death signal = %v, want SIGUSR1", attrs.Pdeathsig)
	}
}

func TestVodArchiveProcessAttributes(t *testing.T) {
	t.Parallel()

	attrs := vodArchiveProcessAttributes()
	if !attrs.Setpgid {
		t.Fatal("VOD archive process must run in its own process group")
	}
	if attrs.Pdeathsig != syscall.SIGUSR1 {
		t.Fatalf("parent death signal = %v, want SIGUSR1", attrs.Pdeathsig)
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

	args := postProcessVideoFFmpegArgs(video, "-c:v copy -c:a copy")

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

	if args[len(args)-1] != video.TmpVideoConvertPath {
		t.Fatalf("last FFmpeg argument = %q, want output path %q", args[len(args)-1], video.TmpVideoConvertPath)
	}
}

func TestNormalizedPostProcessVideoFFmpegArgsResetEachStream(t *testing.T) {
	t.Parallel()

	video := ent.Vod{
		TmpVideoDownloadPath: "/tmp/input.ts",
		TmpVideoConvertPath:  "/tmp/output.mp4",
	}
	args := normalizedPostProcessVideoFFmpegArgs(video, "-c:v copy -c:a copy")

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-bsf" && args[i+1] == "setts=ts=TS-STARTDTS" {
			return
		}
	}
	t.Fatalf("normalized FFmpeg arguments do not reset stream timestamps: %v", args)
}

func TestPostProcessVideoNormalizesAnomalousContainerTimeline(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTimestampOffsetMedia(t, tmpDir, "mp4")
	outputPath := filepath.Join(tmpDir, "normalized.mp4")
	video := ent.Vod{
		Title:                "timestamp normalization regression",
		TmpVideoDownloadPath: inputPath,
		TmpVideoConvertPath:  outputPath,
	}

	if err := postProcessVideo(t.Context(), video, "-c:v copy -c:a copy", io.Discard); err != nil {
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

// TestLiveArchiveProcessGroupDiesOnParentDeathDespiteTermIgnoringFFmpeg covers
// the crash recovery path where ffmpeg ignores SIGTERM while blocked on a
// network read. The kernel delivers Pdeathsig (SIGUSR1) to the forwarding
// shim, which must SIGKILL the capture process group immediately so it does
// not outlive a crashed worker.
func TestLiveArchiveProcessGroupDiesOnParentDeathDespiteTermIgnoringFFmpeg(t *testing.T) {
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

func TestBuildHlsToMp4FFmpegArgsUsesPlaylistInput(t *testing.T) {
	t.Parallel()

	video := ent.Vod{
		Title:           "live title",
		TmpVideoHlsPath: "/tmp/hls",
		ExtID:           "123",
	}
	playlist := "/tmp/hls/123-video.m3u8"
	export := "/tmp/export.mp4"

	args := buildHlsToMp4FFmpegArgs(video, playlist, export, "-c:v copy -c:a copy", false)

	if args[0] != "-y" || args[len(args)-1] != export {
		t.Fatalf("unexpected MP4 export args shape: %v", args)
	}
	foundInput := false
	foundMovflags := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-i" && args[i+1] == playlist {
			foundInput = true
		}
		if args[i] == "-movflags" && args[i+1] == "+faststart" {
			foundMovflags = true
		}
	}
	if !foundInput {
		t.Fatalf("MP4 export args missing playlist input %q: %v", playlist, args)
	}
	if !foundMovflags {
		t.Fatalf("MP4 export args missing +faststart: %v", args)
	}
}

func TestTempExportTargetPublishesOnlyOnCommit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dest := filepath.Join(dir, "export.mp4")

	tmpPath, commit, cleanup, err := tempExportTarget(dest)
	if err != nil {
		t.Fatalf("tempExportTarget: %v", err)
	}
	defer cleanup()

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("destination must not exist before commit, stat err = %v", err)
	}
	if err := os.WriteFile(tmpPath, []byte("media"), 0o600); err != nil {
		t.Fatalf("write temporary export: %v", err)
	}
	if err := commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read committed export: %v", err)
	}
	if string(data) != "media" {
		t.Fatalf("committed export = %q, want %q", data, "media")
	}
}

func TestRunHlsToMp4FFmpegRemovesPartialOutputOnCancel(t *testing.T) {
	tempDir := t.TempDir()
	writeExecutable(t, filepath.Join(tempDir, "ffmpeg"), `#!/bin/sh
for last; do :; done
printf 'partial' > "$last"
exec sleep 30
`)
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	exportPath := filepath.Join(tempDir, "export.mp4")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runHlsToMp4FFmpeg(ctx, ent.Vod{}, []string{"-y", exportPath}, exportPath, io.Discard)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(exportPath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for partial output")
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("runHlsToMp4FFmpeg error = %v, want context.Canceled", err)
	}
	if _, err := os.Stat(exportPath); !os.IsNotExist(err) {
		t.Fatalf("partial output must be removed on cancellation, stat err = %v", err)
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

func TestLiveCaptureIDUsesImmutableStreamID(t *testing.T) {
	t.Parallel()

	video := ent.Vod{
		ExtID:                "vod999",
		ExtStreamID:          "stream123",
		TmpVideoDownloadPath: "/tmp/stream123_uuid-video_hls0/stream123-video.m3u8",
	}
	if got := liveCaptureID(video); got != "stream123" {
		t.Fatalf("liveCaptureID = %q, want stream123", got)
	}

	legacy := ent.Vod{
		ExtID:                "vod999",
		TmpVideoDownloadPath: "/tmp/stream123_uuid-video_hls0/stream123-video.m3u8",
	}
	if got := liveCaptureID(legacy); got != "stream123" {
		t.Fatalf("liveCaptureID fallback = %q, want stream123", got)
	}
}

func TestEnsureLiveHlsPlaylistAfterStreamVideoIDUpdate(t *testing.T) {
	t.Parallel()

	const streamID = "stream123"
	dir := t.TempDir()
	playlistPath := filepath.Join(dir, streamID+"-video.m3u8")
	if err := os.WriteFile(playlistPath, []byte("#EXTM3U\n#EXT-X-PLAYLIST-TYPE:EVENT\n#EXTINF:10.0,\nseg.ts\n"), 0o644); err != nil {
		t.Fatalf("write capture playlist: %v", err)
	}
	video := ent.Vod{
		ExtID:                "vod999",
		ExtStreamID:          streamID,
		TmpVideoHlsPath:      dir,
		TmpVideoDownloadPath: playlistPath,
	}

	rescued, err := EnsureLiveHlsPlaylist(t.Context(), video.TmpVideoHlsPath, liveCaptureID(video))
	if err != nil {
		t.Fatalf("EnsureLiveHlsPlaylist with immutable ID: %v", err)
	}
	if rescued != playlistPath {
		t.Fatalf("rescued playlist = %q, want %q", rescued, playlistPath)
	}
	if _, err := EnsureLiveHlsPlaylist(t.Context(), dir, video.ExtID); err == nil {
		t.Fatal("expected mutated VOD ID lookup to miss the stream-ID capture")
	}
}
