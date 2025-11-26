import { ActionIcon, Tooltip } from "@mantine/core";
import { IconArrowBarLeft, IconArrowBarRight, } from "@tabler/icons-react";
import classes from "./PlayerHideChatIcon.module.css"
import useSettingsStore from "@/app/store/useSettingsStore";
import { useTranslations } from "next-intl";

const VideoPlayerHideChatIcon = () => {
  const t = useTranslations("VideoComponents")
  const { setHideChat } = useSettingsStore()
  const hideChat = useSettingsStore((state) => state.hideChat);

  const toggleHideChat = () => {
    setHideChat(!hideChat);
  };
  return (
    <div className={classes.hideIcon}>
      <Tooltip label={hideChat ? t('showChatIconTooltip') : t('hideChatIconTooltip')} position="bottom">
        <ActionIcon
          size="xl"
          variant="transparent"
          onClick={toggleHideChat}
          onTouchStart={toggleHideChat}
          className={classes.customFullScreenButton}
        >
          {hideChat ? (
            <IconArrowBarLeft size={24} />
          ) : (
            <IconArrowBarRight size={24} />
          )}
        </ActionIcon>
      </Tooltip>
    </div>
  );
}

export default VideoPlayerHideChatIcon;