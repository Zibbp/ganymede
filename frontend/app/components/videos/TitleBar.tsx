"use client"
import { useGetVideoByExternalId, Video } from "@/app/hooks/useVideos";
import { escapeURL, formatBytes } from "@/app/util/util";
import { Avatar, Box, Divider, Tooltip, Text, Group, Badge, Button, rem } from "@mantine/core";
import { env } from "next-runtime-env";
import classes from "./TitleBar.module.css";
import { IconCalendarEvent, IconDatabase, IconLock, IconUser, IconUsers } from "@tabler/icons-react";
import dayjs from "dayjs";
import VideoMenu from "./Menu";
import useAuthStore from "@/app/store/useAuthStore";
import { UserRole } from "@/app/hooks/useAuthentication";
import Link from "next/link";
import { useTranslations } from "next-intl";

interface Params {
  video: Video;
}

const VideoTitleBar = ({ video }: Params) => {
  const t = useTranslations("VideoComponents");
  const hasPermission = useAuthStore(state => state.hasPermission);

  const { data: clipFullVideo } = useGetVideoByExternalId(video.clip_ext_vod_id)

  return (
    <div className={classes.titleBarContainer}>
      <div className={classes.titleBar}>
        <Avatar
          src={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(video.edges.channel.image_path)}`}
          radius="xl"
          alt={video.edges.channel.display_name}
          mr={10}
        />

        <Divider size="sm" orientation="vertical" mr={10} />

        <div style={{ width: "60%" }}>
          <Tooltip label={video.title} openDelay={250}>
            <Text size="xl" lineClamp={1} pt={3}>
              {video.title}
            </Text>
          </Tooltip>
        </div>

        <div className={classes.titleBarRight}>

          <div className={classes.titleBarBadge}>

            {clipFullVideo && (
              <Group mr={15}>
                <Button variant="default" size="xs" component={Link} href={`/videos/${clipFullVideo.id}?t=${video.clip_vod_offset}`}>Go To Full Video</Button>
              </Group>
            )}

            {video.views && (
              <Group mr={15}>
                <Tooltip
                  label={`${video.views.toLocaleString()} ${t('sourceViewsTooltip')}`}
                  openDelay={250}
                >
                  <div className={classes.titleBarBadge}>
                    <Text mr={3}>{video.views.toLocaleString()}</Text>
                    <IconUsers size={20} />
                  </div>
                </Tooltip>
              </Group>
            )}

            {video.local_views && (
              <Group mr={15}>
                <Tooltip
                  label={`${video.local_views.toLocaleString()} ${t('localViewsTooltip')}`}
                  openDelay={250}
                >
                  <div className={classes.titleBarBadge}>
                    <Text mr={3}>{video.local_views.toLocaleString()}</Text>
                    <IconUser size={20} />
                  </div>
                </Tooltip>
              </Group>
            )}

            <Group mr={15}>
              <Tooltip
                label={`${t('streamedOnTooltip')} ${new Date(
                  video.streamed_at
                ).toLocaleString()}`}
                openDelay={250}
              >
                <div className={classes.titleBarBadge}>
                  <Text mr={5}>
                    {dayjs(video.streamed_at).format("YYYY/MM/DD")}
                  </Text>
                  <IconCalendarEvent size={20} />
                </div>
              </Tooltip>
            </Group>

            <Group mr={15}>
              <Tooltip
                label={`${t('storageSizeTooltip')}`}
                openDelay={250}
              >
                <div className={classes.titleBarBadge}>
                  <Text mr={5}>
                    {formatBytes(video.storage_size_bytes ?? 0, 0)}
                  </Text>
                  <IconDatabase size={20} />
                </div>
              </Tooltip>
            </Group>

            {video.locked && (
              <Group mr={5}>
                <Tooltip label={t('lockedText')} openDelay={250}>
                  <div className={classes.titleBarBadge}>
                    <Badge variant="default" leftSection={<IconLock style={{ width: rem(12), height: rem(12) }} />}>
                      {t('locked')}
                    </Badge>
                  </div>
                </Tooltip>
              </Group>
            )}

            <Group>
              <Tooltip label={t('videoTypeTooltip')} openDelay={250}>
                <div className={classes.titleBarBadge}>
                  <Badge variant="default">
                    {video.type}
                  </Badge>
                </div>
              </Tooltip>
            </Group>
          </div>

          {hasPermission(UserRole.Archiver) && (
            <Box mt={5}>
              <VideoMenu video={video} />
            </Box>
          )}

        </div>
      </div>
    </div >
  );
};

export default VideoTitleBar;