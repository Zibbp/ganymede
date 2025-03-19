'use client';
import { Center, Container, SimpleGrid, Title } from "@mantine/core";
import ChannelCard from "../components/channel/Card";
import { useFetchChannels } from "../hooks/useChannels";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import { useEffect, useState } from "react";
import { useFetchVideosFilter, VideoType } from "../hooks/useVideos";
import useSettingsStore from "../store/useSettingsStore";
import VideoGrid from "../components/videos/Grid";

const VideosPage = () => {
  useEffect(() => {
    document.title = "Videos";
  }, []);

  const [activePage, setActivePage] = useState(1);
  const [videoTypes, setVideoTypes] = useState<VideoType[]>([]);

  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const { data: videos, isPending, isError } = useFetchVideosFilter({
    limit: videoLimit,
    offset: (activePage - 1) * videoLimit,
    types: videoTypes,
    playlist_id: ""
  });

  if (isPending) {
    return <GanymedeLoadingText message="Loading Videos" />;
  }

  if (isError) {
    return <div>Error loading videos</div>;
  }

  return (
    <div>
      <Center mt={10}>
        <Title>All Videos</Title>
      </Center>

      <Container size="xl" px="xl" fluid={true}>
        <VideoGrid
          videos={videos.data}
          totalCount={videos.total_count}
          totalPages={videos.pages}
          currentPage={activePage}
          onPageChange={setActivePage}
          isPending={isPending}
          videoLimit={videoLimit}
          onVideoLimitChange={setVideoLimit}
          onVideoTypeChange={setVideoTypes}
          showChannel={true}
        />
      </Container>
    </div>
  );
}

export default VideosPage;
