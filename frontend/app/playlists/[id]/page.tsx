"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import VideoGrid from "@/app/components/videos/Grid";
import { useGetPlaylist } from "@/app/hooks/usePlaylist";
import { useFetchVideosFilter, VideoType } from "@/app/hooks/useVideos";
import useSettingsStore from "@/app/store/useSettingsStore";
import { Center, Container, Title, Text, Button } from "@mantine/core";
import { IconBorderAll } from "@tabler/icons-react";
import Link from "next/link";
import React, { useEffect, useState } from "react";
interface Params {
  id: string;
}

const PlaylistPage = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);
  const {
    data: playlist,
    isPending: playlistPending,
    isError: playlistError
  } = useGetPlaylist(id, false);

  useEffect(() => {
    document.title = `${playlist?.name}`;
  }, [playlist?.name]);

  const [activePage, setActivePage] = useState(1);
  const [videoTypes, setVideoTypes] = useState<VideoType[]>([]);

  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const {
    data: videos,
    isPending: videosPending,
    isError: videosError
  } = useFetchVideosFilter({
    limit: videoLimit,
    offset: (activePage - 1) * videoLimit,
    types: videoTypes,
    playlist_id: id
  });

  if (playlistPending || videosPending) {
    return <GanymedeLoadingText message="Loading Videos" />;
  }

  if (playlistError || videosError) {
    return <div>Error loading channel</div>;
  }

  return (
    <Container size="xl" px="xl" fluid={true}>
      <Center>
        <Title>{playlist.name}</Title>
      </Center>
      <Center>
        <Text>{playlist.description}</Text>
      </Center>
      <Center>
        <Button
          component={Link}
          href={`/playlists/multistream/${playlist.id}`}
          leftSection={<IconBorderAll size={14} />} variant="default">
          Multistream View
        </Button>
      </Center>
      <VideoGrid
        videos={videos.data}
        totalCount={videos.total_count}
        totalPages={videos.pages}
        currentPage={activePage}
        onPageChange={setActivePage}
        isPending={videosPending}
        videoLimit={videoLimit}
        onVideoLimitChange={setVideoLimit}
        onVideoTypeChange={setVideoTypes}
        showChannel={false}
      />
    </Container>
  );
}

export default PlaylistPage;