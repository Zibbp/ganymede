import { IconClock } from "@tabler/icons-react";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useTranslations } from "next-intl";
import { ButtonTooltip } from "@/components/videojs/ui/button-tooltip";
import { Button } from "@/components/videojs/ui/button";

const VideoPlayerAbsoluteTimeIcon = () => {
  const t = useTranslations("VideoComponents")
  const setShowAbsoluteTime = useSettingsStore((state) => state.setShowAbsoluteTime);
  const showAbsoluteTime = useSettingsStore((state) => state.showAbsoluteTime);

  const toggleAbsoluteTime = () => {
    setShowAbsoluteTime(!showAbsoluteTime);
  };
  return (
    <ButtonTooltip label={t('absoluteTimeIconTooltip')} side="top">
      <Button
        type="button"
        onClick={toggleAbsoluteTime}
        aria-label={t('absoluteTimeIconTooltip')}
        aria-pressed={showAbsoluteTime}
      >
        <IconClock className="media-button-icon" />
      </Button>
    </ButtonTooltip>
  );
}

export default VideoPlayerAbsoluteTimeIcon;
