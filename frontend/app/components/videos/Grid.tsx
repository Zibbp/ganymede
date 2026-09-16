import { Box, Button, Center, Checkbox, Group, Menu, Modal, Pagination, SimpleGrid, MultiSelect, Text, Select, Flex } from "@mantine/core";
import { IconHourglassEmpty, IconHourglassHigh, IconLock, IconLockOpen, IconMovie, IconPhoto, IconPlaylistAdd, IconTrash } from "@tabler/icons-react";
import { useState, useMemo } from "react";
import VideoCard from "./Card";
import { useGenerateSpriteThumbnails, useGenerateStaticThumbnail, useLockVideo, Video, VideoOrder, VideoSortBy, VideoType } from "@/app/hooks/useVideos";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { useTranslations } from "next-intl";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useDeletePlayback, useMarkVideoAsWatched } from "@/app/hooks/usePlayback";
import { showNotification } from "@mantine/notifications";
import { useDisclosure } from "@mantine/hooks";
import PlaylistBulkAddModalContent from "../playlist/BulkAddModalContent";
import MultiDeleteVideoModalContent from "../admin/video/MultiDeleteModalContent";
import useAuthStore from "@/app/store/useAuthStore";
import { UserRole } from "@/app/hooks/useAuthentication";
import { useQueryClient } from "@tanstack/react-query";

// Selectable page sizes. Same values the previous +/- 24 stepper could reach.
const VIDEO_LIMIT_OPTIONS = [24, 48, 72, 96, 120];

