package activities

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	osExec "os/exec"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	entChannel "github.com/zibbp/ganymede/ent/channel"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/internal/chapter"
	"github.com/zibbp/ganymede/internal/database"
	"github.com/zibbp/ganymede/internal/dto"
	"github.com/zibbp/ganymede/internal/exec"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
	"github.com/zibbp/ganymede/internal/vod"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

func sendHeartbeat(ctx context.Context, msg string, stop chan bool) {
	ticker := time.NewTicker(20 * time.Second)
	log.Debug().Msgf("starting heartbeat %s", msg)
	for {
		select {
		case <-ticker.C:
			activity.RecordHeartbeat(ctx, msg)
		case <-stop:
			log.Debug().Msgf("stopping heartbeat %s", msg)
			ticker.Stop()
			return
		}
	}
}

func convertTwitchChaptersToChapters(chapters []twitch.Node, duration int) ([]chapter.Chapter, error) {
	if len(chapters) == 0 {
		return nil, fmt.Errorf("no chapters found")
	}

	convertedChapters := make([]chapter.Chapter, len(chapters))
	for i := 0; i < len(chapters); i++ {
		convertedChapters[i].ID = chapters[i].ID
		convertedChapters[i].Title = chapters[i].Description
		convertedChapters[i].Type = string(chapters[i].Type)
		convertedChapters[i].Start = int(chapters[i].PositionMilliseconds / 1000)

		if i+1 < len(chapters) {
			convertedChapters[i].End = int(chapters[i+1].PositionMilliseconds / 1000)
		} else {
			convertedChapters[i].End = duration
		}
	}

	return convertedChapters, nil
}

func ArchiveVideoActivity(ctx context.Context, input dto.ArchiveVideoInput) error {
	return nil
}

func SaveTwitchVideoInfo(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, err := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Running).Save(ctx)
	if err != nil {
		return err
	}

	twitchService := twitch.NewService()
	twitchVideo, err := twitchService.GetVodByID(input.VideoID)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// get chapters
	twitchChapters, err := twitch.GQLGetChapters(input.VideoID)
	if err != nil {
		_, dbEr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
		if dbEr != nil {
			return dbEr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// convert twitch chapters to chapters
	// get nodes from gql response
	var nodes []twitch.Node
	for _, v := range twitchChapters.Data.Video.Moments.Edges {
		nodes = append(nodes, v.Node)
	}
	if len(nodes) > 0 {
		chapters, err := convertTwitchChaptersToChapters(nodes, input.Vod.Duration)
		if err != nil {
			_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
			if dbErr != nil {
				return dbErr
			}
			return temporal.NewApplicationError(err.Error(), "", nil)
		}
		// add chapters to database
		chapterService := chapter.NewService()
		for _, c := range chapters {
			_, err := chapterService.CreateChapter(c, input.Vod.ID)
			if err != nil {
				_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
				if dbErr != nil {
					return dbErr
				}
				return temporal.NewApplicationError(err.Error(), "", nil)
			}
		}

		twitchVideo.Chapters = chapters
	}

	// get muted segments
	mutedSegments, err := twitch.GQLGetMutedSegments(input.VideoID)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}
	cleanMutedSegments := []vod.MutedSegment{}

	// insert muted segments into database
	for _, mutedSegment := range mutedSegments.Data.Video.MuteInfo.MutedSegmentConnection.Nodes {
		segmentEnd := mutedSegment.Offset + mutedSegment.Duration
		if segmentEnd > input.Vod.Duration {
			segmentEnd = input.Vod.Duration
		}
		// insert muted segment into database
		_, err := database.DB().Client.MutedSegment.Create().SetStart(mutedSegment.Offset).SetEnd(segmentEnd).SetVod(input.Vod).Save(ctx)
		if err != nil {
			_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
			if dbErr != nil {
				return dbErr
			}
			return temporal.NewApplicationError(err.Error(), "", nil)
		}
		cleanMutedSegments = append(cleanMutedSegments, vod.MutedSegment{
			Start: mutedSegment.Offset,
			End:   segmentEnd,
		})
	}
	twitchVideo.MutedSegments = cleanMutedSegments

	err = utils.WriteJson(twitchVideo, fmt.Sprintf("%s/%s", input.Channel.Name, input.Vod.FolderName), fmt.Sprintf("%s-info.json", input.Vod.FileName))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, err = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Success).Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

