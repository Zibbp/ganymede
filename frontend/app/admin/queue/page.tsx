"use client"
import { ActionIcon, Container, Group, TextInput, Title, Box, Drawer, Modal, Tooltip, Text, Button } from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useEffect, useState } from "react";
import sortBy from "lodash/sortBy";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { IconEye, IconPencil, IconSearch, IconTrash } from "@tabler/icons-react";
import dayjs from "dayjs";
import classes from "./AdminQueuePage.module.css"
import { Queue, useGetQueueItems } from "@/app/hooks/useQueue";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import AdminQueueDrawerContent from "@/app/components/admin/queue/DrawerContent";
import DeleteQueueModalContent from "@/app/components/admin/queue/DeleteModalContent";
import Link from "next/link";
import MultiDeleteQueueModalContent from "@/app/components/admin/queue/MultiDeleteModalContext";

const AdminQueuePage = () => {
  useEffect(() => {
    document.title = "Admin - Queue";
  }, []);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [records, setRecords] = useState<Queue[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<Queue>>({
    columnAccessor: "name",
    direction: "asc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [activeQueue, setActiveQueue] = useState<Queue | null>(null);

  const [queueDrawerOpened, { open: openQueueDrawer, close: closeQueueDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [multiDeleteModalOpened, { open: openMultiDeleteModal, close: closeMultiDeleteModal }] = useDisclosure(false);

  const [activeQueueItems, setActiveQueueItems] = useState<Queue[] | null>([]);

  const axiosPrivate = useAxiosPrivate()
  const { data: queues, isPending, isError } = useGetQueueItems(axiosPrivate, false)

  useEffect(() => {
    if (!queues) return;

    let filteredData = [...queues] as Queue[];

    // Apply search filter
    if (debouncedQuery) {
      filteredData = filteredData.filter((queue) => {
        return (
          queue.id.toString().includes(debouncedQuery)
        );
      });
    }

    // Apply sorting
    const sortedData = sortBy(filteredData, sortStatus.columnAccessor);
    filteredData = sortStatus.direction === "desc" ? sortedData.reverse() : sortedData;

    // Apply pagination
    const from = (page - 1) * perPage;
    const to = from + perPage;
    setRecords(filteredData.slice(from, to));

    if (!initialRecords) {
      setInitialRecords(true);
    }
  }, [queues, page, perPage, sortStatus, debouncedQuery, initialRecords]);

  const openDrawer = (queue: Queue) => {
    setActiveQueue(queue);
    openQueueDrawer()
  };

  const handleDeleteModal = (queue: Queue) => {
    setActiveQueue(queue);
    openDeleteModal()
  };

  const handleMultiDeleteModalCallback = () => {
    closeMultiDeleteModal()
    setActiveQueueItems([])
  }

  if (isPending) return (
    <GanymedeLoadingText message="Loading queue items" />
  )
  if (isError) return <div>Error loading queue items</div>

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2} >
          <Title>Manage Queue</Title>
          <Box>
            {(activeQueueItems && activeQueueItems.length >= 1) && (
              <Button
                mx={10}
                leftSection={<IconTrash size={16} />}
                color="red"
                disabled={!activeQueueItems.length}
                onClick={() => {
                  openMultiDeleteModal();
                }}
              >
                {activeQueueItems.length
                  ? `Delete ${activeQueueItems.length === 1
                    ? "one selected queue item"
                    : `${activeQueueItems.length} selected queue items`
                  }`
                  : "Select queue items to delete"}
              </Button>
            )}
          </Box>
        </Group>

        <Box mt={5}>
          <div>
            <TextInput
              placeholder="Search queue items..."
              leftSection={<IconSearch size={16} />}
              value={query}
              onChange={(e) => setQuery(e.currentTarget.value)}
              mb={10}
            />

          </div>
          <DataTable<Queue>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={records}
            columns={[
              {
                accessor: "id", title: "ID",
                render: ({ id }) => (
                  <Tooltip label={id}>
                    <Text lineClamp={1}>{id}</Text>
                  </Tooltip>
                ),
              },
              {
                accessor: "edges.vod.id", title: "Video ID", sortable: true,
                render: (queue) => (
                  <Tooltip label={queue.edges.vod.id}>
                    <Text lineClamp={1}>{queue.edges.vod.id}</Text>
                  </Tooltip>
                ),
              },
              {
                accessor: "edges.vod.ext_id",
                title: "External Video ID",
                sortable: true,
                render: (queue) => (
                  <Tooltip label={queue.edges.vod.ext_id}>
                    <Text lineClamp={1}>{queue.edges.vod.ext_id}</Text>
                  </Tooltip>
                ),
              },
              {
                accessor: "processing",
                title: "Processing",
                sortable: true,
                render: ({ processing }) => {
                  return processing ? "✅" : "❌";
                },
              },
              {
                accessor: "on_hold",
                title: "On Hold",
                sortable: true,
                render: ({ on_hold }) => {
                  return on_hold ? "✅" : "❌";
                },
              },
              {
                accessor: "live_archive",
                title: "Live Archive",
                sortable: true,
                render: ({ live_archive }) => {
                  return live_archive ? "✅" : "❌";
                },
              },
              {
                accessor: "created_at",
                title: "Created At",
                sortable: true,
                render: ({ created_at }) => (
                  <div>{dayjs(created_at).format("YYYY/MM/DD")}</div>
                ),
              },
              {
                accessor: "actions",
                title: "Actions",
                width: "150px",
                render: (queue) => (
                  <Group>
                    <ActionIcon
                      component={Link}
                      href={`/queue/${queue.id}`}
                      className={classes.actionButton}
                      variant="light"
                    >
                      <IconEye size={18} />
                    </ActionIcon>
                    <ActionIcon
                      onClick={() => openDrawer(queue)}
                      className={classes.actionButton}
                      variant="light"
                    >
                      <IconPencil size={18} />
                    </ActionIcon>
                    <ActionIcon
                      onClick={() => handleDeleteModal(queue)}
                      className={classes.actionButton}
                      variant="light"
                      color="red"
                    >
                      <IconTrash size={18} />
                    </ActionIcon>
                  </Group>
                ),
              },
            ]}
            totalRecords={queues?.length ?? 0}
            page={page}
            recordsPerPage={perPage}
            onPageChange={(p) => setPage(p)}
            recordsPerPageOptions={[20, 40, 100]}
            onRecordsPerPageChange={setPerPage}
            sortStatus={sortStatus}
            onSortStatusChange={setSortStatus}
            selectedRecords={activeQueueItems ?? []}
            onSelectedRecordsChange={setActiveQueueItems}
          />
        </Box>
      </Container>

      <Drawer opened={queueDrawerOpened} onClose={closeQueueDrawer} position="right" title="Queue">
        {activeQueue && (
          <AdminQueueDrawerContent queue={activeQueue} handleClose={closeQueueDrawer} />
        )}
      </Drawer>


      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title="Delete Queue">
        {activeQueue && (
          <DeleteQueueModalContent queue={activeQueue} handleClose={closeDeleteModal} />
        )}
      </Modal>

      <Modal opened={multiDeleteModalOpened} onClose={closeMultiDeleteModal} title="Delete Queue Items">
        {activeQueueItems && (
          <MultiDeleteQueueModalContent queues={activeQueueItems} handleClose={handleMultiDeleteModalCallback} />
        )}
      </Modal>

    </div>
  );
}

export default AdminQueuePage;