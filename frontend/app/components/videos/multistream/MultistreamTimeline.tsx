import { ActionIcon, Center, Group, Image, NumberInput, Stack, Tooltip } from "@mantine/core";
import classes from "./MultistreamTimeline.module.css"
import { IconPlayerPauseFilled, IconPlayerPlayFilled, IconRewindBackward5, IconRewindBackward60, IconRewindForward5, IconRewindForward60 } from "@tabler/icons-react";
import React, { Fragment, useRef, useState } from "react";
import dayjs from "dayjs";
import { escapeURL } from "@/app/util/util";
import { env } from "next-runtime-env";
import { Video } from "@/app/hooks/useVideos";
import { useTranslations } from "next-intl";

export type MultistreamTimelineProps = {
  seek: (time: number) => void;
  pause: () => void;
  play: () => void;
  vodPlaybackOffsets: Record<string, number>;
  globalTime: number;
  startDateMs: number | null;
  endDateMs: number | null;
  playStartAtDate: number;
  playing: boolean;
  setVodOffset: (vodId: string, offset: number) => void;
  playingVodForStreamer: Record<string, Video | null>;
  streamers: Record<string, {
    name: string
    vods: Video[],
    imagePath: string
  }>;
  gridWidth: number;
  gridHeight: number;
  setGridWidth: (width: number) => void;
  setGridHeight: (height: number) => void;
  onStreamerDragStart: () => void;
}