func SaveTwitchLiveVideoInfo(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	twitchService := twitch.NewService()
	stream, err := twitchService.GetStreams(fmt.Sprintf("?user_login=%s", input.Channel.Name))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	if len(stream.Data) == 0 {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return fmt.Errorf("no stream found for channel %s", input.Channel.Name)
	}

	twitchVideo := stream.Data[0]

	err = utils.WriteJson(twitchVideo, fmt.Sprintf("%s/%s", input.Channel.Name, input.Vod.FolderName), fmt.Sprintf("%s-info.json", input.Vod.FileName))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, err = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodSaveInfo(utils.Success).Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

func DownloadTwitchThumbnails(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	twitchService := twitch.NewService()
	twitchVideo, err := twitchService.GetVodByID(input.VideoID)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	fullResThumbnailUrl := replacePlaceholders(twitchVideo.ThumbnailURL, "1920", "1080")
	webResThumbnailUrl := replacePlaceholders(twitchVideo.ThumbnailURL, "640", "360")

	err = utils.DownloadFile(fullResThumbnailUrl, fmt.Sprintf("%s/%s", input.Channel.Name, input.Vod.FolderName), fmt.Sprintf("%s-thumbnail.jpg", input.Vod.FileName))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	err = utils.DownloadFile(webResThumbnailUrl, fmt.Sprintf("%s/%s", input.Channel.Name, input.Vod.FolderName), fmt.Sprintf("%s-web_thumbnail.jpg", input.Vod.FileName))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, err = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Success).Save(ctx)
	if err != nil {
		return err
	}

	return nil
}

func DownloadTwitchLiveThumbnails(ctx context.Context, input dto.ArchiveVideoInput) error {

	twitchService := twitch.NewService()
	stream, err := twitchService.GetStreams(fmt.Sprintf("?user_login=%s", input.Channel.Name))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	if len(stream.Data) == 0 {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		// stream isn't live so archive shouldn't continue and should be cleaned up
		return temporal.NewApplicationError(fmt.Sprintf("no stream found for channel %s", input.Channel.Name), "", nil)
	}

	twitchVideo := stream.Data[0]

	fullResThumbnailUrl := replaceLivePlaceholders(twitchVideo.ThumbnailURL, "1920", "1080")
	webResThumbnailUrl := replaceLivePlaceholders(twitchVideo.ThumbnailURL, "640", "360")

	err = utils.DownloadFile(fullResThumbnailUrl, fmt.Sprintf("%s/%s", input.Channel.Name, input.Vod.FolderName), fmt.Sprintf("%s-thumbnail.jpg", input.Vod.FileName))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	err = utils.DownloadFile(webResThumbnailUrl, fmt.Sprintf("%s/%s", input.Channel.Name, input.Vod.FolderName), fmt.Sprintf("%s-web_thumbnail.jpg", input.Vod.FileName))
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Failed).Save(ctx)
		if dbErr != nil {
			return dbErr
		}
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVodDownloadThumbnail(utils.Success).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	return nil
}

func replacePlaceholders(url, width, height string) string {
	url = strings.ReplaceAll(url, "%{width}", width)
	url = strings.ReplaceAll(url, "%{height}", height)
	return url
}
func replaceLivePlaceholders(url, width, height string) string {
	url = strings.ReplaceAll(url, "{width}", width)
	url = strings.ReplaceAll(url, "{height}", height)
	return url
}

