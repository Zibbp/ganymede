"use client"
import { useFetchVideo } from "@/app/hooks/useVideos";
import React from "react";
import classes from "./VideoPage.module.css"
import { Box } from "@mantine/core";
import VideoPlayer from "@/app/components/videos/Player";
import VideoTitleBar from "@/app/components/videos/TitleBar";
import ChatPlayer from "@/app/components/videos/ChatPlayer";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useFullscreen } from "@mantine/hooks";

interface Params {
  id: string;
}

const VideoPage = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);

  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { ref, toggle, fullscreen } = useFullscreen();


  const { data, isPending, isError } = useFetchVideo({ id, with_channel: true, with_chapters: true, with_muted_segments: true })

  if (isPending) {
    return <GanymedeLoadingText message="Loading Video" />
  }
  if (isError) {
    return <div>Error loading video</div>
  }

  return (
    <div>
      {/* Player and chat section */}
      <Box className={classes.container}>
        {/* Player */}
        <div className={!data.chat_path ? classes.leftColumnNoChat : classes.leftColumn}>
          <div className={
            videoTheaterMode || fullscreen ? classes.videoPlayerTheaterMode : classes.videoPlayer
          }>
            <VideoPlayer video={data} />
          </div>
        </div>


        {/* Chat */}
        {data.chat_path && (
          <div className={classes.rightColumn} style={{ height: "auto", maxHeight: "auto" }}>
            <div className={
              videoTheaterMode || fullscreen
                ? classes.chatColumnTheaterMode
                : classes.chatColumn
            }
            >
              <ChatPlayer video={data} />
            </div>
          </div>
        )}

      </Box>

      {/* Title bar */}
      {!videoTheaterMode && <VideoTitleBar video={data} />}
    </div>
  );
}

export default VideoPage;