import '@videojs/react/video/skin.css';
import { Video, VideoPlayer as VideoJsPlayer, VideoSkin } from '@videojs/react/video';
import { HlsJsVideo } from '@videojs/react/media/hlsjs-video';
import { I18nProvider } from '@videojs/react/i18n';
import { useEffect, useRef, useState } from "react";
import type { CSSProperties } from "react";
import classes from "./SyncedVideoPlayer.module.css"
import { env } from "next-runtime-env";
import { useLocale } from 'next-intl';

export type SyncedVideoPlayerProps = {
  src: string;
  vodId: string;
  title: string;
  poster: string;
  time: number;
  playing: boolean;
  muted: boolean;
}

const SyncedVideoPlayer = ({ src, vodId, title, poster, time, playing, muted }: SyncedVideoPlayerProps) => {
  const locale = useLocale();
  const videoEl = useRef<HTMLVideoElement>(null)
  const [canPlay, setCanPlay] = useState(false)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    const current = videoEl.current
    if (!current || !canPlay) return;
    (async () => {
      if (playing) {
        try {
          current.currentTime = time;
        } catch {
          // Element not ready yet; play will still start from current position.
        }
        await (new Promise<void>(resolve => setTimeout(resolve, 1)));
        try {
          await current.play();
        } catch {
          // Autoplay with sound is blocked; tiles are muted by default.
        }
      } else {
        await (new Promise<void>(resolve => setTimeout(resolve, 1)));
        try {
          current.pause();
        } catch {
          // Ignore pause errors during teardown.
        }
      }
    })();
  }, [playing, canPlay /* `time` should not be part of dependencies, it already has its own effect */])

  useEffect(() => {
    if (!videoEl.current) return;
    try {
      videoEl.current.muted = muted;
    } catch {
      // Ignore mute errors during teardown.
    }
  }, [muted])

  useEffect(() => {
    if (!videoEl.current || Math.abs(videoEl.current.currentTime - time) < 0.2) return;
    try {
      videoEl.current.currentTime = time;
    } catch {
      // Element not ready yet; the playing effect will seek on play.
    }
  }, [time])

  if (!mounted) {
    return <div className={classes.mediaPlayer} />;
  }

  const isHls = src.endsWith(".m3u8");
  const chapterSrc = `${(env('NEXT_PUBLIC_API_URL') ?? '')}/api/v1/chapter/video/${vodId}/webvtt`;

  const mediaProps = {
    ref: videoEl,
    src,
    crossOrigin: "anonymous" as const,
    onCanPlay: () => setCanPlay(true),
    playsInline: true,
    muted,
    preload: "auto" as const,
  };

  const skinStyle = {
    '--media-border-radius': '0',
    '--media-video-border-radius': '0',
  } as CSSProperties;

  return (
    <VideoJsPlayer title={title} poster={poster}>
      <I18nProvider locale={locale}>
        <VideoSkin className={classes.mediaPlayer} style={skinStyle}>
          {isHls ? (
            <HlsJsVideo {...mediaProps}>
              <track kind="chapters" src={chapterSrc} default />
            </HlsJsVideo>
          ) : (
            <Video {...mediaProps}>
              <track kind="chapters" src={chapterSrc} default />
            </Video>
          )}
        </VideoSkin>
      </I18nProvider>
    </VideoJsPlayer>
  )
};

export default SyncedVideoPlayer;
