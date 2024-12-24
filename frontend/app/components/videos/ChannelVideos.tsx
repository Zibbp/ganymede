// ChannelVideos.tsx
import { useState } from "react";
import { Channel } from "@/app/hooks/useChannels";
import { useFetchVideosFilter, VideoType } from "@/app/hooks/useVideos";
import useSettingsStore from "@/app/store/useSettingsStore";
import VideoGrid from "./Grid";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";

type Props = {
  channel: Channel;
};

const ChannelVideos = ({ channel }: Props) => {
  const [activePage, setActivePage] = useState(1);
  const [videoTypes, setVideoTypes] = useState<VideoType[]>([]);

  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const { data: videos, isPending, isError } = useFetchVideosFilter({
    limit: videoLimit,
    offset: (activePage - 1) * videoLimit,
    channel_id: channel.id,
    types: videoTypes,
  });

  if (isPending) {
    return <GanymedeLoadingText message="Loading Videos" />;
  }

  if (isError) {
    return <div>Error loading channel</div>;
  }

  return (
    <div>
      <VideoGrid
        videos={videos.data}
        totalPages={videos.pages}
        currentPage={activePage}
        onPageChange={setActivePage}
        isPending={isPending}
        videoLimit={videoLimit}
        onVideoLimitChange={setVideoLimit}
        onVideoTypeChange={setVideoTypes}
        showChannel={false}
      />
    </div>
  );
};

export default ChannelVideos;