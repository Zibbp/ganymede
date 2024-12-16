"use client"
import { useFetchVideo, useGetVideoClips } from "@/app/hooks/useVideos";
import React, { useEffect } from "react";
import classes from "./VideoPage.module.css"
import { Box, Container } from "@mantine/core";
import VideoPlayer from "@/app/components/videos/Player";
import VideoTitleBar from "@/app/components/videos/TitleBar";
import ChatPlayer from "@/app/components/videos/ChatPlayer";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useFullscreen } from "@mantine/hooks";
import { env } from "next-runtime-env";
import VideoLoginRequired from "@/app/components/videos/LoginRequired";
import useAuthStore from "@/app/store/useAuthStore";
import VideoPageClips from "@/app/components/videos/VideoClips";

interface Params {
  id: string;
}

const VideoPage = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);
  const { isLoggedIn } = useAuthStore()

  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { ref, toggle, fullscreen } = useFullscreen();

  const { data, isPending, isError } = useFetchVideo({ id, with_channel: true, with_chapters: true, with_muted_segments: true })

  // need to fetch clips here to dynamically render the clips section
  const { data: videoClips, isPending: videoClipsPending, isError: videoClipsError } = useGetVideoClips(id)


  useEffect(() => {
    document.title = `${data?.title}`;
  }, [data?.title]);

  // check if login is required
  const isLoginRequired = () => {
    if (
      env('NEXT_PUBLIC_REQUIRE_LOGIN') == "true" && !isLoggedIn
    ) {
      return true
    }
    return false
  }

  if (isPending) {
    return <GanymedeLoadingText message="Loading Video" />
  }
  if (isError) {
    return <div>Error loading video</div>
  }

  if (isLoginRequired()) {
    return <VideoLoginRequired video={data} />
  }

  return (
    <div>
      {/* Hide navbar. I don't like doing this but the navbar ruins the experience */}
      <style jsx>{`
        :global(html)::-webkit-scrollbar {
          display: none;
        }
        :global(html) {
          -ms-overflow-style: none; /* IE and Edge */
          scrollbar-width: none; /* Firefox */
        }
      `}</style>

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


      {/* Video clips */}
      <Container size="7xl" fluid={true} >
        {videoClipsError && (
          <div>Error loading clips</div>
        )}
        {((!videoClipsPending) && (videoClips && videoClips.length > 0)) && (
          <VideoPageClips clips={videoClips} />
        )}
      </Container>

    </div>
  );
}

export default VideoPage;