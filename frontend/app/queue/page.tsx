"use client"
import { useEffect, useState } from "react";
import { useAxiosPrivate } from "../hooks/useAxios";
import { Queue, QueueTaskStatus, useGetQueueItems, useStopQueueItem } from "../hooks/useQueue";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import { DataTable } from "mantine-datatable";
import { Tooltip, Text, ThemeIcon, ActionIcon, Loader, Container, Modal, Switch, Button } from "@mantine/core";
import { IconEye, IconPlayerPause, IconSquareX } from "@tabler/icons-react";
import Link from "next/link";
import classes from "./QueuePage.module.css"
import { useDisclosure } from "@mantine/hooks";
import { useDeleteVideo } from "../hooks/useVideos";
import { useBlockVideo } from "../hooks/useBlockedVideos";
import { useQueryClient } from "@tanstack/react-query";
import { showNotification } from "@mantine/notifications";

const QueuePage = () => {
  useEffect(() => {
    document.title = "Queue";
  }, []);

  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(10);
  const [records, setRecords] = useState<Queue[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [activeQueue, setActiveQueue] = useState<Queue | null>(null);

  // delete state
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [cancelQueueLoading, setCancelQueueLoading] = useState(false);
  const [deleteVideoAndFiles, setDeleteVideoAndFiles] = useState(false);
  const [blockVideoId, setBlockVideoId] = useState(false);

  const axiosPrivate = useAxiosPrivate()
  const queryClient = useQueryClient()

  const stopQueueItemMutate = useStopQueueItem()
  const deleteVideoMutate = useDeleteVideo()
  const blockVideoMutate = useBlockVideo()

  const { data: queueItems, isPending: queueIsPending, isError: queueIsError } = useGetQueueItems(axiosPrivate, true)

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

  const checkFailed = (record: Queue) => {
    if (
      record.task_vod_create_folder == QueueTaskStatus.Failed ||
      record.task_vod_save_info == QueueTaskStatus.Failed ||
      record.task_video_download == QueueTaskStatus.Failed ||
      record.task_video_convert == QueueTaskStatus.Failed ||
      record.task_video_move == QueueTaskStatus.Failed ||
      record.task_chat_download == QueueTaskStatus.Failed ||
      record.task_chat_convert == QueueTaskStatus.Failed ||
      record.task_chat_render == QueueTaskStatus.Failed ||
      record.task_chat_move == QueueTaskStatus.Failed
    ) {
      return true;
    }
    return false;
  };

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
        message: "Queue item has been cancelled"
      })

      closeDeleteModal()
    } catch (error) {
      console.log(error)
    } finally {
      setCancelQueueLoading(false)
      setActiveQueue(null)
    }

  }

  if (queueIsPending) return (
    <GanymedeLoadingText message="Loading Queue" />
  )
  if (queueIsError) return <div>Error loading queue</div>

  return (

    <div>
      <Container mt={10} size="7xl">
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
              title: "ID",
            },
            { accessor: "edges.vod.edges.channel.name", title: "Channel" },
            { accessor: "edges.vod.ext_id", title: "External ID" },
            {
              accessor: "processing",
              title: "Status",
              render: (value) => (
                <div>
                  {checkFailed(value) && (
                    <div>
                      <Tooltip label="Task in 'failed' state">
                        <Text className={classes.errBadge}>ERROR</Text>
                      </Tooltip>
                    </div>
                  )}
                  {value.processing && !checkFailed(value) && !value.on_hold && (
                    <div>
                      <Tooltip label="Processing">
                        <Loader mt={2} color="green" size="sm" />
                      </Tooltip>
                    </div>
                  )}
                  {value.processing && !checkFailed(value) && value.on_hold && (
                    <div>
                      <Tooltip label="On Hold">
                        <ThemeIcon variant="outline" color="orange">
                          <IconPlayerPause />
                        </ThemeIcon>
                      </Tooltip>
                    </div>
                  )}
                </div>
              ),
            },

            {
              accessor: "live_archive",
              title: "Live Archive",
              render: ({ live_archive }) => (
                <Text>{live_archive ? "✅" : "❌"}</Text>
              ),
            },
            {
              accessor: "created_at",
              title: "Created At",
              render: ({ created_at }) => (
                <Text>{new Date(created_at).toLocaleString()}</Text>
              ),
            },
            {
              accessor: "actions",
              title: "Actions",
              render: (record) => (
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                  }}
                >
                  <Link href={"/queue/" + record.id}>
                    <Tooltip label="View queue item" withinPortal>
                      <ActionIcon variant="light">
                        <IconEye size="1.125rem" />
                      </ActionIcon>
                    </Tooltip>
                  </Link>
                  <Tooltip label="Stop queue item" withinPortal>
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
          onRecordsPerPageChange={setPerPage}
        />
      </Container>
      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title="Cancel Queue Item">
        <div>
          <Text>Are you sure you want to cancel the queue item?</Text>
          <Text size="sm" fs="italic">For live archives this will stop the video and chat download then proceed with post-processing. For VOD archives this will stop all the tasks.</Text>
          <Switch
            mt={5}
            defaultChecked
            color="red"
            label="Delete video and video files"
            checked={deleteVideoAndFiles}
            onChange={(event) => setDeleteVideoAndFiles(event.currentTarget.checked)}
          />
          {(activeQueue != null && !activeQueue.live_archive) && (<Switch
            mt={5}
            defaultChecked
            color="violet"
            label="Block video ID"
            checked={blockVideoId}
            onChange={(event) => setBlockVideoId(event.currentTarget.checked)}
          />)}
          <Button variant="filled" color="orange" fullWidth loading={cancelQueueLoading} mt={10} onClick={cancelQueueItem}>Cancel Queue Item</Button>
        </div>
      </Modal>
    </div>
  );
}

export default QueuePage;