import '@vidstack/react/player/styles/default/theme.css';
import '@vidstack/react/player/styles/default/layouts/video.css';
import { MediaPlayer, MediaPlayerInstance, MediaProvider, MediaSrc, Poster, Track, VideoMimeType, useMediaState } from '@vidstack/react';
import { defaultLayoutIcons, DefaultVideoLayout } from '@vidstack/react/player/layouts/default';
import { Video, VideoType } from '@/app/hooks/useVideos';
import classes from "./Player.module.css"
import { RefObject, useEffect, useMemo, useRef, useState } from 'react';
import { env } from 'next-runtime-env';
import dayjs from 'dayjs';
import { escapeURL, playlistPathForVideo } from '@/app/util/util';
import { usePlaylistSource } from '@/app/hooks/usePlaylistSource';
import { PlaybackStatus, useFetchPlaybackForVideo, useSetPlaybackProgressForVideo, useStartPlaybackForVideo, useUpdatePlaybackProgressForVideo } from '@/app/hooks/usePlayback';
import { useAxiosPrivate } from '@/app/hooks/useAxios';
import useAuthStore from '@/app/store/useAuthStore';
import { useSearchParams } from 'next/navigation';
import VideoEventBus from '@/app/util/VideoEventBus';
import VideoPlayerTheaterModeIcon from './PlayerTheaterModeIcon';
import useSettingsStore from '@/app/store/useSettingsStore';
import VideoPlayerHideChatIcon from './PlayerHideChatIcon';
import VideoPlayerAbsoluteTimeIcon from './PlayerAbsoluteTimeIcon';

interface Params {
  video: Video;
  ref: RefObject<MediaPlayerInstance | null>;
}

