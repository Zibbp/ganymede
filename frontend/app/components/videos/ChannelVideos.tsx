// ChannelVideos.tsx
import { Channel } from "@/app/hooks/useChannels";
import { useFetchVideosFilter } from "@/app/hooks/useVideos";
import { useVideoListParams } from "@/app/hooks/useVideoListParams";
import useSettingsStore from "@/app/store/useSettingsStore";
import VideoGrid from "./Grid";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { useTranslations } from "next-intl";

type Props = {
  channel: Channel;
};

const ChannelVideos = ({ channel }: Props) => {
  const t = useTranslations("VideoComponents");
  const { page, videoTypes, sortBy, order, setPage, setVideoTypes, setSortBy, setOrder } = useVideoListParams();

  const videoLimit = useSettingsStore((state) => state.videoLimit);
  const setVideoLimit = useSettingsStore((state) => state.setVideoLimit);

  const { data: videos, isPending, isError } = useFetchVideosFilter({
    limit: videoLimit,
    offset: (page - 1) * videoLimit,
    channel_id: channel.id,
    types: videoTypes,
    sort_by: sortBy,
    order: order,
  });

  if (isPending) {
    return <GanymedeLoadingText message={t('loadingVideos')} />;
  }

  if (isError) {
    return <div>{t('errorLoadingVideos')}</div>;
  }

  return (
    <div>
      <VideoGrid
        videos={videos.data}
        totalCount={videos.total_count}
        totalPages={videos.pages}
        currentPage={page}
        onPageChange={setPage}
        isPending={isPending}
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
    </div>
  );
};

export default ChannelVideos;
