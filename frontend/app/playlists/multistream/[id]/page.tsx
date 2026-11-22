"use client"
import Head from "next/head";
import React, { ReactNode, useEffect, useMemo, useState } from "react";
import { escapeURL } from "../../../util/util";
import classes from "./PlaylistMultistream.module.css"
import { ActionIcon, Drawer, Text } from "@mantine/core";
import { IconChevronUp } from "@tabler/icons-react";
import { useDisclosure, useInterval } from "@mantine/hooks";
import { useGetPlaylist, useUpdateMultistreamVideoOffset, Playlist } from "@/app/hooks/usePlaylist";
import GanymedeLoader from "@/app/components/utils/GanymedeLoader";
import { MultistreamTimeline } from "@/app/components/videos/multistream/MultistreamTimeline";
import SyncedVideoPlayer from "@/app/components/videos/multistream/SyncedVideoPlayer";
import { env } from "next-runtime-env";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Video } from "@/app/hooks/useVideos";

interface Params {
  id: string;
}

type MultistreamLayout = {
  streamers: Record<string, {
    name: string
    vods: Video[],
    imagePath: string
  }>;
  startDateMs: number | null;
  endDateMs: number | null;
  gridWidth: number;
  gridHeight: number;
  defaultViewState: Record<string, { tileX: number; tileY: number; tileWidth: number; tileHeight: number }>;
  vodPlaybackOffsets: Record<string, number>;
};

// Pure derivation of the multistream layout (timeline bounds, tiles, default
// view and saved offsets) from the playlist data.
function computeMultistreamLayout(playlist: Playlist): MultistreamLayout {
  let _startDateMs: number | null = null;
  let _endDateMs: number | null = null;
  const _streamers: Record<string, {
    name: string
    vods: Video[],
    imagePath: string
  }> = {};
  for (let i = 0; i < playlist.edges.vods.length; i++) {
    const vod = playlist.edges.vods[i];
    const vodStartDateMs = +new Date(getVodStartDate(vod))
    if (_startDateMs == null || vodStartDateMs < _startDateMs) {
      _startDateMs = vodStartDateMs;
    }
    const vodEndDateMs = vodStartDateMs + vod.duration * 1000;
    if (_endDateMs == null || _endDateMs < vodEndDateMs) {
      _endDateMs = vodEndDateMs;
    }

    if (!_streamers[vod.edges.channel.id]) {
      _streamers[vod.edges.channel.id] = {
        name: vod.edges.channel.name,
        vods: [],
        imagePath: vod.edges.channel.image_path,
      }
    }

    _streamers[vod.edges.channel.id].vods.push(vod)
  }

  // Compute default view
  const streamersCount = Object.keys(_streamers).length;
  let rows = 1;
  if (streamersCount > 3) {
    rows = 2;
  }
  const columns = Math.ceil(streamersCount / rows);
  let i = 0;
  const defaultViewState: Record<string, { tileX: number; tileY: number; tileWidth: number; tileHeight: number }> = {}
  for (const streamerId in _streamers) {
    defaultViewState[streamerId] = {
      tileX: i % columns,
      tileY: Math.floor(i / columns),
      tileWidth: 1,
      tileHeight: 1,
    }
    i++;
  }

  const vodPlaybackOffsets: Record<string, number> = {};
  if (playlist.edges.multistream_info) {
    for (let i = 0; i < playlist.edges.multistream_info.length; i++) {
      const multistreamInfo = playlist.edges.multistream_info[i];
      vodPlaybackOffsets[multistreamInfo.edges.vod.id] = multistreamInfo.delay_ms;
    }
  }

  return {
    streamers: _streamers,
    startDateMs: _startDateMs,
    endDateMs: _endDateMs,
    gridWidth: columns,
    gridHeight: rows,
    defaultViewState,
    vodPlaybackOffsets,
  };
}

