import { Video } from "@/app/hooks/useVideos";
import { Badge, Card, Image, Progress, Tooltip, Text, Title, Group, Center, Avatar, Flex, ThemeIcon, LoadingOverlay, Loader, Box } from "@mantine/core";
import Link from "next/link";
import { useEffect, useState } from "react";
import dayjs from "dayjs";
import duration from "dayjs/plugin/duration";
import localizedFormat from "dayjs/plugin/localizedFormat";
import classes from "./Card.module.css"
import { env } from "next-runtime-env";
import { durationToTime, escapeURL, prettyNumber } from "@/app/util/util";
import { PlaybackStatus, useFetchPlaybackForVideo } from "@/app/hooks/usePlayback";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import useAuthStore from "@/app/store/useAuthStore";
import { IconCircleCheck, IconLock } from "@tabler/icons-react";
import VideoMenu from "./Menu";
import { UserRole } from "@/app/hooks/useAuthentication";
import { useTranslations } from "next-intl";
import VideoSpritePeek from "./VideoSpritePeek"

dayjs.extend(duration);
dayjs.extend(localizedFormat);

type Props = {
  video: Video;
  showProgress: boolean;
  showMenu: boolean;
  showChannel: boolean;
  showViewCount?: boolean;
}

const VideoCard = ({ video, showProgress = true, showMenu = true, showChannel = true, showViewCount = true }: Props) => {
  const t = useTranslations('VideoComponents')
  const { isLoggedIn, hasPermission } = useAuthStore()
  const [thumbnailError, setThumbnailError] = useState(false);

  const [playbackProgress, setPlaybackProgress] = useState(0);
  const [playbackIsWatched, setPlaybackIsWatched] = useState(false)

  // Handle thumbnail loading error
  const handleThumbnailError = () => {
    setThumbnailError(true);
  };

  const axiosPrivate = useAxiosPrivate();
  const { data: playbackData } = useFetchPlaybackForVideo(
    axiosPrivate,
    video.id,
    {
      enabled: (showProgress && isLoggedIn)
    }
  );

  // Set playback state
  useEffect(() => {
    if (!playbackData) return
    setPlaybackProgress(((playbackData.time) / video.duration) * 100);
    setPlaybackIsWatched(playbackData.status == PlaybackStatus.Finished)
  }, [playbackData, video.duration])

  return (
    <Card radius="md" padding={5} className={classes.card}>

      {video.processing && (
        <LoadingOverlay visible zIndex={5} overlayProps={{ radius: "sm", blur: 1 }} loaderProps={{
          children: <div><Text size="xl">{t('processingOverlayText')}</Text>
            <Center>
              <Box>
                <Loader color="red" />
              </Box>
            </Center></div>
        }} />
      )}

      <Link href={`/videos/${video.id}`}>

        <Card.Section>
          <div className={classes.videoImageWrapper}>
            <Image
              className={classes.videoImage}
              src={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(
                video.web_thumbnail_path
              )}`}
              onError={handleThumbnailError}
              width={thumbnailError ? "100%" : "100%"}
              height={thumbnailError ? "100%" : "100%"}
              fallbackSrc="/images/ganymede-thumbnail.webp"
              alt={video.title}
            />

            {video.sprite_thumbnails_enabled && (
              <VideoSpritePeek
                video={video}
                progressDisplayed={
                  showProgress &&
                  Math.round(playbackProgress) > 0 &&
                  !playbackIsWatched
                }
              />
            )}
          </div>
          {showProgress && Math.round(playbackProgress) > 0 && !playbackIsWatched && (
            <Tooltip label={`${Math.round(playbackProgress)}% ${t('watched')}`}>
              <Progress
                className={classes.progressBar}
                color="red"
                radius={0}
                size="sm"
                value={playbackProgress}
              />
            </Tooltip>
          )}
        </Card.Section>

      </Link>

      {/* Duration badge */}
      <Badge py={0} px={5} className={classes.durationBadge} radius="md">
        <Text>
          {durationToTime(video.duration)}
        </Text>
      </Badge>

      <Flex
        gap="xs"
        justify="flex-start"
        align="flex-start"
        direction="column"
        wrap="wrap"
        className={classes.watchedIcon}
      >
        {/* Watched icon */}
        {showProgress && playbackIsWatched && (
          <Tooltip label={t('watchedVideoText')}>
            <ThemeIcon radius="xl" color="green">
              <IconCircleCheck />
            </ThemeIcon>
          </Tooltip>
        )}

        {/* Locked icon */}
        {video.locked && (
          <Tooltip label={t('lockedText')}>
            <ThemeIcon radius="xl" color="gray">
              <IconLock />
            </ThemeIcon>
          </Tooltip>
        )}
      </Flex>

      {/* Title */}
      <Link href={video.processing ? `#` : `/videos/${video.id}`}>
        <Tooltip label={video.title} openDelay={250} withinPortal>
          <Title lineClamp={1} order={4} mt="xs">
            {video.title}
          </Title>
        </Tooltip>
      </Link>

      {/* Optionally show channel */}
      {showChannel && (
        <Group className={classes.channelFooter}>
          <Center>
            <Avatar
              src={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(
                video.edges.channel.image_path
              )}`}
              size={24}
              radius="xl"
              mr="xs"
            />
            <Link href={`/channels/${video.edges.channel.name}`}>
              <Text fz="sm" inline>
                {video.edges.channel.display_name}
              </Text>
            </Link>
          </Center>
        </Group>
      )}

      {/* Additional video information and menu */}
      <Flex justify="flex-start"
        align="center" pt={2}>

        <Tooltip
          label={`${t('streamedOnText')} ${new Date(
            video.streamed_at
          ).toLocaleString()}`}
        >
          <Text size="sm">
            {dayjs(video.streamed_at).format("YYYY/MM/DD")}
          </Text>
        </Tooltip>

        <div className={classes.vodMenu}>
          <Box pt={4} pr={5}>
            {showViewCount && (
              <Tooltip
                multiline
                label={`${video.views} ${t('sourceViewsText')}
               ${video.local_views ?? 0} ${t('localViewsText')}`}
              >
                <Text size="sm">
                  {prettyNumber(video.views)} {t('viewsText')}
                </Text>
              </Tooltip>
            )}
          </Box>

          <Badge variant="default" color="rgba(0, 0, 0, 1)" mt={4}>
            {video.type.toUpperCase()}
          </Badge>

          {(showMenu && hasPermission(UserRole.Archiver)) && (
            <VideoMenu video={video} />
          )}
        </div>
      </Flex>

    </Card >
  );
}

export default VideoCard;