export const MultistreamTimeline = ({ vodPlaybackOffsets, globalTime, startDateMs, endDateMs, playStartAtDate, seek, pause, play, playing, playingVodForStreamer, streamers, setVodOffset, onStreamerDragStart, gridWidth, gridHeight, setGridWidth, setGridHeight }: MultistreamTimelineProps) => {
  const t = useTranslations("MultistreamComponents");
  const [timelineTooltipText, setTimelineTooltipText] = useState<string>("");
  const [hoverPlayheadDate, setHoverPlayheadDate] = useState<number | null>(null);
  const timeAtMousePosition = (timelineBar: HTMLDivElement | null, event: React.MouseEvent) => {
    if (!timelineBar) return null;
    const rect = timelineBar.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const percentage = x / rect.width;
    const globalTime = startDateMs! + percentage * timelineDurationMs;
    return globalTime;
  }

  const streamerImageRefs = useRef<Record<string, HTMLImageElement | null>>({});

  const onTimelineClick = (timelineBar: HTMLDivElement | null, event: React.MouseEvent) => {
    const newGlobalTime = timeAtMousePosition(timelineBar, event);
    if (newGlobalTime == null) {
      return;
    }
    seek(newGlobalTime);
  }

  const hideHoverPlayhead = () => {
    setHoverPlayheadDate(null);
  }
  const updateHoverPlayhead = (timelineBar: HTMLDivElement | null, event: React.PointerEvent) => {
    setHoverPlayheadDate(timeAtMousePosition(timelineBar, event));
  }
  const timelineDurationMs: number = startDateMs != null && endDateMs != null ? endDateMs - startDateMs : 0;

  const getCurrentTime = () => {
    return (playing ? (Date.now() - playStartAtDate) : 0) + globalTime
  }

  const updateTimelineTooltip = (timelineBar: HTMLDivElement | null, event: React.PointerEvent) => {
    const timeUnderPointer = timeAtMousePosition(timelineBar, event);
    setTimelineTooltipText(timeUnderPointer != null ? dayjs(timeUnderPointer).format("YYYY/MM/DD HH:mm:ss") : "");
  }

  const onStreamerNameDragStart = (event: React.DragEvent, streamerId: string) => {
    event.dataTransfer.setData("streamerid", streamerId);
    event.dataTransfer.effectAllowed = 'move';
    if (streamerImageRefs.current?.[streamerId]) {
      event.dataTransfer.setDragImage(streamerImageRefs.current[streamerId], 0, 0);
    }
    onStreamerDragStart();
  }

  const timelineEnd = dayjs(endDateMs).format("YYYY/MM/DD HH:mm:ss")

  return <Stack gap="sm">
    <Group justify="center" gap="xs">
      <Tooltip label={t('seekBackTooltip', { seconds: "60" })} position="top">
        <ActionIcon
          size="sm"
          variant="subtle"
          color="violet"
          aria-label={t('seekBackTooltip', { seconds: "60" })}
          onClick={() => seek(getCurrentTime() - 60000)}
        ><IconRewindBackward60 /></ActionIcon>
      </Tooltip>

      <Tooltip label={t('seekBackTooltip', { seconds: "5" })} position="top">
        <ActionIcon
          size="sm"
          variant="subtle"
          color="violet"
          aria-label={t('seekBackTooltip', { seconds: "5" })}
          onClick={() => seek(getCurrentTime() - 5000)}
        ><IconRewindBackward5 /></ActionIcon>
      </Tooltip>

      <Tooltip label={playing ? t('pause') : t('play')} position="top">
        <ActionIcon
          // eslint-disable-next-line @typescript-eslint/no-unused-expressions
          onClick={() => { playing ? pause() : play() }}
          size="md"
          variant="subtle"
          color="violet"
          aria-label={playing ? t('pause') : t('play')}
        >
          {playing ? <IconPlayerPauseFilled /> : <IconPlayerPlayFilled />}
        </ActionIcon>
      </Tooltip>

      <Tooltip label={t('seekForwardTooltip', { seconds: "5" })} position="top">
        <ActionIcon
          size="sm"
          variant="subtle"
          color="violet"
          aria-label={t('seekForwardTooltip', { seconds: "5" })}
          onClick={() => seek(getCurrentTime() + 5000)}
        ><IconRewindForward5 /></ActionIcon>
      </Tooltip>

      <Tooltip label={t('seekForwardTooltip', { seconds: "60" })} position="top">
        <ActionIcon
          size="sm"
          variant="subtle"
          color="violet"
          aria-label={t('seekForwardTooltip', { seconds: "60" })}
          onClick={() => seek(getCurrentTime() + 60000)}
        ><IconRewindForward60 /></ActionIcon>
      </Tooltip>

      <NumberInput className={classes.gridInput} label={t('gridWidth')} value={gridWidth} onChange={(value) => setGridWidth(+value)} size="xs" step={1} min={1} />
      <NumberInput className={classes.gridInput} label={t('gridHeight')} value={gridHeight} onChange={(value) => setGridHeight(+value)} size="xs" step={1} min={1} />
    </Group>

    <Center>
      <Group gap="sm">
        <div>
          {dayjs(getCurrentTime()).format("YYYY/MM/DD HH:mm:ss")} / {timelineEnd}
        </div>
      </Group>
    </Center>

    {
      startDateMs != null && endDateMs != null && <div className={classes.timelineGrid}>
        {Object.keys(streamers).map((streamerId) => {
          const streamer = streamers[streamerId]

          const timelineBar = <div className={classes.timelineBar}>
            {streamer.vods.map(vod => <div key={vod.id + "-vod-timeline-online"} className={classes.timelineBarActive} style={{
              '--bar-start': `${100 * (+new Date(getVodStartDate(vod)) - startDateMs!) / timelineDurationMs}%`,
              '--bar-length': `${100 * 1000 * vod.duration / timelineDurationMs}%`,
            } as React.CSSProperties}></div>)}
          </div>

          const playingVod = playingVodForStreamer[streamerId];

          return (
            <Fragment key={streamer.name + "-timeline-row"}>
              <div className={classes.timelineStreamerColumn}>
                <Group gap='sm'>
                  <div onDragStart={(e) => onStreamerNameDragStart(e, streamerId)} draggable="true">
                    <Group gap="sm">
                      <Image ref={(img) => { streamerImageRefs.current[streamerId] = img }} src={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(streamer.imagePath)}`} alt={streamer.name} w={'1.5em'} h={'1.5em'} radius={'1.5em'} />
                      {streamer.name}
                    </Group>
                  </div>
                  <NumberInput
                    className={classes.offsetInput}
                    size="xs"
                    step={0.1}
                    value={playingVod && vodPlaybackOffsets[playingVod.id] != null ? (vodPlaybackOffsets[playingVod.id] || 0) / 1000 : ''}
                    placeholder="Offset"
                    disabled={!playingVod}
                    onChange={(value) => {
                      if (!playingVod) return;
                      const valAsNumber = Math.trunc(+value * 1000);
                      if (isNaN(valAsNumber)) return;
                      setVodOffset(playingVod.id, valAsNumber);
                    }}
                  />
                </Group>
              </div>
              <div>{timelineBar}</div>
            </Fragment>
          )
        })}

        {
          (() => {
            let timelineBarRef: HTMLDivElement | null = null;

            return <Tooltip.Floating label={timelineTooltipText}>
              <div className={classes.playheadContainer} ref={el => { timelineBarRef = el; }} onClick={(event) => onTimelineClick(timelineBarRef, event)} onPointerMove={(event) => { updateTimelineTooltip(timelineBarRef, event); updateHoverPlayhead(timelineBarRef, event) }} onPointerLeave={hideHoverPlayhead}>
                <div className={classes.playhead} style={{ '--playhead-position': `${((getCurrentTime() - startDateMs) / timelineDurationMs) * 100}%` } as React.CSSProperties}></div>
                {hoverPlayheadDate != null && <div className={`${classes.playhead} ${classes.playheadPreview}`} style={{ '--playhead-position': `${((hoverPlayheadDate - startDateMs) / timelineDurationMs) * 100}%` } as React.CSSProperties}></div>}
              </div>
            </Tooltip.Floating>
          })()
        }
      </div>
    }
  </Stack>
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