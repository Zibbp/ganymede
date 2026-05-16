"use client"
import { useFetchVideo, useGetVideoClips, VideoType } from "@/app/hooks/useVideos";
import React, { useEffect, useRef } from "react";
import classes from "./VideoPage.module.css"
import { Box, Container, useMantineTheme } from "@mantine/core";
import VideoPlayer from "@/app/components/videos/Player";
import VideoTitleBar from "@/app/components/videos/TitleBar";
import ChatPlayer from "@/app/components/videos/ChatPlayer";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useFullscreen, useMediaQuery } from "@mantine/hooks";
import { env } from "next-runtime-env";
import VideoLoginRequired from "@/app/components/videos/LoginRequired";
import useAuthStore from "@/app/store/useAuthStore";
import VideoPageClips from "@/app/components/videos/VideoClips";
import VideoChatHistogram from "@/app/components/videos/ChatHistogram";
import { MediaPlayerInstance } from "@vidstack/react";
import { useTranslations } from "next-intl";

interface Params {
  id: string;
}

const VideoPage = ({ params }: { params: Promise<Params> }) => {
  const theme = useMantineTheme()
  const { id } = React.use(params);
  const { isLoggedIn } = useAuthStore()
  const player = useRef<MediaPlayerInstance>(null);
  const isMobile = useMediaQuery(`(max-width: ${theme.breakpoints.sm})`);

  const t = useTranslations("VideoPage");

  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);
  const hideChat = useSettingsStore((state) => state.hideChat);
  const showChatHistogram = useSettingsStore((state) => state.showChatHistogram);
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
    return <GanymedeLoadingText message={t('loading')} />
  }
  if (isError) {
    return <div>{t('error')}</div>
  }


  if (isLoginRequired()) {
    return <VideoLoginRequired video={data} />
  }

  return (
    <div>
      {/* Player and chat section — single tree on both layouts so VideoPlayer/ChatPlayer instances persist across the breakpoint flip */}
      <Box className={isMobile ? classes.containerMobile : classes.container}>
        {/* Player */}
        <div className={
          isMobile
            ? undefined
            : (!data.chat_path ? classes.leftColumnNoChat : classes.leftColumn)
        }>
          <div className={
            isMobile
              ? undefined
              : (videoTheaterMode || fullscreen ? classes.videoPlayerTheaterMode : classes.videoPlayer)
          }>
            <VideoPlayer video={data} ref={player} />
          </div>
        </div>

        {/* Chat */}
        {data.chat_path && !hideChat && !data.processing && (
          <div
            className={isMobile ? classes.chatColumnMobile : classes.rightColumn}
            style={isMobile ? undefined : { height: "auto", maxHeight: "auto" }}
          >
            <div
              className={
                isMobile
                  ? undefined
                  : (videoTheaterMode || fullscreen ? classes.chatColumnTheaterMode : classes.chatColumn)
              }
              style={isMobile ? { height: "100%" } : undefined}
            >
              <ChatPlayer video={data} playerRef={player} />
            </div>
          </div>
        )}
      </Box>

      {/* Title bar */}
      {!videoTheaterMode && <VideoTitleBar video={data} />}

      {/* Desktop-only sections render after the player/chat block so toggling them doesn't shift player position */}
      {!isMobile && !data.processing && (
        <Container size="7xl" fluid={true} >
          {videoClipsError && (
            <div>Error loading clips</div>
          )}
          {((!videoClipsPending) && (videoClips && videoClips.length > 0)) && (
            <VideoPageClips clips={videoClips} />
          )}
        </Container>
      )}

      {(!isMobile && data.chat_path && (data.type != VideoType.Clip) && showChatHistogram && !data.processing) && (
        <Container size="7xl" fluid={true} >
          <VideoChatHistogram videoId={data.id} playerRef={player} />
        </Container>
      )}

      {!isMobile && (
        <style jsx>{`
          :global(html)::-webkit-scrollbar {
            display: none;
          }
          :global(html) {
            -ms-overflow-style: none; /* IE and Edge */
            scrollbar-width: none; /* Firefox */
          }
        `}</style>
      )}
    </div>
  );
}

export default VideoPage;
