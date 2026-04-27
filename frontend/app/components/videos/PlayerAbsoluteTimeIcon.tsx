import { ActionIcon, Tooltip } from "@mantine/core";
import { IconClock } from "@tabler/icons-react";
import classes from "./PlayerAbsoluteTimeIcon.module.css"
import useSettingsStore from "@/app/store/useSettingsStore";
import { useTranslations } from "next-intl";

const VideoPlayerAbsoluteTimeIcon = () => {
  const t = useTranslations("VideoComponents")
  const setShowAbsoluteTime = useSettingsStore((state) => state.setShowAbsoluteTime);
  const showAbsoluteTime = useSettingsStore((state) => state.showAbsoluteTime);

  const toggleAbsoluteTime = () => {
    setShowAbsoluteTime(!showAbsoluteTime);
  };
  return (
    <div className={classes.absoluteTimeIcon}>
      <Tooltip label={t('absoluteTimeIconTooltip')} position="bottom">
        <ActionIcon
          size="xl"
          variant="transparent"
          onClick={toggleAbsoluteTime}
          className={classes.customFullScreenButton}
        >
          <IconClock size="1.7rem" />
        </ActionIcon>
      </Tooltip>
    </div>
  );
}

export default VideoPlayerAbsoluteTimeIcon;
