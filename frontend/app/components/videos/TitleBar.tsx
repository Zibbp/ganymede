"use client"
import { Video } from "@/app/hooks/useVideos";
import { escapeURL } from "@/app/util/util";
import { Avatar, Box, Divider, Tooltip, Text, Group, Badge } from "@mantine/core";
import { env } from "next-runtime-env";
import classes from "./TitleBar.module.css";
import { IconCalendarEvent, IconUser, IconUsers } from "@tabler/icons-react";
import dayjs from "dayjs";
import VideoMenu from "./Menu";
import useAuthStore from "@/app/store/useAuthStore";
import { UserRole } from "@/app/hooks/useAuthentication";

interface Params {
  video: Video;
}

const VideoTitleBar = ({ video }: Params) => {
  const hasPermission = useAuthStore(state => state.hasPermission);

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

            {video.views && (
              <Group mr={15}>
                <Tooltip
                  label={`${video.views.toLocaleString()} source views`}
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
                  label={`${video.local_views.toLocaleString()} local views`}
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
                label={`Originally streamed at ${video.streamed_at}`}
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

            <Group>
              <Tooltip label={`Video Type`} openDelay={250}>
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
    </div>

    // <Box className={classes.titleBarContainer}>
    //   <div className={classes.titleBar}>
    //     <Group
    //       justify="space-between"
    //       align="center"
    //       style={{ width: '100%' }}
    //       px="md"
    //       gap="xl"
    //     >
    //       {/* Left side - Channel info */}
    //       <Group align="center" wrap="nowrap" style={{ flex: '1 1 auto', minWidth: 0 }}>
    //         <Avatar
    //           src={`${env('CDN_URL')}${escapeURL(video.edges.channel.image_path)}`}
    //           radius="xl"
    //           alt={video.edges.channel.display_name}
    //         />
    //         <Divider size="sm" orientation="vertical" />
    //         <div style={{ minWidth: 0, flex: 1 }}>
    //           <Tooltip label={video.title} openDelay={250}>
    //             <Text size="xl" lineClamp={1}>
    //               {video.title}
    //             </Text>
    //           </Tooltip>
    //         </div>
    //       </Group>

    //       {/* Right side - Metrics */}
    //       <Group align="center" wrap="nowrap" gap="md" style={{ flex: '0 0 auto' }}>
    //         {video.views && (
    //           <MetricBadge
    //             value={video.views}
    //             icon={IconUsers}
    //             tooltip={`${video.views.toLocaleString()} source views`}
    //           />
    //         )}

    //         {video.local_views && (
    //           <MetricBadge
    //             value={video.local_views}
    //             icon={IconUser}
    //             tooltip={`${video.local_views.toLocaleString()} local views`}
    //           />
    //         )}

    //         <Tooltip label={`Originally streamed at ${video.streamed_at}`} openDelay={250}>
    //           <div className={classes.titleBarMetric}>
    //             <Text mr={5}>
    //               {dayjs(video.streamed_at).format("YYYY/MM/DD")}
    //             </Text>
    //             <IconCalendarEvent size={20} />
    //           </div>
    //         </Tooltip>

    //         <Tooltip label="Video Type" openDelay={250}>
    //           <Badge variant="default">
    //             {video.type}
    //           </Badge>
    //         </Tooltip>

    //         {hasPermission(UserRole.Archiver) && (
    //           <VideoMenu />
    //         )}

    //       </Group>
    //     </Group>
    //   </div>
    // </Box>
  );
};

export default VideoTitleBar;