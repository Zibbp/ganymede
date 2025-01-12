import { MediaPlayer, MediaPlayerInstance, MediaProvider, MediaProviderInstance, Poster, Track } from "@vidstack/react";
import { defaultLayoutIcons, DefaultVideoLayout } from '@vidstack/react/player/layouts/default';
import { useEffect, useRef, useState } from "react";
import '@vidstack/react/player/styles/default/theme.css';
import '@vidstack/react/player/styles/default/layouts/video.css';
import classes from "./SyncedVideoPlayer.module.css"
import { env } from "next-runtime-env";

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
  const player = useRef<MediaPlayerInstance>(null)
  const mediaProvider = useRef<MediaProviderInstance>(null)
  const [canPlay, setCanPlay] = useState(false)

  useEffect(() => {
    const currentPlayer = player.current
    if (!currentPlayer || !canPlay) return;
    (async () => {
      if (playing) {
        currentPlayer.currentTime = time;
        await (new Promise<void>(resolve => setTimeout(resolve, 1)));
        await currentPlayer.play();
      } else {
        await (new Promise<void>(resolve => setTimeout(resolve, 1)));
        await currentPlayer.pause();
      }
    })();
  }, [playing, canPlay, time])

  useEffect(() => {
    if (!player.current) return;
    player.current.muted = muted;
  }, [muted])

  useEffect(() => {
    if (!player.current || Math.abs(player.current.currentTime - time) < 0.2) return;
    player.current.currentTime = time;
  }, [time])

  return (
    <MediaPlayer
      className={classes.mediaPlayer}
      src={src}
      ref={player}
      aspect-ratio={16 / 9}
      crossOrigin
      onCanPlay={() => setCanPlay(true)}
      playsInline
      muted={muted}
    >
      <MediaProvider ref={mediaProvider}>
        <Poster className={`${classes.ganymedePoster} vds-poster`} src={poster} alt={title} />
        <Track
          src={`${(env('NEXT_PUBLIC_API_URL') ?? '')}/api/v1/chapter/video/${vodId}/webvtt`}
          kind="chapters"
          default={true}
        />
      </MediaProvider>

      <DefaultVideoLayout icons={defaultLayoutIcons} />
    </MediaPlayer>
  )
};

export default SyncedVideoPlayer;