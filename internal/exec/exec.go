package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	osExec "os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/twitch"
	"github.com/zibbp/ganymede/internal/utils"
)

func DownloadTwitchVodVideo(v *ent.Vod) error {

	var argArr []string
	// Check if twitch token is set
	argArr = append(argArr, fmt.Sprintf("https://twitch.tv/videos/%s", v.ExtID), fmt.Sprintf("%s,best", v.Resolution), "--force-progress", "--force")

	twitchToken := viper.GetString("parameters.twitch_token")
	if twitchToken != "" {
		// Note: if the token is invalid, streamlink will exit with "no playable streams found on this URL"
		argArr = append(argArr, fmt.Sprintf("--twitch-api-header=Authorization=OAuth %s", twitchToken))
	}

	argArr = append(argArr, "-o", fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID))

	log.Debug().Msgf("running streamlink for vod video download: %s", strings.Join(argArr, " "))

	cmd := osExec.Command("streamlink", argArr...)

	videoLogfile, err := os.Create(fmt.Sprintf("/logs/%s_%s-video.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating video logfile")
		return err
	}
	defer videoLogfile.Close()
	cmd.Stdout = videoLogfile
	cmd.Stderr = videoLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running streamlink for vod video download")
		return err
	}

	log.Debug().Msgf("finished downloading vod video for %s", v.ExtID)
	return nil
}

func DownloadTwitchVodChat(v *ent.Vod) error {
	cmd := osExec.Command("TwitchDownloaderCLI", "chatdownload", "--id", v.ExtID, "--embed-images", "-o", fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID))

	chatLogfile, err := os.Create(fmt.Sprintf("/logs/%s_%s-chat.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating chat logfile")
		return err
	}
	defer chatLogfile.Close()
	cmd.Stdout = chatLogfile
	cmd.Stderr = chatLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running TwitchDownloaderCLI for vod chat download")
		return err
	}

	log.Debug().Msgf("finished downloading vod chat for %s", v.ExtID)
	return nil
}

func RenderTwitchVodChat(v *ent.Vod, q *ent.Queue) (error, bool) {
	// Fetch config params
	chatRenderParams := viper.GetString("parameters.chat_render")
	// Split supplied params into array
	arr := strings.Fields(chatRenderParams)
	// Generate args for exec
	argArr := []string{"chatrender", "-i", fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID)}
	// add each config param to arg
	for _, v := range arr {
		argArr = append(argArr, v)
	}
	// add output file
	argArr = append(argArr, "-o", fmt.Sprintf("/tmp/%s_%s-chat.mp4", v.ExtID, v.ID))
	log.Debug().Msgf("chat render args: %v", argArr)
	// Execute chat render
	cmd := osExec.Command("TwitchDownloaderCLI", argArr...)

	chatRenderLogfile, err := os.Create(fmt.Sprintf("/logs/%s_%s-chat-render.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating chat render logfile")
		return err, true
	}
	defer chatRenderLogfile.Close()
	cmd.Stdout = chatRenderLogfile
	cmd.Stderr = chatRenderLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running TwitchDownloaderCLI for vod chat render")

		// Check if error is because of no messages
		checkCmd := fmt.Sprintf("cat /logs/%s_%s-chat-render.log | grep 'Sequence contains no elements'", v.ExtID, v.ID)
		_, err := osExec.Command("bash", "-c", checkCmd).Output()
		if err != nil {
			log.Error().Err(err).Msg("error checking chat render logfile for no messages")
			return err, true
		}

		log.Debug().Msg("no messages found in chat render logfile. setting vod and queue to reflect no chat.")
		v.Update().SetChatPath("").SetChatVideoPath("").SaveX(context.Background())
		q.Update().SetChatProcessing(false).SetTaskChatMove(utils.Success).SaveX(context.Background())
		return nil, false
	}

	log.Debug().Msgf("finished vod chat render for %s", v.ExtID)
	return nil, true
}