func DownloadTwitchVideo(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoDownload(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("download-video-%s", input.VideoID), stopHeartbeat)

	// Start the download
	err := exec.DownloadTwitchVodVideo(input.Vod)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoDownload(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoDownload(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	stopHeartbeat <- true
	return nil
}

func DownloadTwitchLiveVideo(ctx context.Context, input dto.ArchiveVideoInput, ch chan bool) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoDownload(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("download-livevideo-%s", input.VideoID), stopHeartbeat)

	// Start the download
	err := exec.DownloadTwitchLiveVideo(ctx, input.Vod, input.Channel, input.LiveChatWorkflowId)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoDownload(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return temporal.NewApplicationError(err.Error(), "", nil)
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoDownload(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// Update video duration with duration from downloaded video
	duration, err := exec.GetVideoDuration(fmt.Sprintf("/tmp/%s_%s-video.mp4", input.Vod.ExtID, input.Vod.ID))
	if err != nil {
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}
	_, dbErr = database.DB().Client.Vod.UpdateOneID(input.Vod.ID).SetDuration(duration).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	// attempt to find vod id of the livesstream so the external id is correct
	videos, err := twitch.GetVideosByUser(input.Channel.ExtID, "archive")
	if err != nil {
		stopHeartbeat <- true
		log.Err(err).Msg("error getting videos from twitch api")
	}

	// attempt to find vod of current livestream
	var livestreamVodId string
	for _, video := range videos {
		if video.StreamID == input.Vod.ExtID {
			livestreamVodId = video.ID
			log.Info().Msgf("found vod id %s for livestream %s, updating database", livestreamVodId, input.Vod.ExtID)
			// update vod with external id
			_, dbErr = database.DB().Client.Vod.UpdateOneID(input.Vod.ID).SetExtID(livestreamVodId).Save(ctx)
			if dbErr != nil {
				stopHeartbeat <- true
				log.Err(dbErr).Msg("error updating vod with external id")
			}
		}
	}

	if livestreamVodId == "" {
		log.Info().Msgf("no vod found for livestream %s, keeping live stream ID as external id", input.Vod.ExtID)
	}

	stopHeartbeat <- true
	return nil
}

func PostprocessVideo(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoConvert(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("postprocess-video-%s", input.VideoID), stopHeartbeat)

	// Start post process
	err := exec.ConvertTwitchVodVideo(input.Vod)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoConvert(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// Convert to HLS if needed
	if viper.GetBool("archive.save_as_hls") {
		err = exec.ConvertToHLS(input.Vod)
		if err != nil {
			_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoConvert(utils.Failed).Save(ctx)
			if dbErr != nil {
				stopHeartbeat <- true
				return dbErr
			}
			stopHeartbeat <- true
			return temporal.NewApplicationError(err.Error(), "", nil)
		}
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoConvert(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	stopHeartbeat <- true
	return nil
}

func MoveVideo(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoMove(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("move-video-%s", input.VideoID), stopHeartbeat)

	if viper.GetBool("archive.save_as_hls") {
		sourcePath := fmt.Sprintf("/tmp/%s_%s-video_hls0", input.Vod.ExtID, input.Vod.ID)
		destPath := fmt.Sprintf("/vods/%s/%s/%s-video_hls", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName)
		err := utils.MoveFolder(sourcePath, destPath)
		if err != nil {
			_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoMove(utils.Failed).Save(ctx)
			if dbErr != nil {
				stopHeartbeat <- true
				return dbErr
			}
			stopHeartbeat <- true
			return temporal.NewApplicationError(err.Error(), "", nil)
		}
		// Update video path to hls path
		_, dbErr = database.DB().Client.Vod.UpdateOneID(input.Vod.ID).SetVideoPath(fmt.Sprintf("/vods/%s/%s/%s-video_hls/%s-video.m3u8", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName, input.Vod.ExtID)).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
	} else {
		sourcePath := fmt.Sprintf("/tmp/%s_%s-video-convert.mp4", input.Vod.ExtID, input.Vod.ID)
		destPath := fmt.Sprintf("/vods/%s/%s/%s-video.mp4", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName)

		err := utils.MoveFile(sourcePath, destPath)
		if err != nil {
			_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoMove(utils.Failed).Save(ctx)
			if dbErr != nil {
				stopHeartbeat <- true
				return dbErr
			}
			stopHeartbeat <- true
			return temporal.NewApplicationError(err.Error(), "", nil)
		}
	}

	// Clean up files
	// Delete source file
	err := utils.DeleteFile(fmt.Sprintf("/tmp/%s_%s-video.mp4", input.Vod.ExtID, input.Vod.ID))
	if err != nil {
		log.Info().Err(err).Msgf("error deleting source file for vod %s", input.Vod.ID)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskVideoMove(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	stopHeartbeat <- true
	return nil
}

func DownloadTwitchChat(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("download-chat-%s", input.VideoID), stopHeartbeat)

	// Start the download
	err := exec.DownloadTwitchVodChat(input.Vod)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// copy json to vod folder
	sourcePath := fmt.Sprintf("/tmp/%s_%s-chat.json", input.Vod.ExtID, input.Vod.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-chat.json", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName)

	err = utils.CopyFile(sourcePath, destPath)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	stopHeartbeat <- true
	return nil
}

func DownloadTwitchLiveChat(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("download-livechat-%s", input.VideoID), stopHeartbeat)

	// Start the download
	err := exec.DownloadTwitchLiveChat(ctx, input.Vod, input.Channel, input.Queue)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// copy json to vod folder
	sourcePath := fmt.Sprintf("/tmp/%s_%s-live-chat.json", input.Vod.ExtID, input.Vod.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-live-chat.json", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName)

	err = utils.CopyFile(sourcePath, destPath)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}
	stopHeartbeat <- true

	return nil
}

func RenderTwitchChat(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatRender(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("render-chat-%s", input.VideoID), stopHeartbeat)

	// Start the download
	err, _ := exec.RenderTwitchVodChat(input.Vod)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatRender(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatRender(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	stopHeartbeat <- true

	return nil
}

func MoveChat(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatMove(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("move-chat-%s", input.VideoID), stopHeartbeat)

	sourcePath := fmt.Sprintf("/tmp/%s_%s-chat.json", input.Vod.ExtID, input.Vod.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-chat.json", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName)

	err := utils.MoveFile(sourcePath, destPath)
	if err != nil {
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatMove(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	if input.Queue.RenderChat {
		sourcePath = fmt.Sprintf("/tmp/%s_%s-chat.mp4", input.Vod.ExtID, input.Vod.ID)
		destPath = fmt.Sprintf("/vods/%s/%s/%s-chat.mp4", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName)

		err = utils.MoveFile(sourcePath, destPath)
		if err != nil {
			_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatMove(utils.Failed).Save(ctx)
			if dbErr != nil {
				stopHeartbeat <- true
				return dbErr
			}
			stopHeartbeat <- true
			return temporal.NewApplicationError(err.Error(), "", nil)
		}
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatMove(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	stopHeartbeat <- true
	return nil
}

func KillTwitchLiveChatDownload(ctx context.Context, input dto.ArchiveVideoInput) error {

	log.Info().Msgf("killing chat downloader for channel %s", input.Channel.Name)

	// find pid of chat_downloader to kill
	cmd := osExec.Command("pgrep", "-f", fmt.Sprintf("chat_downloader https://twitch.tv/%s", input.Channel.Name))
	out, err := cmd.Output()
	if err != nil {
		return temporal.NewApplicationError(err.Error(), "", nil)
	}
	// convert out to a string and remove newline
	pid := strings.TrimSpace(string(out))
	pid = strings.ReplaceAll(pid, "\n", "")
	log.Debug().Msgf("found pid %s for chat_downloader", string(out))

	// kill pid
	cmd = osExec.Command("kill", "-15", pid)
	_, err = cmd.Output()
	if err != nil {
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	log.Info().Msgf("killed chat downloader for channel %s", input.Channel.Name)

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatDownload(utils.Success).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	return nil
}

func ConvertTwitchLiveChat(ctx context.Context, input dto.ArchiveVideoInput) error {

	_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Running).Save(ctx)
	if dbErr != nil {
		return dbErr
	}

	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, fmt.Sprintf("convert-livechat-%s", input.VideoID), stopHeartbeat)

	// Check if chat file exists
	chatPath := fmt.Sprintf("/tmp/%s_%s-live-chat.json", input.Vod.ExtID, input.Vod.ID)
	if !utils.FileExists(chatPath) {
		log.Debug().Msgf("chat file does not exist %s - this means there were no chat messages - setting chat to complete", chatPath)
		// Set queue chat task to complete
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Success).SetTaskChatRender(utils.Success).SetTaskChatMove((utils.Success)).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		// Set VOD chat to empty
		_, dbErr = database.DB().Client.Vod.UpdateOneID(input.Vod.ID).SetChatVideoPath("").SetChatPath("").Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return nil
	}

	// Fetch streamer from Twitch API for their user ID
	streamer, err := twitch.API.GetUserByLogin(input.Channel.Name)
	if err != nil {
		log.Error().Err(err).Msg("error getting streamer from Twitch API")
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}
	cID, err := strconv.Atoi(streamer.ID)
	if err != nil {
		log.Error().Err(err).Msg("error converting streamer ID to int")
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// update queue item
	updatedQueue, dbErr := database.DB().Client.Queue.Get(ctx, input.Queue.ID)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}
	input.Queue = updatedQueue
	log.Info().Msgf("streamer ID: %s", streamer.ID)
	// TwitchDownloader requires the ID of the video, or at least a previous video ID
	videos, err := twitch.GetVideosByUser(streamer.ID, "archive")
	if err != nil {
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// attempt to find vod of current livestream
	var previousVideoID string
	for _, video := range videos {
		if video.StreamID == input.Vod.ExtID {
			previousVideoID = video.ID
		}
	}
	// If no previous video ID was found, use a random id
	if previousVideoID == "" {
		log.Warn().Msgf("Stream %s on channel %s has no previous video ID, using %s", input.VideoID, input.Channel.Name, previousVideoID)
		previousVideoID = "132195945"
	}

	err = utils.ConvertTwitchLiveChatToVodChat(fmt.Sprintf("/tmp/%s_%s-live-chat.json", input.Vod.ExtID, input.Vod.ID), input.Channel.Name, input.Vod.ID.String(), input.Vod.ExtID, cID, input.Queue.ChatStart, string(previousVideoID))
	if err != nil {
		log.Error().Err(err).Msg("error converting chat")
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// TwitchDownloader "chatupdate"
	// Embeds emotes and badges into the chat file
	err = exec.TwitchChatUpdate(input.Vod)
	if err != nil {
		log.Error().Err(err).Msg("error updating chat")
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	// copy converted chat
	sourcePath := fmt.Sprintf("/tmp/%s_%s-chat-convert.json", input.Vod.ExtID, input.Vod.ID)
	destPath := fmt.Sprintf("/vods/%s/%s/%s-chat-convert.json", input.Channel.Name, input.Vod.FolderName, input.Vod.FileName)

	err = utils.CopyFile(sourcePath, destPath)
	if err != nil {
		log.Error().Err(err).Msg("error copying chat convert")
		_, dbErr := database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Failed).Save(ctx)
		if dbErr != nil {
			stopHeartbeat <- true
			return dbErr
		}
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	_, dbErr = database.DB().Client.Queue.UpdateOneID(input.Queue.ID).SetTaskChatConvert(utils.Success).Save(ctx)
	if dbErr != nil {
		stopHeartbeat <- true
		return dbErr
	}

	stopHeartbeat <- true
	return nil
}

func TwitchSaveVideoChapters(ctx context.Context) error {
	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, "save-video-chapters", stopHeartbeat)

	// get all videos
	videos, err := database.DB().Client.Vod.Query().All(ctx)
	if err != nil {
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	for _, video := range videos {
		if video.Type == "live" {
			continue
		}
		if video.ExtID == "" {
			continue
		}
		log.Debug().Msgf("getting chapters for video %s", video.ID)
		// get chapters
		twitchChapters, err := twitch.GQLGetChapters(video.ExtID)
		if err != nil {
			log.Error().Err(err).Msgf("error getting chapters for video %s", video.ID)
			continue
		}

		// convert twitch chapters to chapters
		// get nodes from gql response
		var nodes []twitch.Node
		for _, v := range twitchChapters.Data.Video.Moments.Edges {
			nodes = append(nodes, v.Node)
		}
		if len(nodes) > 0 {
			chapters, err := convertTwitchChaptersToChapters(nodes, video.Duration)
			if err != nil {
				return temporal.NewApplicationError(err.Error(), "", nil)
			}
			// add chapters to database
			chapterService := chapter.NewService()
			// check if chapters already exist
			existingChapters, err := chapterService.GetVideoChapters(video.ID)
			if err != nil {
				log.Error().Err(err).Msgf("error getting chapters for video %s", video.ID)
			}
			if len(existingChapters) > 0 {
				log.Debug().Msgf("chapters already exist for video %s", video.ID)
				continue
			}

			for _, c := range chapters {
				_, err := chapterService.CreateChapter(c, video.ID)
				if err != nil {
					stopHeartbeat <- true
					return temporal.NewApplicationError(err.Error(), "", nil)
				}
			}
			log.Info().Msgf("added %d chapters to video %s", len(chapters), video.ID)
		}
		// sleep for 0.25 seconds to not hit rate limit
		time.Sleep(250 * time.Millisecond)
	}
	stopHeartbeat <- true
	return nil
}

func UpdateTwitchLiveStreamArchivesWithVodIds(ctx context.Context) error {
	stopHeartbeat := make(chan bool)
	go sendHeartbeat(ctx, "update-video-ids", stopHeartbeat)

	// get all channels
	channels, err := database.DB().Client.Channel.Query().All(ctx)
	if err != nil {
		stopHeartbeat <- true
		return temporal.NewApplicationError(err.Error(), "", nil)
	}

	for _, channel := range channels {
		log.Info().Msgf("processing channel %s", channel.Name)
		// get all videos for channel
		videos, err := database.DB().Client.Vod.Query().Where(entVod.HasChannelWith(entChannel.ID(channel.ID))).All(ctx)
		if err != nil {
			stopHeartbeat <- true
			return temporal.NewApplicationError(err.Error(), "", nil)
		}

		// get all videos from twitch for channel
		twitchChannelVideoss, err := twitch.GetVideosByUser(channel.ExtID, "archive")
		if err != nil {
			stopHeartbeat <- true
			return temporal.NewApplicationError(err.Error(), "", nil)
		}

		for _, video := range videos {
			if video.Type != "live" {
				continue
			}
			if video.ExtID == "" {
				continue
			}
			// find video in twitch videos
			for _, twitchVideo := range twitchChannelVideoss {
				if video.ExtID == twitchVideo.StreamID {
					log.Debug().Msgf("found video %s in twitch videos", video.ExtID)
					// update video with vod id
					_, err := database.DB().Client.Vod.UpdateOneID(video.ID).SetExtID(twitchVideo.ID).Save(ctx)
					if err != nil {
						stopHeartbeat <- true
						return temporal.NewApplicationError(err.Error(), "", nil)
					}
				}
			}

		}
	}
	stopHeartbeat <- true
	return nil
}
