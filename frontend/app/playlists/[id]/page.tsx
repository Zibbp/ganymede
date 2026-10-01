"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import VideoGrid from "@/app/components/videos/Grid";
import { useGetPlaylist } from "@/app/hooks/usePlaylist";
import { useFetchVideosFilter } from "@/app/hooks/useVideos";
import { useVideoListParams } from "@/app/hooks/useVideoListParams";
import useSettingsStore from "@/app/store/useSettingsStore";
import { Center, Container, Title, Text, Button } from "@mantine/core";
import { IconBorderAll } from "@tabler/icons-react";
import { useTranslations } from "next-intl";
import Link from "next/link";
import React, { useEffect } from "react";
interface Params {
  id: string;
}

const PlaylistPage = ({ params }: { params: Promise<Params> }) => {
  const { id } = React.use(params);
  const t = useTranslations("PlaylistsPage");
  const {
    data: playlist,
    isPending: playlistPending,
    isError: playlistError
  } = useGetPlaylist(id, false);

  useEffect(() => {
    document.title = `${playlist?.name}`;
  }, [playlist?.name]);

  const { page, videoTypes, sortBy, order, setPage, setVideoTypes, setSortBy, setOrder } = useVideoListParams();

  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const {
    data: videos,
    isPending: videosPending,
    isError: videosError
  } = useFetchVideosFilter({
    limit: videoLimit,
    offset: (page - 1) * videoLimit,
    types: videoTypes,
    playlist_id: id,
    sort_by: sortBy,
    order: order,
  });

  if (playlistPending || videosPending) {
    return <GanymedeLoadingText message={t('loadingVideos')} />;
  }

  if (playlistError || videosError) {
    return <div>{t('errorLoadingVideos')}</div>;
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
        currentPage={page}
        onPageChange={setPage}
        isPending={videosPending}
        videoLimit={videoLimit}
        onVideoLimitChange={setVideoLimit}
        videoTypes={videoTypes}
        onVideoTypeChange={setVideoTypes}
        sortBy={sortBy}
        onSortByChange={setSortBy}
        order={order}
        onOrderChange={setOrder}
        showChannel={false}
      />
    </Container>
  );
}

export default PlaylistPage;