const PlaylistMultistream = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);
  const axiosPrivate = useAxiosPrivate()
  const videoGrid = React.createRef<HTMLDivElement>();
  const [streamerViewState, setStreamerViewState] = useState<Record<string, { tileX: number; tileY: number; tileWidth: number; tileHeight: number } | null>>({});
  const [vodPlaybackOffsets, setVodPlaybackOffsets] = useState<Record<string, number>>({});

  const { data, isLoading, error } = useGetPlaylist(id, true)

  const updateVideoOffsetMutate = useUpdateMultistreamVideoOffset()

  const handleUpdateVideoOffset = async (vodId: string, offset: number) => {
    try {
      await updateVideoOffsetMutate.mutateAsync({ axiosPrivate, playlistId: id, videoId: vodId, delayMs: offset })
    } catch (error) {
      console.error("Error updating video offset", error)
    }
  }

  const [playing, setPlaying] = useState<boolean>(false);
  const [playStartAtDate, setPlayStartAtDate] = useState<number>(0);
  const [globalTime, setGlobalTime] = useState<number>(0);
  const [globalTimeUpdate, setGlobalTimeUpdate] = useState<number>(0);
  const [dragOverTile, setDragOverTile] = useState<[number, number] | null>(null);
  const [dropEnabled, setDropEnabled] = useState<boolean>(false);
  // Grid size defaults to the layout computed from the data but can be
  // overridden from the timeline drawer.
  const [gridWidth, setGridWidth] = useState<number>(2);
  const [gridHeight, setGridHeight] = useState<number>(1);
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [enterEvents, setEnterEvents] = useState<number>(0);
  const [resizeMode, setResizeMode] = useState<false | 'resize' | 'move'>(false);
  const [resizeOverlayParams, setResizeOverlayParams] = useState<{ tileX: number; tileY: number; tileWidth: number; tileHeight: number, streamerId: string } | null>(null);

  const [opened, { open, close }] = useDisclosure(true);

  const videoCheckInterval = useInterval(() => {
    setGlobalTimeUpdate((playing ? (Date.now() - playStartAtDate) : 0) + globalTime)
  }, 1000)

  // Layout is derived from the playlist data; recomputed only when it changes.
  const layout = useMemo(() => (data ? computeMultistreamLayout(data) : null), [data]);
  const streamers = layout?.streamers ?? {};
  const startDateMs = layout?.startDateMs ?? null;
  const endDateMs = layout?.endDateMs ?? null;

  // Tile view state, playback offsets and grid size are user-mutated; reset
  // them when the playlist data arrives or changes.
  const [prevPlaylist, setPrevPlaylist] = useState(data);
  if (data !== prevPlaylist) {
    setPrevPlaylist(data);
    setStreamerViewState(layout?.defaultViewState ?? {});
    setVodPlaybackOffsets((prevState) => ({
      ...prevState,
      ...(layout?.vodPlaybackOffsets ?? {}),
    }));
    setGridWidth(layout?.gridWidth ?? 2);
    setGridHeight(layout?.gridHeight ?? 1);
  }

  useEffect(() => {
    videoCheckInterval.stop()
    if (!playing) {
      return
    }
    videoCheckInterval.start();
    return videoCheckInterval.stop;
  }, [playing, videoCheckInterval]);

  if (error) return <div>failed to load</div>;
  if (isLoading) return <GanymedeLoader />;
  if (!data || data.edges.vods.length === 0) return <div>Empty playlist, unable to start multistream.</div>;

  // Multistream wall-clock synchronization: seek records the wall-clock
  // anchor for live progress, and the scheduled-start clamp below calls it
  // during render (guarded, so it only fires once before the state lands).
  const seek = (newGlobalTime: number) => {
    // eslint-disable-next-line react-hooks/purity
    setPlayStartAtDate(Date.now());
    setGlobalTime(newGlobalTime);
    setGlobalTimeUpdate(newGlobalTime);
  }

  // Clamp the timeline to the playlist's scheduled start.
  if (startDateMs != null && globalTime < startDateMs) {
    seek(startDateMs);
  }

  const onUserPlay = () => {
    setPlayStartAtDate(Date.now());
    setPlaying(true);
  }

  const onUserPause = (pausedAtGlobalTime: number) => {
    setPlaying(false);
    setGlobalTime(pausedAtGlobalTime)
    setGlobalTimeUpdate(pausedAtGlobalTime);
  }

  const startMovingTile = (streamerId: string) => {
    if (!streamerViewState[streamerId]) return
    setResizeMode('move');
    setResizeOverlayParams({ ...streamerViewState[streamerId], streamerId });
  }

  const startResizingTile = (streamerId: string) => {
    if (!streamerViewState[streamerId]) return
    setResizeMode('resize');
    setResizeOverlayParams({ ...streamerViewState[streamerId], streamerId });
  }

  const checkResize = (event: React.PointerEvent) => {
    if (!resizeOverlayParams || !resizeMode) return;
    const gridRect = videoGrid.current?.getBoundingClientRect();
    if (!gridRect) return;
    const tileXUnderMouse = Math.floor((event.clientX - gridRect.left) / (gridRect.width / gridWidth));
    const tileYUnderMouse = Math.floor((event.clientY - gridRect.top) / (gridRect.height / gridHeight));
    switch (resizeMode) {
      case 'move': {
        setResizeOverlayParams((prevState) => {
          if (!prevState) return null;
          const newState = { ...prevState };
          newState.tileX = Math.min(tileXUnderMouse, gridWidth - 1);
          newState.tileY = Math.min(tileYUnderMouse, gridHeight - 1);
          return newState;
        })
        break
      }
      case 'resize': {
        setResizeOverlayParams((prevState) => {
          if (!prevState) return null;
          const newState = { ...prevState };
          newState.tileWidth = Math.min(Math.max(1, tileXUnderMouse - prevState.tileX + 1), gridWidth - prevState.tileX);
          newState.tileHeight = Math.min(Math.max(1, tileYUnderMouse - prevState.tileY + 1), gridHeight - prevState.tileY);
          return newState;
        })
        break
      }
    }
  }

  const endTileResize = () => {
    if (resizeMode) {
      setResizeMode(false);
    } else {
      return
    }
    if (!resizeOverlayParams) return;
    setStreamerViewState((prevState) => {
      const newState = { ...prevState };
      newState[resizeOverlayParams.streamerId] = {
        tileX: resizeOverlayParams.tileX,
        tileY: resizeOverlayParams.tileY,
        tileWidth: resizeOverlayParams.tileWidth,
        tileHeight: resizeOverlayParams.tileHeight,
      };
      return newState;
    })
  }

  const playingVodForStreamer: Record<string, Video | null> = {};

  const playerTiles = Object.keys(streamerViewState).map((streamerId) => {
    const streamer = streamers[streamerId];
    const viewState = streamerViewState[streamerId];
    if (!viewState || !streamer) {
      return null;
    }
    const playingVod = getVodAtTime(streamer.vods, vodPlaybackOffsets, globalTimeUpdate);
    playingVodForStreamer[streamerId] = playingVod;
    if (!playingVod) {
      return <ResizableTile
        className={`${classes.streamerOffline} ${classes.playerTile}`}
        style={{ '--tile-x': `${viewState.tileX + 1} / ${viewState.tileX + viewState.tileWidth + 1}`, '--tile-y': `${viewState.tileY + 1} / ${viewState.tileY + viewState.tileHeight + 1}` } as React.CSSProperties}
        key={streamer.name + "-no-playing-vod"}
        startMoving={() => startMovingTile(streamerId)}
        startResizing={() => startResizingTile(streamerId)}
        remove={() => {
          setStreamerViewState((prevState) => {
            const newState = { ...prevState };
            delete newState[streamerId];
            return newState;
          })
        }}
      >
        <Text size="xl" span>
          {streamer.name}<br />
          <Text size="xl" fw={700} span>OFFLINE</Text>
        </Text>
      </ResizableTile>
    }
    const playbackOffset = (vodPlaybackOffsets[playingVod.id] || 0) / 1000;
    // Live wall-clock progress: the rendered position is the current time,
    // so the clock must be read during render for frame-accurate sync.
    // eslint-disable-next-line react-hooks/purity
    const currentGlobalTime = (playing ? (Date.now() - playStartAtDate) : 0) + globalTime
    const vodTime = (currentGlobalTime - (+new Date(getVodStartDate(playingVod)))) / 1000 + playbackOffset;
    return (
      <ResizableTile
        className={classes.playerTile}
        style={{ '--tile-x': `${viewState.tileX + 1} / ${viewState.tileX + viewState.tileWidth + 1}`, '--tile-y': `${viewState.tileY + 1} / ${viewState.tileY + viewState.tileHeight + 1}` } as React.CSSProperties}
        key={playingVod.id + "-vod-player"}
        startMoving={() => startMovingTile(streamerId)}
        startResizing={() => startResizingTile(streamerId)}
        remove={() => {
          setStreamerViewState((prevState) => {
            const newState = { ...prevState };
            delete newState[streamerId];
            return newState;
          })
        }}
      >
        <SyncedVideoPlayer
          src={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(playingVod.video_path)}`}
          vodId={playingVod.id}
          title={playingVod.title}
          poster={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(playingVod.web_thumbnail_path)}`}
          time={vodTime}
          playing={playing}
          muted={true}
        />
      </ResizableTile>
    )
  })

  const checkDropData = (event: React.DragEvent) => {
    if (!event.dataTransfer.types.includes('streamerid')) return
    setDropEnabled(true);
    setEnterEvents((prevState) => {
      const newEnterEvents = Math.min(2, prevState + 1)
      return newEnterEvents
    })
  }
  const leaveDropZone = () => {
    setEnterEvents((prevState) => {
      const newEnterEvents = Math.max(0, prevState - 1)
      if (newEnterEvents == 0) {
        setDropEnabled(false)
        setDragOverTile(null)
      }
      return newEnterEvents
    })
  }

  const dragOverTileHandler = (x: number, y: number, event: React.DragEvent, immediate: boolean) => {
    if (!event.dataTransfer.types.includes('streamerid')) return
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
    if (immediate) {
      setDragOverTile([x, y])
    } else {
      setTimeout(() => {
        setDragOverTile([x, y])
      })
    }
  }

  const dropOverTileHander = (x: number, y: number, event: React.DragEvent) => {
    event.preventDefault();
    const streamerId = event.dataTransfer.getData('streamerid');
    setStreamerViewState((prevState) => {
      const newState = { ...prevState };
      newState[streamerId] = { tileX: x, tileY: y, tileWidth: 1, tileHeight: 1 };
      return newState;
    })
    setEnterEvents(0)
    setDropEnabled(false)
    setDragOverTile(null)
  }

  const dropTiles: React.JSX.Element[] = []
  for (let x = 0; x < gridWidth; x++) {
    for (let y = 0; y < gridHeight; y++) {
      dropTiles.push(<div
        className={`${classes.dropTile} ${dragOverTile != null && dragOverTile[0] === x && dragOverTile[1] === y ? classes.dropTileHovered : ''}`}
        style={{ '--tile-x': x + 1, '--tile-y': y + 1 } as React.CSSProperties}
        key={`drop-tile-${x}-${y}`}
        onDragEnter={(event) => { dragOverTileHandler(x, y, event, false) }}
        onDragOver={(event) => { dragOverTileHandler(x, y, event, true) }}
        onDrop={(event) => { dropOverTileHander(x, y, event) }}
      ></div>)
    }
  }

  return (
    <div>
      <Head>
        <title>{data.name} - Ganymede Multistream</title>
      </Head>

      <div className={classes.pageWrapper}>
        <div
          ref={videoGrid}
          className={`${classes.videosGrid} ${dropEnabled ? classes.dropEnabled : ''} ${resizeMode ? classes.resizeMode : ''}`}
          style={{ '--grid-columns-count': gridWidth, '--grid-rows-count': gridHeight } as React.CSSProperties}
          onDragLeave={() => { leaveDropZone(); }}
          onDragEnter={(event) => { checkDropData(event); }}
          onPointerUp={() => { endTileResize() }}
          onPointerMove={(event) => { if (resizeMode) checkResize(event) }}
        >
          {playerTiles}
          {dropTiles}
          {resizeMode && <div className={`${classes.resizeOverlay}`} style={{ '--tile-x': resizeOverlayParams?.tileX, '--tile-y': resizeOverlayParams?.tileY, '--tile-width': resizeOverlayParams?.tileWidth, '--tile-height': resizeOverlayParams?.tileHeight } as React.CSSProperties}></div>}
        </div>
        <div className={classes.timelineOpenButtonContainer}>
          <ActionIcon
            onClick={open}
            size="sm"
            color="violet"
            variant="light"
          >
            <IconChevronUp />
          </ActionIcon>
        </div>
        <Drawer opened={opened} onClose={close} position="bottom" size="xs" overlayProps={{ backgroundOpacity: 0.1 }}>
          <MultistreamTimeline
            play={onUserPlay}
            pause={() => { onUserPause(globalTime + Date.now() - playStartAtDate) }}
            endDateMs={endDateMs}
            startDateMs={startDateMs}
            globalTime={globalTime}
            playing={playing}
            playingVodForStreamer={playingVodForStreamer}
            streamers={streamers}
            playStartAtDate={playStartAtDate}
            seek={seek}
            setVodOffset={(vodId, offset) => {
              setVodPlaybackOffsets((prevState) => {
                const newState = { ...prevState };
                newState[vodId] = offset;
                return newState;
              })
              handleUpdateVideoOffset(vodId, offset)
            }}
            vodPlaybackOffsets={vodPlaybackOffsets}
            onStreamerDragStart={() => { close() }}
            gridWidth={gridWidth}
            gridHeight={gridHeight}
            setGridWidth={(width) => { setGridWidth(width) }}
            setGridHeight={(height) => { setGridHeight(height) }}
          />
        </Drawer>
      </div>

    </div>
  )
}

