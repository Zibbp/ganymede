'use client';
import { Center, Container, Title } from "@mantine/core";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import { useEffect, useState } from "react";
import { useFetchVideosFilter, VideoOrder, VideoSortBy, VideoType } from "../hooks/useVideos";
import useSettingsStore from "../store/useSettingsStore";
import VideoGrid from "../components/videos/Grid";
import { useTranslations } from "next-intl";
import { usePageTitle } from "../util/util";

const VideosPage = () => {
  const t = useTranslations("VideosPage");
  usePageTitle(t('title'));

  const [activePage, setActivePage] = useState(1);
  const [videoTypes, setVideoTypes] = useState<VideoType[]>([]);
  const [sortBy, setSortBy] = useState<VideoSortBy>(VideoSortBy.Date);
  const [order, setOrder] = useState<VideoOrder>(VideoOrder.Desc);

  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const { data: videos, isPending, isError } = useFetchVideosFilter({
    limit: videoLimit,
    offset: (activePage - 1) * videoLimit,
    types: videoTypes,
    playlist_id: "",
    sort_by: sortBy,
    order: order,
  });

  if (isPending) {
    return <GanymedeLoadingText message={t('loading')} />;
  }

  if (isError) {
    return <div>{t('error')}</div>;
  }

  return (
    <div>
      <Center mt={10}>
        <Title>{t('title')}</Title>
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
          onSortByChange={setSortBy}
          onOrderChange={setOrder}
          showChannel={true}
        />
      </Container>
    </div>
  );
}

export default VideosPage;
