import '@vidstack/react/player/styles/default/theme.css';
import '@vidstack/react/player/styles/default/layouts/video.css';
import { MediaPlayer, MediaPlayerInstance, MediaProvider, MediaSrc, Poster, Track, VideoMimeType } from '@vidstack/react';
import { defaultLayoutIcons, DefaultVideoLayout } from '@vidstack/react/player/layouts/default';
import { Video, VideoType } from '@/app/hooks/useVideos';
import classes from "./Player.module.css"
import { RefObject, useEffect, useRef, useState } from 'react';
import { env } from 'next-runtime-env';
import { escapeURL } from '@/app/util/util';
import { PlaybackStatus, useFetchPlaybackForVideo, useSetPlaybackProgressForVideo, useStartPlaybackForVideo, useUpdatePlaybackProgressForVideo } from '@/app/hooks/usePlayback';
import { useAxiosPrivate } from '@/app/hooks/useAxios';
import useAuthStore from '@/app/store/useAuthStore';
import { useSearchParams } from 'next/navigation';
import VideoEventBus from '@/app/util/VideoEventBus';
import VideoPlayerTheaterModeIcon from './PlayerTheaterModeIcon';
import useSettingsStore from '@/app/store/useSettingsStore';
import VideoPlayerHideChatIcon from './PlayerHideChatIcon';

interface Params {
  video: Video;
  ref: RefObject<MediaPlayerInstance | null>;
}

const VideoPlayer = ({ video, ref }: Params) => {
  const searchParams = useSearchParams()

  const isLoggedIn = useAuthStore(state => state.isLoggedIn);

  const player = ref;
  const [videoSource, setVideoSource] = useState<MediaSrc>();
  const [videoPoster, setVideoPoster] = useState<string>("");

  const hasStartedPlayback = useRef(false);

  const [playerVolume, setPlayerVolume] = useState(1);

  const updatePlaybackProgressMutation = useUpdatePlaybackProgressForVideo()
  const setPlaybackProgressMutation = useSetPlaybackProgressForVideo()

  const videoTheaterMode = useSettingsStore((state) => state.videoTheaterMode);

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

  useEffect(() => {
    if (!player) return

    const videoExtension = video.video_path.substr(video.video_path.length - 4)
    let videoType: VideoMimeType = "video/mp4"
    if (videoExtension == "m3u8") {
      videoType = "video/object";
    }

    // Allow for processing videos to be played via HLS from the temp directory if enabled
    if (video.processing) {
      setVideoSource({
        src: `${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.tmp_video_hls_path)}/${video.ext_id}-video.m3u8`,
        type: "application/x-mpegurl"
      })
    } else {
      setVideoSource({
        src: `${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.video_path)}`,
        type: videoType
      })
    }

    if (video.thumbnail_path) {
      setVideoPoster(`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.thumbnail_path)}`)
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

    // Resume from server-side playback progress.
    if (playbackData && playbackData.time) {
      player.current!.currentTime = playbackData.time
    }

    // Check if time is set in the url
    const time = searchParams.get("t");
    if (time) {
      player.current!.currentTime = parseInt(time);
    }

  }, [player, video, playbackData, searchParams])


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
    >
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
          afterFullscreenButton: <VideoPlayerHideChatIcon />
        }}
        thumbnails={thumbnails}
      />
    </MediaPlayer>
  );
}

export default VideoPlayer;
