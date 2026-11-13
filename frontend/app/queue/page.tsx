"use client"
import ArchiveStatusBadge from "@/app/components/videos/ArchiveStatusBadge";
import { useEffect, useRef, useState } from "react";
import { ApiResponse, useAxiosPrivate } from "../hooks/useAxios";
import { Queue, useGetQueueItems, useStopQueueItem } from "../hooks/useQueue";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import { DataTable } from "mantine-datatable";
import { Tooltip, Text, ActionIcon, Container, Modal, Switch, Button, Group, Box } from "@mantine/core";
import { IconEye, IconSquareX } from "@tabler/icons-react";
import Link from "next/link";
import { useDisclosure } from "@mantine/hooks";
import { Video, useDeleteVideo } from "../hooks/useVideos";
import { isArchiveActive } from "../util/archiveStatus";
import { useBlockVideo } from "../hooks/useBlockedVideos";
import { useQueryClient } from "@tanstack/react-query";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { usePageTitle } from "../util/util";
import useSettingsStore from "../store/useSettingsStore";

// Per queue item record of the bulk cancel stages that already succeeded, so a retry does not repeat them.
type BulkCancelProgress = { stopped: boolean; deleted: boolean; blocked: boolean };

// Live archive finalization can take a while after the capture is stopped.
const LIVE_SETTLE_POLL_MS = 3000;
const LIVE_SETTLE_TIMEOUT_MS = 10 * 60 * 1000;

