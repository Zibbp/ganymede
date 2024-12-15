import { Video } from "@/app/hooks/useVideos";
import classes from "./LoginRequired.module.css"
import { env } from "next-runtime-env";
import { escapeURL } from "@/app/util/util";
import { Center } from "@mantine/core";
import { IconLock } from "@tabler/icons-react";

interface Params {
  video: Video;
}

const VideoLoginRequired = ({ video }: Params) => {
  return (
    <div className={classes.container}>
      <div
        style={{
          backgroundImage: `url(${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(
            video.web_thumbnail_path
          )})`
        }}
        className={classes.thumbnail}
      ></div>
      <div className={classes.textContainer}>
        <Center>
          <IconLock size={64} />
        </Center>
        <div className={classes.text}>
          You must be logged in to view this video
        </div>
      </div>
    </div>
  );
}

export default VideoLoginRequired;