"use client";
import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useSearchVideos, VideoType } from "../hooks/useVideos";
import useSettingsStore from "../store/useSettingsStore";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import VideoGrid from "../components/videos/Grid";
import { Center, Container, Title } from "@mantine/core";
import { useTranslations } from "next-intl";

const SearchPage = () => {
  const searchParams = useSearchParams();
  const queryParam = searchParams.get("q");

  const t = useTranslations("SearchPage");

  useEffect(() => {
    document.title = `${t('title')} - ${queryParam}`;
  }, [queryParam]);

  // State and ref for search query
  const [searchQuery, setSearchQuery] = useState(queryParam || "");
  const defaultSearchQuery = useRef("");

  useEffect(() => {
    if (queryParam && queryParam.length > 0) {
      setSearchQuery(queryParam)
    } else {
      defaultSearchQuery.current = ""
    }
  }, [queryParam]);

  const [activePage, setActivePage] = useState(1);
  const [videoTypes, setVideoTypes] = useState<VideoType[]>([]);

  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const { data: videos, isPending, isError } = useSearchVideos({
    limit: videoLimit,
    offset: (activePage - 1) * videoLimit,
    query: searchQuery || defaultSearchQuery.current, // Use state or ref
    types: videoTypes,
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
          showChannel={true}
        />
      </Container>
    </div>
  );
};

export default SearchPage;
