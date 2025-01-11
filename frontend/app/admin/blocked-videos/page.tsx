"use client"
import { ActionIcon, Container, Group, TextInput, Title, Box, Button, Drawer, Modal, Text } from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useEffect, useState } from "react";
import sortBy from "lodash/sortBy";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { IconSearch, IconTrash } from "@tabler/icons-react";
import dayjs from "dayjs";

import classes from "./AdminBlockedVideosPage.module.css"
import { BlockedVideo, useGetBlockedVideos } from "@/app/hooks/useBlockedVideos";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import DeleteBlockedVideoModalContent from "@/app/components/admin/blocked-videos/DeleteModalContent";
import AdminBlockedVideosDrawerContent from "@/app/components/admin/blocked-videos/DrawerContent";
import MultiDeleteBlockedVideoModalContent from "@/app/components/admin/blocked-videos/MultiDeleteModalContent";

const AdminBlockedVideosPage = () => {
  useEffect(() => {
    document.title = "Admin - Blocked Videos";
  }, []);
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [records, setRecords] = useState<BlockedVideo[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<BlockedVideo>>({
    columnAccessor: "id",
    direction: "asc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [activeBlockedVideo, setActiveBlockedVideo] = useState<BlockedVideo | null>(null);

  const [blockedVideoDrawerOpened, { open: openBlockedVideoDrawer, close: closeBlockedVideoDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [multiDeleteModalOpened, { open: openMultiDeleteModal, close: closeMultiDeleteModal }] = useDisclosure(false);

  const [activeBlockedVideos, setActiveBlockedVideos] = useState<BlockedVideo[] | null>([]);

  const axiosPrivate = useAxiosPrivate()

  const { data: blockedVideos, isPending, isError } = useGetBlockedVideos(axiosPrivate)

  useEffect(() => {
    if (!blockedVideos) return;

    let filteredData = [...blockedVideos] as BlockedVideo[];

    // Apply search filter
    if (debouncedQuery) {
      filteredData = filteredData.filter((blockedVideo) => {
        return (
          blockedVideo.id.toString().includes(debouncedQuery)
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
  }, [blockedVideos, page, perPage, sortStatus, debouncedQuery, initialRecords]);

  const handleDeleteModal = (blockedVideo: BlockedVideo) => {
    setActiveBlockedVideo(blockedVideo);
    openDeleteModal()
  };

  const handleMultiDeleteModalCallback = () => {
    closeMultiDeleteModal()
    setActiveBlockedVideos([])
  }

  if (isPending) return (
    <GanymedeLoadingText message="Loading Blocked Videos" />
  )
  if (isError) return <div>Error loading blocked videos</div>

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2} >
          <Title>Manage Blocked Videos</Title>
          <Box>
            {(activeBlockedVideos && activeBlockedVideos.length >= 1) && (
              <Button
                mx={10}
                leftSection={<IconTrash size={16} />}
                color="red"
                disabled={!activeBlockedVideos.length}
                onClick={() => {
                  openMultiDeleteModal();
                }}
              >
                {activeBlockedVideos.length
                  ? `Delete ${activeBlockedVideos.length === 1
                    ? "one selected blocked video"
                    : `${activeBlockedVideos.length} selected blocked videos`
                  }`
                  : "Select blocked videos to delete"}
              </Button>
            )}
            <Button
              onClick={() => {
                setActiveBlockedVideo(null)
                openBlockedVideoDrawer()
              }}
              mr={5}
              variant="default"
            >
              Add Blocked Video ID
            </Button>
          </Box>
        </Group>

        <Text>External platform video IDs that are blocked from being archived.</Text>

        <Box mt={5}>
          <div>
            <TextInput
              placeholder="Search blocked videos..."
              leftSection={<IconSearch size={16} />}
              value={query}
              onChange={(e) => setQuery(e.currentTarget.value)}
              mb={10}
            />

          </div>
          <DataTable<BlockedVideo>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={records}
            columns={[
              { accessor: "id", title: "ID" },
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
                render: (blockedVideo) => (
                  <Group>
                    <ActionIcon
                      onClick={() => handleDeleteModal(blockedVideo)}
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
            totalRecords={blockedVideos?.length ?? 0}
            page={page}
            recordsPerPage={perPage}
            onPageChange={(p) => setPage(p)}
            recordsPerPageOptions={[20, 40, 100]}
            onRecordsPerPageChange={setPerPage}
            sortStatus={sortStatus}
            onSortStatusChange={setSortStatus}
            selectedRecords={activeBlockedVideos ?? []}
            onSelectedRecordsChange={setActiveBlockedVideos}
          />
        </Box>
      </Container>

      <Drawer opened={blockedVideoDrawerOpened} onClose={closeBlockedVideoDrawer} position="right" title="Channel">
        <AdminBlockedVideosDrawerContent blockedVideo={activeBlockedVideo} handleClose={closeBlockedVideoDrawer} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title="Delete Channel">
        {activeBlockedVideo && (
          <DeleteBlockedVideoModalContent blockedVideo={activeBlockedVideo} handleClose={closeDeleteModal} />
        )}
      </Modal>

      <Modal opened={multiDeleteModalOpened} onClose={closeMultiDeleteModal} title="Delete Queue Items">
        {activeBlockedVideos && (
          <MultiDeleteBlockedVideoModalContent blockedVideos={activeBlockedVideos} handleClose={handleMultiDeleteModalCallback} />
        )}
      </Modal>

    </div>
  );
}

export default AdminBlockedVideosPage;