function getVodStartDate(vod: Video): Date {
  if (!vod) {
    return new Date();
  }
  if (vod.type === 'live') {
    return vod.created_at;
  }
  return vod.streamed_at;
}

function getVodAtTime(vods: Video[], vodPlaybackOffsets: Record<string, number>, time: number): Video | null {
  for (let i = 0; i < vods.length; i++) {
    const vod = vods[i];
    const playbackOffset = (vodPlaybackOffsets[vod.id] || 0) / 1000
    const offsettedTime = time + playbackOffset;
    const vodStartDateMs = +new Date(getVodStartDate(vod))
    const vodEndDateMs = vodStartDateMs + vod.duration * 1000;
    if (vodStartDateMs <= offsettedTime && offsettedTime <= vodEndDateMs) {
      return vod;
    }
  }
  return null;
}

type ResizableTileProps = {
  className: string;
  style?: React.CSSProperties;
  startMoving: () => void;
  startResizing: () => void;
  remove: () => void;
  children: ReactNode
}
function ResizableTile(props: ResizableTileProps) {
  return <div className={`${classes.resizableTile} ${props.className}`} style={props.style}>
    {props.children}
    <div className={classes.topLeftHandle} onPointerDown={() => props.startMoving()}></div>
    <div className={classes.topRightHandle} onClick={() => props.remove()}></div>
    <div className={classes.bottomRightHandle} onPointerDown={() => props.startResizing()}></div>
  </div>
}

export default PlaylistMultistream;