func ConvertTwitchVodVideo(v *ent.Vod) error {
	// Fetch config params
	ffmpegParams := viper.GetString("parameters.video_convert")
	// Split supplied params into array
	arr := strings.Fields(ffmpegParams)
	// Generate args for exec
	argArr := []string{"-y", "-hide_banner", "-i", fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID)}
	// add each config param to arg
	for _, v := range arr {
		argArr = append(argArr, v)
	}
	// add output file
	argArr = append(argArr, fmt.Sprintf("/tmp/%s_%s-video-convert.mp4", v.ExtID, v.ID))
	log.Debug().Msgf("video convert args: %v", argArr)
	// Execute ffmpeg
	cmd := osExec.Command("ffmpeg", argArr...)

	videoConvertLogfile, err := os.Create(fmt.Sprintf("/logs/%s_%s-video-convert.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating video convert logfile")
		return err
	}
	defer videoConvertLogfile.Close()
	cmd.Stdout = videoConvertLogfile
	cmd.Stderr = videoConvertLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running ffmpeg for vod video convert")
		return err
	}

	log.Debug().Msgf("finished vod video convert for %s", v.ExtID)
	return nil
}

func ConvertToHLS(v *ent.Vod) error {
	// Delete original video file to save space
	log.Debug().Msgf("deleting original video file for %s to save space", v.ExtID)
	if err := os.Remove(fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID)); err != nil {
		log.Error().Err(err).Msg("error deleting original video file")
		return err
	}

	cmd := osExec.Command("ffmpeg", "-y", "-hide_banner", "-i", fmt.Sprintf("/tmp/%s_%s-video-convert.mp4", v.ExtID, v.ID), "-c", "copy", "-start_number", "0", "-hls_time", "10", "-hls_list_size", "0", "-hls_segment_filename", fmt.Sprintf("/tmp/%s_%s-video_hls%s/%s_segment%s.ts", v.ExtID, v.ID, "%v", v.ExtID, "%d"), "-f", "hls", fmt.Sprintf("/tmp/%s_%s-video_hls%s/%s-video.m3u8", v.ExtID, v.ID, "%v", v.ExtID))

	videoConverLogFile, err := os.Open(fmt.Sprintf("/logs/%s_%s-video-convert.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error opening video convert logfile")
		return err
	}
	defer videoConverLogFile.Close()
	cmd.Stdout = videoConverLogFile
	cmd.Stderr = videoConverLogFile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running ffmpeg for vod video convert - hls")
		return err
	}

	log.Debug().Msgf("finished vod video convert - hls for %s", v.ExtID)
	return nil

}

func DownloadTwitchLiveVideo(v *ent.Vod, ch *ent.Channel) error {
	// Fetch config params
	liveStreamlinkParams := viper.GetString("parameters.streamlink_live")
	// Split supplied params into array
	splitParams := strings.Split(liveStreamlinkParams, ",")

	for i, arg := range splitParams {
		// Attempt to find access token
		if strings.Contains(arg, "twitch-api-header") {
			log.Debug().Msg("found twitch-api-header in streamlink args")
			// Extract access token
			accessToken := strings.Split(arg, "=OAuth ")[1]
			// Check access token
			err := twitch.CheckUserAccessToken(accessToken)
			if err != nil {
				log.Error().Err(err).Msg("error checking access token")
				// Remove arg from array if token is bad
				splitParams = append(splitParams[:i], splitParams[i+1:]...)
			}
		}
	}
	// Generate args for exec

	newArgs := []string{fmt.Sprintf("https://twitch.tv/%s", ch.Name), fmt.Sprintf("%s,best", v.Resolution)}
	newArgs = append(splitParams, newArgs...)
	newArgs = append(newArgs, "-o", fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID))

	log.Debug().Msgf("streamlink live args: %v", newArgs)
	// Execute streamlink
	cmd := osExec.Command("streamlink", newArgs...)

	videoLogfile, err := os.Create(fmt.Sprintf("/logs/%s_%s-video.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating video logfile")
		return err
	}
	defer videoLogfile.Close()
	cmd.Stdout = videoLogfile
	cmd.Stderr = videoLogfile

	if err := cmd.Run(); err != nil {
		// Streamlink will error when the stream is offline - do not log this as an error
		//log.Error().Err(err).Msg("error running streamlink for live video download")
		//return err
	}

	log.Debug().Msgf("finished downloading live video for %s", v.ExtID)
	return nil
}

