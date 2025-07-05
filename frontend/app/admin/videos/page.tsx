"use client"
import { ActionIcon, Container, Group, TextInput, Text, Title, Box, Button, Drawer, Modal, Tooltip, Flex, Menu } from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useEffect, useState } from "react";
import sortBy from "lodash/sortBy";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { IconPencil, IconSearch, IconTrash } from "@tabler/icons-react";
import dayjs from "dayjs";

import classes from "./AdminVideosPage.module.css"
import { useGetVideosNoPaginate, useLockVideo, Video } from "@/app/hooks/useVideos";
import AdminVideoDrawerContent, { VideoEditMode } from "@/app/components/admin/video/DrawerContent";
import DeleteVideoModalContent from "@/app/components/admin/video/DeleteModalContent";
import MultiDeleteVideoModalContent from "@/app/components/admin/video/MultiDeleteModalContent";
import { useTranslations } from "next-intl";
import { formatBytes, usePageTitle } from "@/app/util/util";
import { useDeletePlayback, useMarkVideoAsWatched } from "@/app/hooks/usePlayback";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { showNotification } from "@mantine/notifications";
import PlaylistBulkAddModalContent from "@/app/components/playlist/BulkAddModalContent";

const AdminVideosPage = () => {
  const t = useTranslations('AdminVideosPage')
  usePageTitle(t('title'))
  const axiosPrivate = useAxiosPrivate()
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [records, setRecords] = useState<Video[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<Video>>({
    columnAccessor: "name",
    direction: "asc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [activeVideo, setActiveVideo] = useState<Video | null>(null);
  const [drawerEditMode, setDrawerEditMode] = useState<VideoEditMode>(VideoEditMode.Create)

  const [videoDrawerOpened, { open: openVideoDrawer, close: closeVideoDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [multiDeleteModalOpened, { open: openMultiDeleteModal, close: closeMultiDeleteModal }] = useDisclosure(false);
  const [multiVideoPlaylistModal, { open: openMultiVideoPlaylistModal, close: closeMultiVideoPlaylistModal }] = useDisclosure(false);
  const [activeVideos, setActiveVideos] = useState<Video[] | null>([]);
  const [bulkActionLoading, setBulkActionLoading] = useState<boolean>(false);

  const markAsWatchedMutate = useMarkVideoAsWatched()
  const deletePlaybackMutate = useDeletePlayback()
  const lockVideoMutate = useLockVideo()

  const { data: videos, isPending, isError } = useGetVideosNoPaginate()

  useEffect(() => {
    if (!videos) return;

    let filteredData = [...videos] as Video[];

    // Apply search filter
    if (debouncedQuery) {
      filteredData = filteredData.filter((video) => {
        return (
          video.id?.toString().includes(debouncedQuery) ||
          video.title?.toLowerCase().includes(debouncedQuery.toLowerCase()) ||
          video.ext_id?.toString().includes(debouncedQuery.toLowerCase()) ||
          video.streamed_at?.toString().includes(debouncedQuery.toLowerCase()) ||
          video.edges?.channel?.name?.toLowerCase().includes(debouncedQuery.toLowerCase()) ||
          video.edges?.channel?.id?.toString().includes(debouncedQuery.toLowerCase())
        );
      });
    }

    // Apply sorting
    let sortedData;
    if (sortStatus.columnAccessor === "storage_size_bytes") {
      sortedData = [...filteredData].sort((a, b) => {
        const aVal = a.storage_size_bytes ?? 0;
        const bVal = b.storage_size_bytes ?? 0;
        return aVal - bVal;
      });
    } else {
      sortedData = sortBy(filteredData, sortStatus.columnAccessor);
    }
    filteredData = sortStatus.direction === "desc" ? sortedData.reverse() : sortedData;

    // Apply pagination
    const from = (page - 1) * perPage;
    const to = from + perPage;
    setRecords(filteredData.slice(from, to));

    if (!initialRecords) {
      setInitialRecords(true);
    }
  }, [videos, page, perPage, sortStatus, debouncedQuery, initialRecords]);

  const openDrawer = (mode: VideoEditMode, video: Video) => {
    setActiveVideo(video);
    setDrawerEditMode(VideoEditMode.Edit)
    openVideoDrawer()
  };

  const handleDeleteModal = (video: Video) => {
    setActiveVideo(video);
    openDeleteModal()
  };

  const handleMultiDeleteModalCallback = () => {
    closeMultiDeleteModal()
    setActiveVideos([])
  }

  const handleMultiVideoPlaylistModalCallback = () => {
    closeMultiVideoPlaylistModal()
  }

  // Bulk mark videos as watched
  const handleMarkVideosAsWatched = async () => {
    try {
      setBulkActionLoading(true);

      if (activeVideos && activeVideos.length > 0) {
        await Promise.all(
          activeVideos.map((video) =>
            markAsWatchedMutate.mutateAsync({
              axiosPrivate,
              videoId: video.id,
            })
          )
        );

        showNotification({
          message: t('markedVideosAsWatchedNotification')
        });
      }
    } catch (error) {
      showNotification({
        title: t('notificationError'),
        message: error instanceof Error ? error.message : String(error),
      });
      console.error(error);
    } finally {
      setBulkActionLoading(false);
    }
  };

  // Bulk mark videos as unwatched
  const handleMarkVideosAsUnWatched = async () => {
    try {
      setBulkActionLoading(true);

      if (activeVideos && activeVideos.length > 0) {
        await Promise.all(
          activeVideos.map((video) =>
            deletePlaybackMutate.mutateAsync({
              axiosPrivate,
              videoId: video.id,
            })
          )
        );

        showNotification({
          message: t('markedVideosAsUnWatchedNotification')
        });
      }
    } catch (error) {
      showNotification({
        title: t('notificationError'),
        message: error instanceof Error ? error.message : String(error),
      });
      console.error(error);
    } finally {
      setBulkActionLoading(false);
    }
  };

  // Bulk mark videos as locked or unlocked
  const handleLockVideos = async (locked: boolean) => {
    try {
      setBulkActionLoading(true);

      if (activeVideos && activeVideos.length > 0) {
        await Promise.all(
          activeVideos.map((video) =>
            lockVideoMutate.mutateAsync({
              axiosPrivate,
              videoId: video.id,
              locked: locked
            })
          )
        );

        showNotification({
          message: `${t('videosLockedNotification', { status: locked ? t('locked') : t('unlocked') })}`
        });
      }
    } catch (error) {
      showNotification({
        title: t('notificationError'),
        message: error instanceof Error ? error.message : String(error),
      });
      console.error(error);
    } finally {
      setBulkActionLoading(false);
    }
  };

  if (isPending) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isError) return <div>{t('error')}</div>

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2} >
          <Title>{t('header')}</Title>
          <Box>
            {(activeVideos && activeVideos.length >= 1) && (
              <>
                <Menu shadow="md" width={200}>
                  <Menu.Target>
                    <Button loading={bulkActionLoading}>{t('bulkActionsButton')}</Button>
                  </Menu.Target>

                  <Menu.Dropdown>
                    <Menu.Item onClick={handleMarkVideosAsWatched}>
                      {t('bulkActionMenu.markAsWatched')}
                    </Menu.Item>
                    <Menu.Item onClick={handleMarkVideosAsUnWatched}>
                      {t('bulkActionMenu.markAsUnwatched')}
                    </Menu.Item>
                    <Menu.Item onClick={() => handleLockVideos(true)}>
                      {t('bulkActionMenu.lock')}
                    </Menu.Item>
                    <Menu.Item onClick={() => handleLockVideos(false)}>
                      {t('bulkActionMenu.unlock')}
                    </Menu.Item>
                    <Menu.Item onClick={() => openMultiVideoPlaylistModal()}>
                      {t('bulkActionMenu.playlist')}
                    </Menu.Item>


                  </Menu.Dropdown>
                </Menu>

                <Button
                  mx={10}
                  leftSection={<IconTrash size={16} />}
                  color="red"
                  disabled={!activeVideos.length}
                  onClick={() => {
                    openMultiDeleteModal();
                  }}
                >
                  {activeVideos.length
                    ? `${t('delete.delete')} ${activeVideos.length === 1
                      ? t('delete.one')
                      : `${activeVideos.length} ${t('delete.many')}`
                    }`
                    : t('delete.select')}
                </Button>
              </>
            )}

            <Button
              onClick={() => {
                setDrawerEditMode(VideoEditMode.Create)
                setActiveVideo(null)
                openVideoDrawer()
              }}
              mr={5}
              variant="default"
            >
              {t('manuallyAddButton')}
            </Button>
          </Box>
        </Group>



        <Box mt={5}>
          <div>
            <TextInput
              placeholder={t('searchPlaceholder')}
              leftSection={<IconSearch size={16} />}
              value={query}
              onChange={(e) => setQuery(e.currentTarget.value)}
              mb={10}
            />

          </div>
          <DataTable<Video>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover={true}
            records={records}
            columns={[
              {
                accessor: "id",
                title: t('columns.id'),
                width: 90,
                render: ({ id }) => (
                  <Tooltip label={id}>
                    <Text lineClamp={1}>{id}</Text>
                  </Tooltip>
                ),
              },
              {
                accessor: "ext_id", title: t('columns.extId'),
                render: ({ ext_id }) => (
                  <Tooltip label={ext_id}>
                    <Text lineClamp={1}>{ext_id}</Text>
                  </Tooltip>
                ),
              },
              {
                accessor: "edges.channel.display_name",
                title: t('columns.channel'),
                sortable: true,
              },
              { accessor: "title", title: t('columns.title'), sortable: true, },
              { accessor: "type", title: t('columns.type'), sortable: true },
              {
                accessor: "locked", title: t('columns.locked'), sortable: true,
                render: ({ locked }) => {
                  return locked ? "✅" : "❌";
                },
              },
              {
                accessor: "storage_size_bytes", title: t('columns.storageSize'), sortable: true,
                render: ({ storage_size_bytes }) => {
                  return formatBytes(storage_size_bytes ?? 0, 2);
                },
              },
              {
                accessor: "streamed_at",
                title: t('columns.streamedAt'),
                sortable: true,
                render: ({ streamed_at }) => (
                  <div title={`${new Date(streamed_at).toLocaleString()}`}>
                    {dayjs(streamed_at).format("YYYY/MM/DD")}
                  </div>
                ),
              },
              {
                accessor: "created_at",
                title: t('columns.archivedAt'),
                sortable: true,
                render: ({ created_at }) => (
                  <div title={`${new Date(created_at).toLocaleString()}`}>
                    {dayjs(created_at).format("YYYY/MM/DD")}
                  </div>
                ),
              },
              {
                accessor: "actions",
                title: t('columns.actions'),
                render: (video) => (
                  <Flex>
                    <ActionIcon
                      mx={2}
                      onClick={() => openDrawer(VideoEditMode.Edit, video)}
                      className={classes.actionButton}
                      variant="light"
                    >
                      <IconPencil size={18} />
                    </ActionIcon>
                    <ActionIcon
                      mx={2}
                      onClick={() => handleDeleteModal(video)}
                      className={classes.actionButton}
                      variant="light" color="red"
                    >
                      <IconTrash size={18} />
                    </ActionIcon>
                  </Flex>
                ),
              },
            ]}
            totalRecords={videos?.length ?? 0}
            page={page}
            recordsPerPage={perPage}
            onPageChange={(p) => setPage(p)}
            recordsPerPageOptions={[20, 40, 100]}
            onRecordsPerPageChange={setPerPage}
            sortStatus={sortStatus}
            onSortStatusChange={setSortStatus}
            selectedRecords={activeVideos ?? []}
            onSelectedRecordsChange={setActiveVideos}
          />
        </Box>
      </Container>

      <Drawer opened={videoDrawerOpened} onClose={closeVideoDrawer} position="right" size="lg" title={t('drawer')}>
        <AdminVideoDrawerContent mode={drawerEditMode} video={activeVideo} handleClose={closeVideoDrawer} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('deleteModal')}>
        {activeVideo && (
          <DeleteVideoModalContent video={activeVideo} handleClose={closeDeleteModal} />
        )}
      </Modal>
      <Modal opened={multiDeleteModalOpened} onClose={closeMultiDeleteModal} title={t('multiDeleteModal')}>
        {activeVideos && (
          <MultiDeleteVideoModalContent videos={activeVideos} handleClose={handleMultiDeleteModalCallback} />
        )}
      </Modal>

      <Modal opened={multiVideoPlaylistModal} onClose={closeMultiVideoPlaylistModal} title={t('addVideosToPlaylistModalTitle')}>
        {activeVideos && (
          <PlaylistBulkAddModalContent videos={activeVideos} handleClose={handleMultiVideoPlaylistModalCallback} />
        )}
      </Modal>

    </div >
  );
}

export default AdminVideosPage;