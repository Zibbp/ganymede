"use client";

import { useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { useSearchVideos, VideoType } from "../hooks/useVideos";
import useSettingsStore from "../store/useSettingsStore";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import VideoGrid from "../components/videos/Grid";
import { Center, Container, Title } from "@mantine/core";

const SearchPage = () => {
  const searchParams = useSearchParams();
  const queryParam = searchParams.get("q");

  useEffect(() => {
    document.title = `Search - ${queryParam}`;
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
    return <GanymedeLoadingText message="Loading Videos" />;
  }

  if (isError) {
    return <div>Error loading videos</div>;
  }

  return (
    <div>
      <Center mt={10}>
        <Title>Search</Title>
      </Center>



      <Container size="xl" px="xl" fluid={true}>
        <VideoGrid
          videos={videos.data}
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
