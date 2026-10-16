import { IconArrowBarLeft, IconArrowBarRight, } from "@tabler/icons-react";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useTranslations } from "next-intl";
import { ButtonTooltip } from "@/components/videojs/ui/button-tooltip";
import { Button } from "@/components/videojs/ui/button";

const VideoPlayerHideChatIcon = () => {
  const t = useTranslations("VideoComponents")
  const { setHideChat } = useSettingsStore()
  const hideChat = useSettingsStore((state) => state.hideChat);

  const toggleHideChat = () => {
    setHideChat(!hideChat);
  };
  const label = hideChat ? t('showChatIconTooltip') : t('hideChatIconTooltip');

  return (
    <ButtonTooltip label={label} side="top">
      <Button
        type="button"
        onClick={toggleHideChat}
        aria-label={label}
        aria-pressed={hideChat}
      >
        {hideChat ? (
          <IconArrowBarLeft className="media-button-icon" />
        ) : (
          <IconArrowBarRight className="media-button-icon" />
        )}
      </Button>
    </ButtonTooltip>
  );
}

export default VideoPlayerHideChatIcon;
