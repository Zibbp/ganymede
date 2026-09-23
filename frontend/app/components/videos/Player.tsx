import '@videojs/react/video/skin.css';
import '@videojs/react/live-video/skin.css';
import { Video, VideoPlayer as VideoJsPlayer, VideoSkin, usePlayer as useVodPlayer } from '@videojs/react/video';
import { LiveVideoPlayer as LiveVideoJsPlayer, LiveVideoSkin, usePlayer as useLivePlayer } from '@videojs/react/live-video';
import { HlsJsVideo } from '@videojs/react/media/hlsjs-video';
import { I18nProvider } from '@videojs/react/i18n';
import { Video as VideoType, VideoType as GanymedeVideoType } from '@/app/hooks/useVideos';
import classes from "./Player.module.css"
import type { CSSProperties } from 'react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { env } from 'next-runtime-env';
import dayjs from 'dayjs';
import { useLocale } from 'next-intl';
import { escapeURL } from '@/app/util/util';
import { PlaybackStatus, useFetchPlaybackForVideo, useSetPlaybackProgressForVideo, useStartPlaybackForVideo, useUpdatePlaybackProgressForVideo } from '@/app/hooks/usePlayback';
import { useAxiosPrivate } from '@/app/hooks/useAxios';
import useAuthStore from '@/app/store/useAuthStore';
import { useSearchParams } from 'next/navigation';
import VideoEventBus from '@/app/util/VideoEventBus';
import VideoPlayerTheaterModeIcon from './PlayerTheaterModeIcon';
import useSettingsStore from '@/app/store/useSettingsStore';
import VideoPlayerHideChatIcon from './PlayerHideChatIcon';
import VideoPlayerAbsoluteTimeIcon from './PlayerAbsoluteTimeIcon';
import type { GanymedePlayerRef } from './ganymedePlayerRef';

interface Params {
  video: VideoType;
  ref: GanymedePlayerRef;
}

const AbsoluteTimeOverlay = ({ streamedAt, currentTime }: { streamedAt: string | Date; currentTime: number }) => {
  const flooredCurrentTime = Math.floor(currentTime);
  const absoluteTime = useMemo(
    () => dayjs(streamedAt).add(flooredCurrentTime, 'second'),
    [streamedAt, flooredCurrentTime],
  );

  return (
    <div className={classes.absoluteTimeOverlay}>
      <span className={classes.absoluteTimeText}>{absoluteTime.format('YYYY-MM-DD HH:mm:ss')}</span>
    </div>
  );
};

// Preset `usePlayer` hooks are typed, but TS resolves the selector state as
// `unknown` under the repo toolchain; narrow at the selection site.
const AbsoluteTimeDisplay = ({ streamedAt }: { streamedAt: string | Date }) => {
  const currentTime = useVodPlayer((s) => (s as unknown as { currentTime: number }).currentTime);
  const safeCurrentTime = typeof currentTime === 'number' && Number.isFinite(currentTime) ? currentTime : 0;
  return <AbsoluteTimeOverlay streamedAt={streamedAt} currentTime={safeCurrentTime} />;
};

const LiveAbsoluteTimeDisplay = ({ streamedAt }: { streamedAt: string | Date }) => {
  const currentTime = useLivePlayer((s) => (s as unknown as { currentTime: number }).currentTime);
  const safeCurrentTime = typeof currentTime === 'number' && Number.isFinite(currentTime) ? currentTime : 0;
  return <AbsoluteTimeOverlay streamedAt={streamedAt} currentTime={safeCurrentTime} />;
};

const PlayerOverlayButtons = () => {
  return (
    <div className={classes.overlayControls}>
      <VideoPlayerTheaterModeIcon />
      <VideoPlayerAbsoluteTimeIcon />
      <VideoPlayerHideChatIcon />
    </div>
  );
};