const AbsoluteTimeDisplay = ({ streamedAt }: { streamedAt: string | Date }) => {
  const currentTime = useMediaState('currentTime');
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

const VideoPlayer = ({ video, ref }: Params) => {
  const searchParams = useSearchParams()

  const isLoggedIn = useAuthStore(state => state.isLoggedIn);

  const player = ref;
  const [videoSource, setVideoSource] = useState<MediaSrc>();
  const [videoPoster, setVideoPoster] = useState<string>("");

  const hasStartedPlayback = useRef(false);
  const hasInitializedPlaybackTime = useRef(false);
  // Where to pick up after swapping the source, so giving up on a playlist
  // mid-playback does not throw the viewer back to the start.
  const resumeAfterSourceChange = useRef<number | null>(null);

  const [playerVolume, setPlayerVolume] = useState(1);

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

  const { playlistUrl, dismissPlaylist } = usePlaylistSource(video.video_path, video.processing)

  useEffect(() => {
    if (video.thumbnail_path) {
      setVideoPoster(`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.thumbnail_path)}`)
    }
  }, [video.thumbnail_path])

  useEffect(() => {
    if (!player) return
    // Waiting for the playlist check keeps the player from loading the video
    // file and switching away from it a moment later.
    if (playlistUrl === undefined) return

    // A video that is itself a playlist has none beside it, which is the same
    // test playlistPathForVideo makes.
    const videoIsPlaylist = playlistPathForVideo(video.video_path) == null
    let videoType: VideoMimeType = "video/mp4"
    if (videoIsPlaylist) {
      videoType = "video/object";
    }

    // Allow for processing videos to be played via HLS from the temp directory if enabled
    if (video.processing) {
      setVideoSource({
        src: `${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.tmp_video_hls_path)}/${video.ext_id}-video.m3u8`,
        type: "application/x-mpegurl"
      })
    } else if (playlistUrl) {
      setVideoSource({ src: playlistUrl, type: "application/x-mpegurl" })
    } else {
      setVideoSource({
        src: `${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.video_path)}`,
        type: videoType
      })
    }

    // todo: captions?

    const localVolume = localStorage.getItem("ganymede-volume")
    if (localVolume) {
      setPlayerVolume(parseFloat(localVolume))
    }

    player.current?.subscribe(({ volume }) => {
      if (volume != 1) {
        localStorage.setItem("ganymede-volume", volume.toString());
      }
    });

    if (!hasInitializedPlaybackTime.current) {
      // A shared link names the moment it wants to show, so it wins over
      // whatever this viewer watched before. The order matters now that the
      // source arrives asynchronously: this used to run before the saved
      // position had been fetched, so the link won by timing rather than by
      // rule, and would otherwise start losing that race.
      const time = searchParams.get("t");
      if (time !== null) {
        player.current!.currentTime = parseInt(time);
        hasInitializedPlaybackTime.current = true
      } else if (playbackData && playbackData.time != null) {
        // Resume from server-side playback progress.
        player.current!.currentTime = playbackData.time
        hasInitializedPlaybackTime.current = true
      }
    }

  }, [player, video, playbackData, searchParams, playlistUrl])


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

      let time = player.current.state.currentTime
      // Clip chats are offset with the position of the clip in the VOD
      // Append the offset to the current player time to account for this
      if (video.type == VideoType.Clip && video.clip_vod_offset) {
        time = time + video.clip_vod_offset
      };

      VideoEventBus.setData({
        isPaused: player.current.state.paused,
        isPlaying: player.current.state.playing,
        time: time
      })
    }, 100);
    return () => {
      clearInterval(ticketInterval);
    };
  }, [player, video.clip_vod_offset, video.type]);

  // thumbnails URL only when not processing
  const thumbnails = !video.processing
    ? `${(env('NEXT_PUBLIC_API_URL') ?? '')}/api/v1/vod/${video.id}/thumbnails/vtt`
    : undefined
  return (
    <MediaPlayer
      ref={player}
      className={
        videoTheaterMode
          ? classes.mediaPlayerTheaterMode
          : classes.mediaPlayer
      }
      src={videoSource}
      aspect-ratio={16 / 9}
      crossOrigin={true}
      playsInline={true}
      load="eager"
      posterLoad="eager"
      volume={playerVolume}
      autoPlay={autoplayVideo}
      onError={() => {
        if (!playlistUrl) return
        // Changing the source resets the clock, so where to pick up is decided
        // here and applied once the video file is ready: the position reached
        // if playback had begun, otherwise the start this view was opened at.
        // Deciding it now rather than re-running the source effect keeps the
        // dying source from swallowing the seek.
        const reached = player.current?.currentTime ?? 0
        const openedAt = Number(searchParams.get("t") ?? playbackData?.time ?? 0)
        const startAt = reached > 0.5 ? reached : openedAt
        resumeAfterSourceChange.current = startAt > 0 ? startAt : null
        dismissPlaylist()
      }}
      onCanPlay={() => {
        if (resumeAfterSourceChange.current == null) return
        player.current!.currentTime = resumeAfterSourceChange.current
        resumeAfterSourceChange.current = null
      }}
    >
      {showAbsoluteTime && <AbsoluteTimeDisplay streamedAt={video.streamed_at} />}
      <MediaProvider>
        <Poster className={`${classes.mediaPlayerPoster} vds-poster`} src={videoPoster} alt={video.title} />
        {!video.processing && (
          <Track
            src={`${(env('NEXT_PUBLIC_API_URL') ?? '')}/api/v1/chapter/video/${video.id}/webvtt`}
            kind="chapters"
            default={true}
          />
        )}
      </MediaProvider>
      <DefaultVideoLayout icons={defaultLayoutIcons} noScrubGesture={false}
        slots={{
          beforeFullscreenButton: <VideoPlayerTheaterModeIcon />,
          afterFullscreenButton: (
            <>
              <VideoPlayerAbsoluteTimeIcon />
              <VideoPlayerHideChatIcon />
            </>
          )
        }}
        thumbnails={thumbnails}
      />
    </MediaPlayer>
  );
}

export default VideoPlayer;
