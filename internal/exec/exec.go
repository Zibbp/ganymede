package exec

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"os"
	osExec "os/exec"
	"strconv"
	"strings"
	"time"
)

func DownloadTwitchVodVideo(v *ent.Vod) error {

	cmd := osExec.Command("streamlink", fmt.Sprintf("https://twitch.tv/videos/%s", v.ExtID), fmt.Sprintf("%s,best", v.Resolution), "--force-progress", "--force", "-o", fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID))

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
	cmd := osExec.Command("TwitchDownloaderCLI", "-m", "ChatDownload", "--id", v.ExtID, "--embed-emotes", "-o", fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID))

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

func RenderTwitchVodChat(v *ent.Vod) error {
	// Fetch config params
	chatRenderParams := viper.GetString("parameters.chat_render")
	// Split supplied params into array
	arr := strings.Fields(chatRenderParams)
	// Generate args for exec
	argArr := []string{"-m", "ChatRender", "-i", fmt.Sprintf("/tmp/%s_%s-chat.json", v.ExtID, v.ID)}
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
		return err
	}
	defer chatRenderLogfile.Close()
	cmd.Stdout = chatRenderLogfile
	cmd.Stderr = chatRenderLogfile

	if err := cmd.Run(); err != nil {
		log.Error().Err(err).Msg("error running TwitchDownloaderCLI for vod chat render")
		return err
	}

	log.Debug().Msgf("finished vod chat render for %s", v.ExtID)
	return nil
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

func DownloadTwitchLiveVideo(v *ent.Vod, ch *ent.Channel) error {

	cmd := osExec.Command("streamlink", fmt.Sprintf("https://twitch.tv/%s", ch.Name), fmt.Sprintf("%s,best", v.Resolution), "--force-progress", "--force", "--twitch-low-latency", "--twitch-disable-ads", "--twitch-disable-hosting", "-o", fmt.Sprintf("/tmp/%s_%s-video.mp4", v.ExtID, v.ID))

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

func DownloadTwitchLiveChat(v *ent.Vod, ch *ent.Channel, busC chan bool) error {

	log.Debug().Msg("sleeping 3 seconds for streamlink to start.")
	time.Sleep(3 * time.Second)
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