const VideoPlayer = ({ video, ref }: Params) => {
  const searchParams = useSearchParams()
  const locale = useLocale();

  const isLoggedIn = useAuthStore(state => state.isLoggedIn);

  const player = ref;
  const hasStartedPlayback = useRef(false);
  const hasInitializedPlaybackTime = useRef(false);
  const pendingResumeTime = useRef<number | null>(null);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const updatePlaybackProgressMutation = useUpdatePlaybackProgressForVideo()
  const setPlaybackProgressMutation = useSetPlaybackProgressForVideo()

  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);
  const showAbsoluteTime = useSettingsStore((state) => state.showAbsoluteTime);
  const autoplayVideo = useSettingsStore((state) => state.autoplayVideo);

  const axiosPrivate = useAxiosPrivate();
  // get playback data
  const { data: playbackData } = useFetchPlaybackForVideo(axiosPrivate, video.id, {
    refetchOnMount: "always",
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: false,
    enabled: (isLoggedIn)
  })

  // start playback
  const startPlaybackMutation = useStartPlaybackForVideo(axiosPrivate, video.id)
  useEffect(() => {
    if (isLoggedIn && !hasStartedPlayback.current) {
      startPlaybackMutation.mutate();
      hasStartedPlayback.current = true;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const isHls = video.processing || video.video_path.endsWith("m3u8");

  const videoSrc = useMemo(() => {
    // Allow for processing videos to be played via HLS from the temp directory if enabled
    if (video.processing) {
      return `${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.tmp_video_hls_path)}/${video.ext_id}-video.m3u8`;
    }
    return `${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.video_path)}`;
  }, [video.processing, video.tmp_video_hls_path, video.ext_id, video.video_path]);

  const videoPoster = useMemo(() => {
    if (video.thumbnail_path) {
      return `${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.thumbnail_path)}`;
    }
    return "";
  }, [video.thumbnail_path]);

  const chapterSrc = useMemo(() => {
    if (video.processing) return undefined;
    return `${(env('NEXT_PUBLIC_API_URL') ?? '')}/api/v1/chapter/video/${video.id}/webvtt`;
  }, [video.processing, video.id]);

  // thumbnails URL only when not processing
  const thumbnailsSrc = useMemo(() => {
    if (video.processing) return undefined;
    return `${(env('NEXT_PUBLIC_API_URL') ?? '')}/api/v1/vod/${video.id}/thumbnails/vtt`;
  }, [video.processing, video.id]);

  // Resolve the resume target once server playback or ?t= is known.
  useEffect(() => {
    if (hasInitializedPlaybackTime.current) return;
    if (pendingResumeTime.current !== null) return;
    if (playbackData && playbackData.time != null) {
      // Resume from server-side playback progress.
      pendingResumeTime.current = playbackData.time;
    } else {
      // Check if time is set in the url
      const time = searchParams.get("t");
      if (time !== null) {
        const parsed = parseInt(time, 10);
        if (Number.isFinite(parsed)) {
          pendingResumeTime.current = parsed;
        }
      }
    }
  }, [playbackData, searchParams]);

  const applyPendingResume = useCallback(() => {
    const el = player.current;
    if (!el) return;
    if (hasInitializedPlaybackTime.current) return;
    if (pendingResumeTime.current === null) return;
    try {
      el.currentTime = pendingResumeTime.current;
    } catch {
      return;
    }
    hasInitializedPlaybackTime.current = true;
  }, [player]);

  const applyStoredVolume = useCallback(() => {
    const el = player.current;
    if (!el) return;
    try {
      const localVolume = localStorage.getItem("ganymede-volume");
      if (localVolume !== null) {
        const parsed = parseFloat(localVolume);
        if (Number.isFinite(parsed)) {
          el.volume = Math.min(1, Math.max(0, parsed));
        }
      }
    } catch {
      // localStorage may be unavailable; keep element default.
    }
  }, [player]);

  const handleVolumeChange = useCallback(() => {
    const el = player.current;
    if (!el) return;
    const volume = el.volume;
    if (volume !== 1) {
      try {
        localStorage.setItem("ganymede-volume", volume.toString());
      } catch {
        // Ignore storage errors.
      }
    }
  }, [player]);

  const handleMediaReady = useCallback(() => {
    applyStoredVolume();
    applyPendingResume();
  }, [applyStoredVolume, applyPendingResume]);

  // Retry resume once the element exists and the target is known.
  useEffect(() => {
    if (!mounted) return;
    if (hasInitializedPlaybackTime.current) return;
    if (pendingResumeTime.current === null) return;
    applyPendingResume();
  }, [mounted, playbackData, videoSrc, applyPendingResume]);


  // Playback progress reporting
  useEffect(() => {
    if (!isLoggedIn) return;
    const playbackInerval = setInterval(async () => {
      if (player.current == null) return;
      if (player.current.paused) return;

      const playerTimeInt = Math.floor(player.current.currentTime)
      if (playerTimeInt == 0) return;


      updatePlaybackProgressMutation.mutate({
        axiosPrivate: axiosPrivate,
        videoId: video.id,
        time: playerTimeInt
      })

      // mark video as finished if over duration threshold
      if (!video.processing && (playerTimeInt / video.duration >= 0.98)) {
        setPlaybackProgressMutation.mutate({
          axiosPrivate: axiosPrivate,
          videoId: video.id,
          status: PlaybackStatus.Finished
        })

        // remove interval
        clearInterval(playbackInerval)
      }
    }, 10000);
    return () => clearInterval(playbackInerval);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Fast tick for chat player - set player information in bus
  useEffect(() => {
    const ticketInterval = setInterval(() => {
      if (player.current == null) return;

      let time = player.current.currentTime
      // Clip chats are offset with the position of the clip in the VOD
      // Append the offset to the current player time to account for this
      if (video.type == GanymedeVideoType.Clip && video.clip_vod_offset) {
        time = time + video.clip_vod_offset
      };

      VideoEventBus.setData({
        isPaused: player.current.paused,
        isPlaying: !player.current.paused,
        time: time
      })
    }, 100);
    return () => {
      clearInterval(ticketInterval);
    };
  }, [player, video.clip_vod_offset, video.type]);

  const skinStyle = {
    '--media-accent-color': 'var(--mantine-color-violet-6)',
    '--media-object-fit': 'contain',
    '--media-border-radius': '0',
    '--media-video-border-radius': '0',
  } as CSSProperties;

  // Avoid SSR mismatch: Video.js media engines need the browser.
  if (!mounted) {
    return (
      <div
        className={
          videoTheaterMode
            ? classes.mediaPlayerTheaterMode
            : classes.mediaPlayer
        }
      />
    );
  }

  const mediaProps = {
    ref: player,
    src: videoSrc,
    crossOrigin: "anonymous" as const,
    playsInline: true,
    autoPlay: autoplayVideo,
    preload: "auto" as const,
    onLoadedMetadata: handleMediaReady,
    onCanPlay: handleMediaReady,
    onVolumeChange: handleVolumeChange,
  };

  return (
    <div className={classes.playerWrapper}>
      {video.processing ? (
        <LiveVideoJsPlayer title={video.title} poster={videoPoster}>
          <I18nProvider locale={locale}>
            <LiveVideoSkin
              className={
                videoTheaterMode
                  ? classes.mediaPlayerTheaterMode
                  : classes.mediaPlayer
              }
              style={skinStyle}
            >
              <HlsJsVideo {...mediaProps} />
              {showAbsoluteTime && <LiveAbsoluteTimeDisplay streamedAt={video.streamed_at} />}
              <PlayerOverlayButtons />
            </LiveVideoSkin>
          </I18nProvider>
        </LiveVideoJsPlayer>
      ) : (
        <VideoJsPlayer title={video.title} poster={videoPoster}>
          <I18nProvider locale={locale}>
            <VideoSkin
              className={
                videoTheaterMode
                  ? classes.mediaPlayerTheaterMode
                  : classes.mediaPlayer
              }
              style={skinStyle}
            >
              {isHls ? (
                <HlsJsVideo {...mediaProps}>
                  {chapterSrc && (
                    <track kind="chapters" src={chapterSrc} default />
                  )}
                  {thumbnailsSrc && (
                    <track kind="metadata" label="thumbnails" src={thumbnailsSrc} default />
                  )}
                </HlsJsVideo>
              ) : (
                <Video {...mediaProps}>
                  {chapterSrc && (
                    <track kind="chapters" src={chapterSrc} default />
                  )}
                  {thumbnailsSrc && (
                    <track kind="metadata" label="thumbnails" src={thumbnailsSrc} default />
                  )}
                </Video>
              )}
              {showAbsoluteTime && <AbsoluteTimeDisplay streamedAt={video.streamed_at} />}
              <PlayerOverlayButtons />
            </VideoSkin>
          </I18nProvider>
        </VideoJsPlayer>
      )}
    </div>
  );
}

export default VideoPlayer;
