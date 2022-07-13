package exec

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/zibbp/ganymede/ent"
	"os"
	osExec "os/exec"
	"strings"
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