export type VideoGridProps<T extends Video> = {
  videos: T[];
  totalCount: number;
  totalPages: number;
  currentPage: number;
  onPageChange: (page: number) => void;
  isPending?: boolean;
  videoLimit: number;
  onVideoLimitChange: (limit: number) => void;
  videoTypes: VideoType[];
  onVideoTypeChange: (types: VideoType[]) => void;
  sortBy?: VideoSortBy;
  onSortByChange?: (sort: VideoSortBy) => void;
  order?: VideoOrder;
  onOrderChange?: (order: VideoOrder) => void;
  showChannel?: boolean;
  showMenu?: boolean;
  showProgress?: boolean;
  enableSelection?: boolean;
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
  videoTypes,
  onVideoTypeChange,
  sortBy = VideoSortBy.Date,
  onSortByChange = () => { },
  order = VideoOrder.Desc,
  onOrderChange = () => { },
  showChannel = false,
  showMenu = true,
  showProgress = true,
  enableSelection = true,
}: VideoGridProps<T>) => {
  const t = useTranslations("VideoComponents");
  const axiosPrivate = useAxiosPrivate();
  const queryClient = useQueryClient();
  const { hasPermission } = useAuthStore();
  const canBulkManage = hasPermission(UserRole.Archiver);
  const canBulkDelete = hasPermission(UserRole.Admin);
  const selectionEnabled = enableSelection && canBulkManage;
  const [selectedVideos, setSelectedVideos] = useState<Record<string, T>>({});
  const [bulkActionLoading, setBulkActionLoading] = useState(false);
  const [bulkMenuOpened, setBulkMenuOpened] = useState(false);
  const [playlistModalOpened, { open: openPlaylistModal, close: closePlaylistModal }] = useDisclosure(false);
  const [multiDeleteModalOpened, { open: openMultiDeleteModal, close: closeMultiDeleteModal }] = useDisclosure(false);
  const selectedVideoList = useMemo(() => Object.values(selectedVideos), [selectedVideos]);
  const selectedVideoIdSet = useMemo(
    () => new Set(selectedVideoList.map((video) => video.id)),
    [selectedVideoList]
  );

  const markAsWatchedMutate = useMarkVideoAsWatched();
  const deletePlaybackMutate = useDeletePlayback();
  const lockVideoMutate = useLockVideo();
  const generateStaticThumbnailMutate = useGenerateStaticThumbnail();
  const generateSpriteThumbnailsMutate = useGenerateSpriteThumbnails();

  // Keep a stored page size that is not one of the options selectable, so the select is never empty
  const selectorVideoLimit = (
    VIDEO_LIMIT_OPTIONS.includes(videoLimit)
      ? VIDEO_LIMIT_OPTIONS
      : [...VIDEO_LIMIT_OPTIONS, videoLimit].sort((a, b) => a - b)
  ).map((limit) => String(limit));

  const handleSetVideoLimit = (value: string | null) => {
    if (value === null) return;
    onVideoLimitChange(Number(value));
    // A different page size changes what is on each page, so start over at the first page
    onPageChange(1);
  };

  // Convert the enum VideoType to an array for the multiselector
  const selectorVideoTypes = Object.values(VideoType).map((type) => ({
    value: type,
    label: t(`enums.VideoType.${type}`),
  }));

  const handleSetSortBy = (value: VideoSortBy | null) => {
    const next = value ?? VideoSortBy.Date; // handle clearable
    onSortByChange(next);
  };

  const selectorSortBy = Object.values(VideoSortBy).map((sort) => ({
    value: sort,
    label: t(`enums.VideoSortBy.${sort}`),
  }));

  const handleSetOrder = (value: VideoOrder | null) => {
    const next = value ?? VideoOrder.Desc; // handle clearable
    onOrderChange(next);
  }

  const selectorOrder = Object.values(VideoOrder).map((order) => ({
    value: order,
    label: t(`enums.VideoOrder.${order}`),
  }));

  const convertToVideoTypes = (selectedValues: string[]): VideoType[] => {
    return selectedValues
      .filter((value) => Object.values(VideoType).includes(value as VideoType))
      .map((value) => value as VideoType);
  };

  const handleSetVideoTypes = (selectedStrings: string[]) => {
    const videoTypesArray = convertToVideoTypes(selectedStrings);
    onVideoTypeChange(videoTypesArray)
  };

  const handleVideoSelectionChange = (video: T, selected: boolean) => {
    setSelectedVideos((current) => {
      const next = { ...current };
      if (selected) {
        next[video.id] = video;
      } else {
        delete next[video.id];
      }
      return next;
    });
  };

  const handleSelectAllOnPage = (selected: boolean) => {
    setSelectedVideos((current) => {
      const next = { ...current };
      videos.forEach((video) => {
        if (selected) {
          next[video.id] = video;
        } else {
          delete next[video.id];
        }
      });
      return next;
    });
  };

  const runBulkOperation = async (
    operation: (video: T) => Promise<unknown>,
    successMessage: string,
    onSuccess?: () => Promise<void> | void
  ) => {
    if (selectedVideoList.length === 0) return;
    try {
      setBulkActionLoading(true);
      const results = await Promise.allSettled(
        selectedVideoList.map((video) => Promise.resolve().then(() => operation(video)))
      );
      const successCount = results.filter((result) => result.status === "fulfilled").length;
      const failedResults = results.filter((result) => result.status === "rejected");
      const failureCount = failedResults.length;

      if (onSuccess && successCount > 0) {
        await onSuccess();
      }

      failedResults.forEach((result) => {
        console.error(result.reason);
      });

      showNotification({
        title:
          failureCount === 0
            ? successMessage
            : failureCount === results.length
              ? t("error")
              : successMessage,
        message:
          failureCount === 0
            ? `${successMessage} (${successCount}/${results.length})`
            : `${successMessage} (${successCount}/${results.length}) • ${failureCount} failed`,
        color:
          failureCount === 0
            ? undefined
            : failureCount === results.length
              ? "red"
              : "yellow",
      });
    } catch (error) {
      showNotification({
        title: t("error"),
        message: error instanceof Error ? error.message : String(error),
        color: "red",
      });
      console.error(error);
    } finally {
      setBulkActionLoading(false);
    }
  };

  const handleMarkVideosAsWatched = async () => {
    if (bulkActionLoading) return;
    setBulkMenuOpened(false);
    await runBulkOperation(
      (video) =>
        markAsWatchedMutate.mutateAsync({
          axiosPrivate,
          videoId: video.id,
          invalidatePlaybackQuery: false,
        }),
      t("markedVideosAsWatchedNotification"),
      async () => {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["playback-data"] }),
          queryClient.invalidateQueries({ queryKey: ["playback-videos"] }),
        ]);
      }
    );
  };

  const handleMarkVideosAsUnwatched = async () => {
    if (bulkActionLoading) return;
    setBulkMenuOpened(false);
    await runBulkOperation(
      (video) =>
        deletePlaybackMutate.mutateAsync({
          axiosPrivate,
          videoId: video.id,
          invalidatePlaybackQuery: false,
        }),
      t("markedVideosAsUnwatchedNotification"),
      async () => {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["playback-data"] }),
          queryClient.invalidateQueries({ queryKey: ["playback-videos"] }),
        ]);
      }
    );
  };

  const handleLockVideos = async (locked: boolean) => {
    if (bulkActionLoading) return;
    setBulkMenuOpened(false);
    await runBulkOperation(
      (video) =>
        lockVideoMutate.mutateAsync({
          axiosPrivate,
          videoId: video.id,
          locked,
          invalidateVideoQueries: false,
        }),
      t("videosLockedNotification", { status: locked ? t("locked") : t("unlocked") }),
      async () => {
        await Promise.all([
          queryClient.invalidateQueries({ queryKey: ["videos"] }),
          queryClient.invalidateQueries({ queryKey: ["channel_videos"] }),
          queryClient.invalidateQueries({ queryKey: ["playlist_videos"] }),
          queryClient.invalidateQueries({ queryKey: ["search"] }),
        ]);
      }
    );
  };

  const handleGenerateStaticThumbnails = async () => {
    if (bulkActionLoading) return;
    setBulkMenuOpened(false);
    await runBulkOperation(
      (video) =>
        generateStaticThumbnailMutate.mutateAsync({
          axiosPrivate,
          videoId: video.id,
        }),
      t("bulkGenerateStaticThumbnailsNotification")
    );
  };

  const handleGenerateSpriteThumbnails = async () => {
    if (bulkActionLoading) return;
    setBulkMenuOpened(false);
    await runBulkOperation(
      (video) =>
        generateSpriteThumbnailsMutate.mutateAsync({
          axiosPrivate,
          videoId: video.id,
        }),
      t("bulkGenerateSpriteThumbnailsNotification")
    );
  };

  const handleCloseMultiDeleteModal = () => {
    closeMultiDeleteModal();
    setSelectedVideos({});
  };

  const handleOpenPlaylistModal = () => {
    if (bulkActionLoading) return;
    setBulkMenuOpened(false);
    openPlaylistModal();
  };

  const handleOpenMultiDeleteModal = () => {
    if (bulkActionLoading) return;
    setBulkMenuOpened(false);
    openMultiDeleteModal();
  };

  if (isPending) {
    return <GanymedeLoadingText message={t('loadingVideos')} />;
  }

  const allVisibleSelected = videos.length > 0 && videos.every((video) => selectedVideoIdSet.has(video.id));
  const someVisibleSelected = videos.some((video) => selectedVideoIdSet.has(video.id)) && !allVisibleSelected;

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
            onChange={(value) => handleSetSortBy((value as VideoSortBy) ?? null)}
            label={t('sortByLabel')}
            placeholder={t("sortByPlaceholder")}
            clearable
          />
          <Select
            data={selectorOrder}
            value={order}
            onChange={(value) => handleSetOrder((value as VideoOrder) ?? null)}
            label={t('orderByLabel')}
            placeholder={t("orderByPlaceholder")}
            w={200}
          />
        </Flex>
        <div>
          <Text>{t('videosCount', { count: totalCount.toLocaleString() })}</Text>
        </div>
      </Group>

      {selectionEnabled && (
        <Group justify="space-between" mb="sm">
          <Group gap="sm">
            <Checkbox
              label={t("selectAllOnPage")}
              checked={allVisibleSelected}
              indeterminate={someVisibleSelected}
              onChange={(event) => handleSelectAllOnPage(event.currentTarget.checked)}
              disabled={videos.length === 0 || bulkActionLoading}
            />
            <Text size="sm">{t("selectedVideosCount", { count: selectedVideoList.length })}</Text>
          </Group>
          <Group gap="xs">
            <Button
              variant="default"
              onClick={() => setSelectedVideos({})}
              disabled={selectedVideoList.length === 0 || bulkActionLoading}
            >
              {t("clearSelectionButton")}
            </Button>
            <Menu shadow="md" width={270} opened={bulkMenuOpened} onChange={setBulkMenuOpened}>
              <Menu.Target>
                <Button loading={bulkActionLoading} disabled={selectedVideoList.length === 0}>
                  {t("bulkActionsButton")}
                </Button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item
                  onClick={handleMarkVideosAsWatched}
                  leftSection={<IconHourglassHigh size={14} />}
                  disabled={bulkActionLoading}
                >
                  {t("bulkActionMenu.markAsWatched")}
                </Menu.Item>
                <Menu.Item
                  onClick={handleMarkVideosAsUnwatched}
                  leftSection={<IconHourglassEmpty size={14} />}
                  disabled={bulkActionLoading}
                >
                  {t("bulkActionMenu.markAsUnwatched")}
                </Menu.Item>
                <Menu.Item
                  onClick={() => handleLockVideos(true)}
                  leftSection={<IconLock size={14} />}
                  disabled={bulkActionLoading}
                >
                  {t("bulkActionMenu.lock")}
                </Menu.Item>
                <Menu.Item
                  onClick={() => handleLockVideos(false)}
                  leftSection={<IconLockOpen size={14} />}
                  disabled={bulkActionLoading}
                >
                  {t("bulkActionMenu.unlock")}
                </Menu.Item>
                <Menu.Item
                  onClick={handleGenerateStaticThumbnails}
                  leftSection={<IconPhoto size={14} />}
                  disabled={bulkActionLoading}
                >
                  {t("bulkActionMenu.regenerateThumbnails")}
                </Menu.Item>
                <Menu.Item
                  onClick={handleGenerateSpriteThumbnails}
                  leftSection={<IconMovie size={14} />}
                  disabled={bulkActionLoading}
                >
                  {t("bulkActionMenu.generateSpriteThumbnails")}
                </Menu.Item>
                <Menu.Item
                  onClick={handleOpenPlaylistModal}
                  leftSection={<IconPlaylistAdd size={14} />}
                  disabled={bulkActionLoading}
                >
                  {t("bulkActionMenu.playlists")}
                </Menu.Item>
                {canBulkDelete && (
                  <>
                    <Menu.Divider />
                    <Menu.Item
                      color="red"
                      onClick={handleOpenMultiDeleteModal}
                      leftSection={<IconTrash size={14} />}
                      disabled={bulkActionLoading}
                    >
                      {t("bulkActionMenu.delete")}
                    </Menu.Item>
                  </>
                )}
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Group>
      )}

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
            selectable={selectionEnabled}
            selected={selectedVideoIdSet.has(video.id)}
            onSelectionChange={(selected) => handleVideoSelectionChange(video, selected)}
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
          <Select
            data={selectorVideoLimit}
            value={String(videoLimit)}
            onChange={handleSetVideoLimit}
            allowDeselect={false}
            checkIconPosition="right"
            w={90}
          />
        </Center>
      </div>

      <Modal
        opened={playlistModalOpened}
        onClose={closePlaylistModal}
        title={t("bulkAddToPlaylistModalTitle")}
      >
        {selectedVideoList.length > 0 && (
          <PlaylistBulkAddModalContent videos={selectedVideoList} handleClose={closePlaylistModal} />
        )}
      </Modal>

      <Modal
        opened={multiDeleteModalOpened}
        onClose={closeMultiDeleteModal}
        title={t("bulkDeleteVideosModalTitle")}
      >
        {selectedVideoList.length > 0 && (
          <MultiDeleteVideoModalContent videos={selectedVideoList} handleClose={handleCloseMultiDeleteModal} />
        )}
      </Modal>
    </Box>
  );
};

export default VideoGrid;
