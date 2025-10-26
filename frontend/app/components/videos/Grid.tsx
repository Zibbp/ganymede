import { Box, Center, Group, Pagination, SimpleGrid, ActionIcon, NumberInput, MultiSelect, Text, Select, Flex } from "@mantine/core";
import { IconMinus, IconPlus } from "@tabler/icons-react";
import { useRef, useState, useEffect } from "react";
import type { NumberInputHandlers } from "@mantine/core";
import VideoCard from "./Card";
import { Video, VideoOrder, VideoSortBy, VideoType } from "@/app/hooks/useVideos";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { useTranslations } from "next-intl";

export type VideoGridProps<T extends Video> = {
  videos: T[];
  totalCount: number;
  totalPages: number;
  currentPage: number;
  onPageChange: (page: number) => void;
  isPending?: boolean;
  videoLimit: number;
  onVideoLimitChange: (limit: number) => void;
  onVideoTypeChange: (types: VideoType[]) => void;
  onSortByChange?: (sort: VideoSortBy) => void;
  onOrderChange?: (order: VideoOrder) => void;
  showChannel?: boolean;
  showMenu?: boolean;
  showProgress?: boolean;
};

const VideoGrid = <T extends Video>({
  videos,
  totalCount,
  totalPages,
  currentPage,
  onPageChange,
  isPending = false,
  videoLimit,
  onVideoLimitChange,
  onVideoTypeChange,
  onSortByChange = () => { },
  onOrderChange = () => { },
  showChannel = false,
  showMenu = true,
  showProgress = true,
}: VideoGridProps<T>) => {
  const t = useTranslations("VideoComponents");
  const handlersRef = useRef<NumberInputHandlers>(null);
  // Local state to handle the input value while typing
  const [localLimit, setLocalLimit] = useState(videoLimit);
  const [videoTypes, setVideoTypes] = useState<VideoType[]>([]);
  const [sortBy, setSortBy] = useState<VideoSortBy>(VideoSortBy.Date);
  const [order, setOrder] = useState<VideoOrder>(VideoOrder.Desc);

  useEffect(() => {
    setLocalLimit(videoLimit);
  }, [videoLimit]);

  const handleSetVideoLimit = (value: string | number) => {
    const numValue = Number(value);
    // Update local state immediately
    setLocalLimit(numValue);

    // Only update parent state if it's a valid number within bounds
    if (!isNaN(numValue)) {
      onVideoLimitChange(numValue);
    }
  };

  // Convert the enum VideoType to an array for the multiselector
  const selectorVideoTypes = Object.values(VideoType).map((type) => ({
    value: type,
    label: type.charAt(0).toUpperCase() + type.slice(1),
  }));

  const handleSetSortBy = (value: VideoSortBy | null) => {
    const next = value ?? VideoSortBy.Date; // handle clearable
    setSortBy(next);
    onSortByChange(next);
    onPageChange(1);
  };

  const selectorSortBy = Object.values(VideoSortBy).map((sort) => ({
    value: sort,
    label: (() => {
      const s = String(sort).replace(/_/g, " ");
      return s.charAt(0).toUpperCase() + s.slice(1);
    })(),
  }));

  const handleSetOrder = (value: VideoOrder | null) => {
    const next = value ?? VideoOrder.Desc; // handle clearable
    setOrder(next);
    onOrderChange(next);
    onPageChange(1);
  }

  const selectorOrder = Object.values(VideoOrder).map((order) => ({
    value: order,
    label: order.charAt(0).toUpperCase() + order.slice(1),
  }));

  const convertToVideoTypes = (selectedValues: string[]): VideoType[] => {
    return selectedValues
      .filter((value) => Object.values(VideoType).includes(value as VideoType))
      .map((value) => value as VideoType);
  };

  const handleSetVideoTypes = (selectedStrings: string[]) => {
    const videoTypesArray = convertToVideoTypes(selectedStrings);
    onVideoTypeChange(videoTypesArray)
    setVideoTypes(videoTypesArray)
    onPageChange(1);
  };

  const handleIncrement = () => {
    const newValue = Math.min(localLimit + 24, 120);
    setLocalLimit(newValue);
    onVideoLimitChange(newValue);
  };

  const handleDecrement = () => {
    const newValue = Math.max(localLimit - 24, 24);
    setLocalLimit(newValue);
    onVideoLimitChange(newValue);
  };

  if (isPending) {
    return <GanymedeLoadingText message={t('loadingVideos')} />;
  }

  return (
    <Box>
      <Group justify="space-between" gap="xs" mb="md">
        <Flex gap="xs">
          <MultiSelect
            data={selectorVideoTypes}
            value={videoTypes}
            onChange={(value) => handleSetVideoTypes(value)}
            label={t('filterByLabel')}
            placeholder={t("filterByPlaceholder")}
            clearable
          />
          <Select
            data={selectorSortBy}
            value={sortBy}
            onChange={(value) => handleSetSortBy((value as VideoSortBy) ?? null)} // <— guard
            label={t('sortByLabel')}
            placeholder={t("sortByPlaceholder")}
            clearable
          />
          <Select
            data={selectorOrder}
            value={order}
            onChange={(value) => handleSetOrder((value as VideoOrder) ?? null)} // <— guard
            label={t('orderByLabel')}
            placeholder={t("orderByPlaceholder")}
            w={100}
          />
        </Flex>
        <div>
          <Text>{t('videosCount', { count: totalCount.toLocaleString() })}</Text>
        </div>
      </Group>

      <SimpleGrid
        cols={{ base: 1, sm: 2, md: 3, lg: 4, xl: 5, xxl: 6 }}
        spacing="xs"
        verticalSpacing="xs"
      >
        {videos.map((video) => (
          <VideoCard
            key={video.id}
            video={video}
            showChannel={showChannel}
            showMenu={showMenu}
            showProgress={showProgress}
          />
        ))}
      </SimpleGrid>

      <div>
        <Center>
          <Pagination
            value={currentPage}
            onChange={onPageChange}
            total={totalPages}
            color="violet"
            size="lg"
            withEdges
          />
        </Center>
        <Center mt={5}>
          <Group>
            <ActionIcon
              size="lg"
              variant="default"
              onClick={handleDecrement}
            >
              <IconMinus style={{ width: '70%', height: '70%' }} stroke={1.5} />
            </ActionIcon>

            <NumberInput
              hideControls
              value={localLimit}
              onChange={handleSetVideoLimit}
              handlersRef={handlersRef}
              max={120}
              min={24}
              step={24}
              styles={{ input: { width: 54, textAlign: "center" } }}
              clampBehavior="strict"
              allowDecimal={false}
            />

            <ActionIcon
              size="lg"
              variant="default"
              onClick={handleIncrement}
            >
              <IconPlus style={{ width: '70%', height: '70%' }} stroke={1.5} />
            </ActionIcon>
          </Group>
        </Center>
      </div>
    </Box>
  );
};

export default VideoGrid;