import { ActionIcon, Tooltip } from "@mantine/core";
import { IconMaximize } from "@tabler/icons-react";
import classes from "./PlayerTheaterModeIcon.module.css"
import useSettingsStore from "@/app/store/useSettingsStore";

const VideoPlayerTheaterModeIcon = () => {
  const { setVideoTheaterMode } = useSettingsStore()
  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);

  const toggleTheaterMode = () => {
    setVideoTheaterMode(!videoTheaterMode)
  };
  return (
    <div className={classes.theaterIcon}>
      <Tooltip label="Theater Mode" position="bottom">
        <ActionIcon
          size="xl"
          variant="transparent"
          onClick={toggleTheaterMode}
          onTouchStart={toggleTheaterMode}
          className={classes.customFullScreenButton}
        >
          <IconMaximize size="1.7rem" />
        </ActionIcon>
      </Tooltip>
    </div>
  );
}

export default VideoPlayerTheaterModeIcon;