func DownloadTwitchLiveChat(v *ent.Vod, ch *ent.Channel, q *ent.Queue, busC chan bool) error {

	log.Debug().Msg("sleeping 3 seconds for streamlink to start.")
	time.Sleep(3 * time.Second)

	log.Debug().Msg("setting chat start time")
	chatStartTime := time.Now()
	q.Update().SetChatStart(chatStartTime).SaveX(context.Background())

	log.Debug().Msgf("spawning chat_downloader for live stream %s", v.ID)

	cmd := osExec.Command("chat_downloader", fmt.Sprintf("https://twitch.tv/%s", ch.Name), "--output", fmt.Sprintf("/tmp/%s_%s-live-chat.json", v.ExtID, v.ID), "-q")

	chatLogfile, err := os.Create(fmt.Sprintf("/logs/%s_%s-chat.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating chat logfile")
		return err
	}
	defer chatLogfile.Close()
	cmd.Stdout = chatLogfile
	cmd.Stderr = chatLogfile
	// Append string to chatLogFile
	_, err = chatLogfile.WriteString("Chat downloader started. It it unlikely that you will see further output in this log.")
	if err != nil {
		log.Error().Err(err).Msg("error writing to chat logfile")
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("error starting chat_downloader for live chat download")
		return err
	}

	// When video download is complete kill chat download
	k := <-busC
	if k {
		log.Debug().Msg("streamlink detected the stream was down - killing chat_downloader")
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Error().Err(err).Msg("error killing chat_downloader")
			return err
		}
	}

	if err := cmd.Wait(); err != nil {
		log.Error().Err(err).Msg("error waiting for chat_downloader for live chat download")
		return err
	}

	log.Debug().Msgf("finished downloading live chat for %s", v.ExtID)
	return nil
}

func GetVideoDuration(path string) (int, error) {
	log.Debug().Msg("getting video duration")
	cmd := osExec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msg("error getting video duration")
		return 1, err
	}
	durOut := strings.TrimSpace(string(out))
	durFloat, err := strconv.ParseFloat(durOut, 8)
	if err != nil {
		log.Error().Err(err).Msg("error converting video duration")
		return 1, err
	}
	duration := int(durFloat)
	log.Debug().Msgf("video duration: %d", duration)
	return duration, nil
}

func GetFfprobeData(path string) (map[string]interface{}, error) {
	cmd := osExec.Command("ffprobe", "-hide_banner", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	out, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msgf("error getting ffprobe data for %s - err: %v", path, err)
		return nil, fmt.Errorf("error getting ffprobe data for %s - err: %w ", path, err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(out, &data); err != nil {
		log.Error().Err(err).Msg("error unmarshalling ffprobe data")
		return nil, err
	}
	return data, nil
}

func TwitchChatUpdate(v *ent.Vod) error {

	cmd := osExec.Command("TwitchDownloaderCLI", "chatupdate", "-i", fmt.Sprintf("/tmp/%s_%s-chat-convert.json", v.ExtID, v.ID), "--embed-missing", "-o", fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID))

	chatLogfile, err := os.Create(fmt.Sprintf("/logs/%s_%s-chat-convert.log", v.ExtID, v.ID))
	if err != nil {
		log.Error().Err(err).Msg("error creating chat convert logfile")
		return err
	}
	defer chatLogfile.Close()
	cmd.Stdout = chatLogfile
	cmd.Stderr = chatLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running TwitchDownloaderCLI for chat update")
		return err
	}

	log.Debug().Msgf("finished updating chat for %s", v.ExtID)
	return nil
}
