import { Video } from "@/app/hooks/useVideos";
import classes from "./LoginRequired.module.css"
import { mediaSrc } from "@/app/util/util";
import { Center } from "@mantine/core";
import { IconLock } from "@tabler/icons-react";
import { useTranslations } from "next-intl";

interface Params {
  video: Video;
}

const VideoLoginRequired = ({ video }: Params) => {
  const t = useTranslations('VideoComponents')
  return (
    <div className={classes.container}>
      <div
        style={{
          backgroundImage: `url(${mediaSrc(video.web_thumbnail_path)})`
        }}
        className={classes.thumbnail}
      ></div>
      <div className={classes.textContainer}>
        <Center>
          <IconLock size={64} />
        </Center>
        <div className={classes.text}>
          {t('loginRequiredText')}
        </div>
      </div>
    </div>
  );
}

export default VideoLoginRequired;