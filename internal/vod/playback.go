package vod

import (
	"os"
	"path/filepath"

	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/hls"
)

// LivePreviewAvailable checks the capture output, independent of settings changed mid-recording.
func LivePreviewAvailable(video *ent.Vod) bool {
	captureID := hls.LiveCaptureID(video.ExtID, video.ExtStreamID, video.TmpVideoDownloadPath)
	if !video.Status.Active() || video.TmpVideoHlsPath == "" || captureID == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(video.TmpVideoHlsPath, captureID+"-video.m3u8"))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}
