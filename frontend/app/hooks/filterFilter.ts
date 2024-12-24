import { Channel } from "@/app/hooks/useChannels";
import { useFetchVideosFilter, VideoType } from "@/app/hooks/useVideos";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useState } from "react";

export type FilterParams = {
  limit: number;
  page: number;
  types: VideoType[];
};

export const useFilteredVideos = (channel: Channel) => {
  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const [filterParams, setFilterParams] = useState<FilterParams>({
    limit: videoLimit,
    page: 1,
    types: [],
  });

  const {
    data: videos,
    isPending,
    isError,
  } = useFetchVideosFilter({
    limit: filterParams.limit,
    offset: (filterParams.page - 1) * filterParams.limit,
    types: filterParams.types,
    channel_id: channel.id,
  });

  const handleFilterChange = (newParams: Partial<FilterParams>) => {
    setFilterParams((current) => ({
      ...current,
      ...newParams,
    }));
  };

  return {
    videos,
    isPending,
    isError,
    filterParams,
    handleFilterChange,
    setVideoLimit,
  };
};
