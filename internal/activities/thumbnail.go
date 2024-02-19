package activities

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/utils"
	"go.temporal.io/sdk/temporal"
)

func GenerateThumbnailsForVideo(ctx context.Context, vID string) (*string, error) {
	videoID, err := uuid.Parse(vID)
	if err != nil {
		return nil, temporal.NewApplicationError("error parsing video id", "", err)
	}

	stopHeatbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("generate-thumbnails-%s", videoID.String()), stopHeatbeat)

	dbVideo, err := database.DB().Client.Vod.Query().Where(entVod.ID(videoID)).WithChannel().Only(ctx)
	if err != nil {
		stopHeatbeat <- true
		return nil, temporal.NewApplicationError("error getting video", "", err)
	}

	// use ffprobe to find what the codec of the video is
	// that way we can hardware decode it
	ffprobeArgs := []string{
		"-v",
		"error",
		"-select_streams",
		"v:0",
		"-show_entries",
		"stream=codec_name",
		"-of",
		"default=noprint_wrappers=1:nokey=1",
	}

	videoCodec, err := exec.ExecuteFFprobeCommand(ffprobeArgs, dbVideo.VideoPath)
	if err != nil {
		stopHeatbeat <- true
		return nil, temporal.NewApplicationError("error getting video codec", "", err)
	}

	// trim whitespace
	videoCodec = string([]rune(videoCodec)[:len(videoCodec)-1])

	hardwareDecode := false
	inputCodec := ""

	switch videoCodec {
	case "h264":
		if hardwareDecode {
			inputCodec = "h264_qsv"
		} else {
			inputCodec = "h264"
		}
	case "hevc":
		if hardwareDecode {
			inputCodec = "hevc_qsv"
		} else {
			inputCodec = "hevc"
		}
	case "vp9":
		if hardwareDecode {
			inputCodec = "vp9_qsv"
		} else {
			inputCodec = "vp9"
		}
	case "av1":
		if hardwareDecode {
			inputCodec = "av1_qsv"
		} else {
			inputCodec = "av1"
		}
	}

	preFFmpegArgs := []string{}

	// append "pre" arguments
	if hardwareDecode {
		preFFmpegArgs = append(preFFmpegArgs, "", "-hwaccel", "qsv", "-hwaccel_output_format", "qsv", "-c:v", inputCodec, "-an")
	} else {
		preFFmpegArgs = append(preFFmpegArgs, "-an")
	}

	postFFmpegArgs := []string{}

	// append "post" arguments
	if hardwareDecode {
		postFFmpegArgs = append(postFFmpegArgs, "-filter:v", "fps=1/10,vpp_qsv=w=160:h=90:async_depth=4,tile=5x5", "-c:v", "mjpeg_qsv", "-global_quality:v", "75")
	} else {
		postFFmpegArgs = append(postFFmpegArgs, "-filter:v", "fps=1/10,scale=160:90,tile=5x5", "-c:v", "mjpeg")
	}

	thumbnailPath := fmt.Sprintf("/vods/%s/%s/thumbnails", dbVideo.Edges.Channel.Name, dbVideo.FolderName)

	// create folder for thumbnails
	err = utils.CreateFolder(thumbnailPath)
	if err != nil {
		stopHeatbeat <- true
		return nil, temporal.NewApplicationError("error creating folder for thumbnails", "", err)
	}

	// append output file
	ffmpegOutputPath := fmt.Sprintf("%s/thumb_%%03d.jpg", thumbnailPath)

	// execute ffmpeg command
	out, err := exec.ExecuteFFmpegCommand(preFFmpegArgs, postFFmpegArgs, dbVideo.VideoPath, ffmpegOutputPath)
	if err != nil {
		stopHeatbeat <- true
		return nil, temporal.NewApplicationError("error executing ffmpeg command", out, err)
	}

	log.Debug().Msgf("finished generating thumbnails for video %s", videoID.String())
	stopHeatbeat <- true
	return &out, nil
}
