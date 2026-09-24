package vod

import (
	"os"
	"path/filepath"

	"github.com/zibbp/ganymede/ent"
)

// LivePreviewAvailable checks the capture output, independent of settings changed mid-recording.
func LivePreviewAvailable(video *ent.Vod) bool {
	if !video.Status.Active() || video.TmpVideoHlsPath == "" || video.ExtID == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(video.TmpVideoHlsPath, video.ExtID+"-video.m3u8"))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}
