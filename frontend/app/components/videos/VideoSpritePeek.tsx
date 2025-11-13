import { Video } from "@/app/hooks/useVideos";
import classes from "./VideoSpritePeek.module.css";
import { useRef, useState } from "react";
import { Flex, Popover, Progress } from "@mantine/core";
import { env } from "next-runtime-env";
import { durationToTime, escapeURL } from "@/app/util/util";
import clsx from "clsx";
import {
  useDebouncedCallback,
  useElementSize,
  useThrottledCallback,
} from "@mantine/hooks";

interface Props {
  video: Video;
  progressDisplayed?: boolean;
}

interface HoveredTimeElement {
  time: number;
  spriteX: number;
  spriteY: number;
  imageSrc: string;
}

const VideoSpritePeek = ({ video, progressDisplayed }: Props) => {
  const [opened, setOpened] = useState<boolean>(false);
  const [hoveredTimeElement, setHoveredTimeElement] =
    useState<HoveredTimeElement | null>(null);
  const { ref: spritePeekContainerRef, width: spritePeekContainerWidth } =
    useElementSize();

  const onPointerMove = useThrottledCallback(
    (e: React.PointerEvent<HTMLDivElement>, currentTarget: HTMLDivElement) => {
      const { clientX } = e;
      const { left, width } = currentTarget.getBoundingClientRect();
      const offsetX = clientX - left;
      const percentage = Math.min(Math.max(offsetX / width, 0), 1);
      const time = percentage * video.duration;

      const interval = video.sprite_thumbnails_interval;
      const index = Math.floor(time / interval);
      const spritesPerSheet =
        video.sprite_thumbnails_columns * video.sprite_thumbnails_rows;
      const sheetIndex = Math.floor(index / spritesPerSheet);

      if (sheetIndex >= video.sprite_thumbnails_images.length) {
        setHoveredTimeElement(null);
        return;
      }

      const indexInSheet = index % spritesPerSheet;
      const spriteX = indexInSheet % video.sprite_thumbnails_columns;
      const spriteY = Math.floor(
        indexInSheet / video.sprite_thumbnails_columns
      );

      const imageSrc = video.sprite_thumbnails_images[sheetIndex];

      setHoveredTimeElement({
        time: index * interval,
        spriteX,
        spriteY,
        imageSrc,
      });
    },
    16
  );

  return (
    <Popover
      opened={opened}
      position="top"
      offset={{
        crossAxis:
          hoveredTimeElement && spritePeekContainerRef.current
            ? 0.5 *
              (hoveredTimeElement.time / video.duration - 0.5) *
              spritePeekContainerWidth
            : 0,
      }}
    >
      <Popover.Target>
        <div
          data-opened={opened}
          data-with-progress={progressDisplayed ? "true" : "false"}
          className={classes.spritePeekContainer}
          ref={spritePeekContainerRef}
          onPointerEnter={() => setOpened(true)}
          onPointerLeave={() => setOpened(false)}
          onPointerMove={(e) => onPointerMove(e, e.currentTarget)}
          onPointerDown={(e) => e.preventDefault()}
          onClick={(e) => e.preventDefault()}
        >
          <Progress
            className={clsx(Progress.classes.root, classes.spritePeekBar)}
            value={
              hoveredTimeElement
                ? (hoveredTimeElement.time / video.duration) * 100
                : 0
            }
            size="sm"
            radius={0}
            color="#acacac"
          />
        </div>
      </Popover.Target>
      <Popover.Dropdown p={0}>
        <Flex gap={0} justify="flex-start" align="center" direction="column">
          <div
            className={classes.spritePeekImage}
            style={
              {
                "--spritesheet-cols": `${video.sprite_thumbnails_columns}`,
                "--spritesheet-rows": `${video.sprite_thumbnails_rows}`,
                "--aspect-ratio": `${video.sprite_thumbnails_width / video.sprite_thumbnails_height}`,
                "--sprite-x": `${hoveredTimeElement ? hoveredTimeElement.spriteX : 0}`,
                "--sprite-y": `${hoveredTimeElement ? hoveredTimeElement.spriteY : 0}`,
                "--sprite-image": `url(${
                  hoveredTimeElement
                    ? `${env("NEXT_PUBLIC_CDN_URL") ?? ""}${escapeURL(
                        hoveredTimeElement.imageSrc
                      )}`
                    : ""
                })`,
              } as React.CSSProperties
            }
          ></div>
          <div>
            {hoveredTimeElement ? durationToTime(hoveredTimeElement.time) : ""}
          </div>
        </Flex>
      </Popover.Dropdown>
    </Popover>
  );
};

export default VideoSpritePeek;
