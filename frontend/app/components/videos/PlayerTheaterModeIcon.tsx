import { IconMaximize } from "@tabler/icons-react";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useTranslations } from "next-intl";
import { ButtonTooltip } from "@/components/videojs/ui/button-tooltip";
import { Button } from "@/components/videojs/ui/button";

const VideoPlayerTheaterModeIcon = () => {
  const t = useTranslations("VideoComponents")
  const { setVideoTheaterMode } = useSettingsStore()
  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);

  const toggleTheaterMode = () => {
    setVideoTheaterMode(!videoTheaterMode)
  };
  return (
    <ButtonTooltip label={t('theaterModeIconTooltip')} side="top">
      <Button
        type="button"
        onClick={toggleTheaterMode}
        aria-label={t('theaterModeIconTooltip')}
        aria-pressed={videoTheaterMode}
      >
        <IconMaximize className="media-button-icon" />
      </Button>
    </ButtonTooltip>
  );
}

export default VideoPlayerTheaterModeIcon;