const QueuePage = () => {
  const t = useTranslations("QueuePage");
  const statusText = useTranslations('ArchiveStatus');
  const miscT = useTranslations("MiscComponents");
  usePageTitle(t('pageTitle'))

  const settingsAdminItemsPerPage = useSettingsStore((state) => state.adminItemsPerPage);
  const setSettingsAdminItemsPerPage = useSettingsStore((state) => state.setAdminItemsPerPage)

  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(settingsAdminItemsPerPage);
  const [records, setRecords] = useState<Queue[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [activeQueue, setActiveQueue] = useState<Queue | null>(null);

  // delete state
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [cancelQueueLoading, setCancelQueueLoading] = useState(false);
  const [deleteVideoAndFiles, setDeleteVideoAndFiles] = useState(false);
  const [blockVideoId, setBlockVideoId] = useState(false);

  // bulk cancel state
  const [selectedQueueItems, setSelectedQueueItems] = useState<Queue[]>([]);
  const [multiDeleteModalOpened, { open: openMultiDeleteModal, close: closeMultiDeleteModal }] = useDisclosure(false);
  const [multiCancelLoading, setMultiCancelLoading] = useState(false);
  const [multiDeleteVideoAndFiles, setMultiDeleteVideoAndFiles] = useState(false);
  const [multiBlockVideoId, setMultiBlockVideoId] = useState(false);
  const bulkProgress = useRef<Map<string, BulkCancelProgress>>(new Map());
  const isMounted = useRef(true);

  const axiosPrivate = useAxiosPrivate()
  const queryClient = useQueryClient()

  const stopQueueItemMutate = useStopQueueItem()
  const deleteVideoMutate = useDeleteVideo()
  const blockVideoMutate = useBlockVideo()

  const { data: queueItems, isPending: queueIsPending, isError: queueIsError } = useGetQueueItems(axiosPrivate, true)

  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
    };
  }, []);

  useEffect(() => {
    setPerPage(settingsAdminItemsPerPage);
  }, [settingsAdminItemsPerPage]);

  useEffect(() => {
    if (queueItems && !initialRecords) {
      setRecords(queueItems.slice(0, perPage));
      setInitialRecords(true);
    }
    if (queueItems) {
      const from = (page - 1) * perPage;
      const to = from + perPage;
      setRecords(queueItems.slice(from, to));
    }
  }, [queueItems, page, perPage, initialRecords]);

  const cancelQueueItem = async () => {
    try {

      if (activeQueue == null) return;

      setCancelQueueLoading(true)

      // cancel queue item
      stopQueueItemMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        id: activeQueue.id
      })

      // delete video and files if requested
      if (deleteVideoAndFiles) {
        deleteVideoMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          id: activeQueue.edges.vod.id,
          deleteFiles: true
        })
      }

      // block video id if requested
      if (blockVideoId) {
        blockVideoMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          id: activeQueue.edges.vod.ext_id
        })
      }

      queryClient.invalidateQueries({ queryKey: ["queue"] })

      showNotification({
        message: t('cancel')
      })

      closeDeleteModal()
    } catch (error) {
      console.log(error)
    } finally {
      setCancelQueueLoading(false)
      setActiveQueue(null)
    }

  }

  // Stopping a live archive only signals the capture worker; finalization and post-processing continue
  // afterwards. Wait for the video to leave its active states before its files are deleted.
  const waitForVideoToSettle = async (videoId: string) => {
    const deadline = Date.now() + LIVE_SETTLE_TIMEOUT_MS;
    while (isMounted.current && Date.now() < deadline) {
      // bound the request by the remaining time so a stalled response cannot outlive the deadline
      const response = await axiosPrivate.get<ApiResponse<Video>>(`/api/v1/vod/${videoId}`, {
        timeout: Math.max(1, deadline - Date.now()),
      });
      if (!isArchiveActive(response.data.data.status)) return;
      await new Promise((resolve) => setTimeout(resolve, LIVE_SETTLE_POLL_MS));
    }
    throw new Error(`Timed out waiting for video ${videoId} to finish processing`);
  };

  const processQueueItem = async (queue: Queue) => {
    let progress = bulkProgress.current.get(queue.id);
    if (!progress) {
      progress = { stopped: false, deleted: false, blocked: false };
      bulkProgress.current.set(queue.id, progress);
    }

    // cancel queue item
    if (!progress.stopped) {
      await stopQueueItemMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        id: queue.id
      })
      progress.stopped = true
    }

    // delete video and files if requested
    if (multiDeleteVideoAndFiles && !progress.deleted) {
      if (queue.live_archive) {
        await waitForVideoToSettle(queue.edges.vod.id)
      }
      await deleteVideoMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        id: queue.edges.vod.id,
        deleteFiles: true
      })
      progress.deleted = true
    }

    // block video id if requested (not applicable to live archives)
    if (multiBlockVideoId && !queue.live_archive && !progress.blocked) {
      await blockVideoMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        id: queue.edges.vod.ext_id
      })
      progress.blocked = true
    }
  }

  const handleSelectedQueueItemsChange = (items: Queue[]) => {
    // drop progress of items that are no longer selected
    const selectedIds = new Set(items.map((item) => item.id))
    for (const id of Array.from(bulkProgress.current.keys())) {
      if (!selectedIds.has(id)) bulkProgress.current.delete(id)
    }
    setSelectedQueueItems(items)
  }

  const cancelSelectedQueueItems = async () => {
    if (selectedQueueItems.length === 0) return;

    setMultiCancelLoading(true)
    try {
      // wait for every item to settle so loading is only cleared once nothing is still running
      const results = await Promise.allSettled(selectedQueueItems.map(processQueueItem))
      const failedItems = selectedQueueItems.filter((_, index) => results[index].status === "rejected")

      results.forEach((result) => {
        if (result.status === "rejected") console.log(result.reason)
      })

      queryClient.invalidateQueries({ queryKey: ["queue"] })

      if (failedItems.length === 0) {
        showNotification({
          message: t('multiCancel')
        })

        bulkProgress.current.clear()
        setSelectedQueueItems([])
        setMultiDeleteVideoAndFiles(false)
        setMultiBlockVideoId(false)
        closeMultiDeleteModal()
      } else {
        // keep only the failed items selected; retrying skips the stages that already completed
        showNotification({
          color: "red",
          message: t('multiCancelPartial', { failed: failedItems.length, total: selectedQueueItems.length })
        })
        handleSelectedQueueItemsChange(failedItems)
      }
    } finally {
      setMultiCancelLoading(false)
    }
  }

  if (queueIsPending) return (
    <GanymedeLoadingText message="Loading Queue" />
  )
  if (queueIsError) return <div>{t('error')}</div>

  return (

    <div>
      <Container mt={10} size="7xl">
        <Group justify="flex-end" mb={10} mih={36}>
          <Box>
            {selectedQueueItems.length >= 1 && (
              <Button
                leftSection={<IconSquareX size={16} />}
                color="red"
                onClick={openMultiDeleteModal}
              >
                {selectedQueueItems.length === 1
                  ? t('bulk.cancelOne')
                  : t('bulk.cancelMany', { number: selectedQueueItems.length })}
              </Button>
            )}
          </Box>
        </Group>
        <DataTable
          withTableBorder
          borderRadius="sm"
          withColumnBorders
          striped
          highlightOnHover
          records={records}
          columns={[
            {
              accessor: "id",
              title: t('column.id'),
            },
            { accessor: "edges.vod.edges.channel.name", title: t('column.channel') },
            { accessor: "edges.vod.ext_id", title: t('column.ext_id') },
            {
              accessor: "edges.vod.status",
              title: statusText('label'),
              render: ({ edges }) => <ArchiveStatusBadge videoId={edges.vod.id} status={edges.vod.status} />,
            },

            {
              accessor: "live_archive",
              title: t('column.live_archive'),
              render: ({ live_archive }) => (
                <Text>{live_archive ? "✅" : "❌"}</Text>
              ),
            },
            {
              accessor: "created_at",
              title: t('column.created_at'),
              render: ({ created_at }) => (
                <Text>{new Date(created_at).toLocaleString()}</Text>
              ),
            },
            {
              accessor: "actions",
              title: t('column.actions.actions'),
              render: (record) => (
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                  }}
                >
                  <Link href={"/queue/" + record.id}>
                    <Tooltip label={t('column.actions.view')} withinPortal>
                      <ActionIcon variant="light">
                        <IconEye size="1.125rem" />
                      </ActionIcon>
                    </Tooltip>
                  </Link>
                  <Tooltip label={t('column.actions.stop')} withinPortal>
                    <ActionIcon
                      variant="light"
                      color="red"
                      onClick={() => {
                        setActiveQueue(record);
                        openDeleteModal();
                      }}
                    >
                      <IconSquareX size="1.125rem" />
                    </ActionIcon>
                  </Tooltip>
                </div>
              ),
            },
          ]}
          totalRecords={queueItems.length}
          page={page}
          recordsPerPage={perPage}
          onPageChange={(p) => setPage(p)}
          recordsPerPageOptions={[10, 20, 50]}
          onRecordsPerPageChange={(value) => {
            setPerPage(value);
            setSettingsAdminItemsPerPage(value);
            setPage(1);
          }}
          selectedRecords={selectedQueueItems}
          onSelectedRecordsChange={handleSelectedQueueItemsChange}
          recordsPerPageLabel={miscT('recordsPerPageLabel')}
        />
      </Container>
      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('modal.cancel.title')}>
        <div>
          <Text>{t('modal.cancel.text')}</Text>
          <Text size="sm" fs="italic">{t('modal.cancel.description')}</Text>
          <Switch
            mt={5}
            defaultChecked
            color="red"
            label={t('modal.cancel.delete')}
            checked={deleteVideoAndFiles}
            onChange={(event) => setDeleteVideoAndFiles(event.currentTarget.checked)}
          />
          {(activeQueue != null && !activeQueue.live_archive) && (<Switch
            mt={5}
            defaultChecked
            color="violet"
            label={t('modal.cancel.block')}
            checked={blockVideoId}
            onChange={(event) => setBlockVideoId(event.currentTarget.checked)}
          />)}
          <Button variant="filled" color="orange" fullWidth loading={cancelQueueLoading} mt={10} onClick={cancelQueueItem}>{t('modal.cancel.title')}</Button>
        </div>
      </Modal>
      <Modal opened={multiDeleteModalOpened} onClose={closeMultiDeleteModal} title={t('modal.cancel.title')}>
        <div>
          <Text>{t('modal.cancel.multiText', { number: selectedQueueItems.length })}</Text>
          <Text size="sm" fs="italic">{t('modal.cancel.description')}</Text>
          <Switch
            mt={5}
            color="red"
            label={t('modal.cancel.delete')}
            disabled={multiCancelLoading}
            checked={multiDeleteVideoAndFiles}
            onChange={(event) => setMultiDeleteVideoAndFiles(event.currentTarget.checked)}
          />
          {selectedQueueItems.some((queue) => !queue.live_archive) && (<Switch
            mt={5}
            color="violet"
            label={t('modal.cancel.block')}
            disabled={multiCancelLoading}
            checked={multiBlockVideoId}
            onChange={(event) => setMultiBlockVideoId(event.currentTarget.checked)}
          />)}
          <Button variant="filled" color="orange" fullWidth loading={multiCancelLoading} mt={10} onClick={cancelSelectedQueueItems}>{t('modal.cancel.title')}</Button>
        </div>
      </Modal>
    </div>
  );
}

export default QueuePage;