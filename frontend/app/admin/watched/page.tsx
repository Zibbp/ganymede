"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useGetWatchedChannesl, WatchedChannel } from "@/app/hooks/useWatchedChannels";
import { Container, Group, Title, Box, Button, TextInput, ActionIcon, Drawer, Modal } from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { IconSearch, IconPencil, IconTrash } from "@tabler/icons-react";
import { sortBy } from "lodash";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useState, useEffect } from "react";
import classes from "./AdminWatchedChannelsPage.module.css"
import AdminWatchedChannelDrawerContent, { WatchedChannelEditMode } from "@/app/components/admin/watched/DrawerContent";
import DeleteWatchedChannelModalContent from "@/app/components/admin/watched/DeleteModalContent";
import { useTranslations } from "next-intl";
import { usePageTitle } from "@/app/util/util";
import useSettingsStore from "@/app/store/useSettingsStore";

const AdminWatchChannelsPage = () => {
  const t = useTranslations('AdminWatchedChannelsPage')
  const miscT = useTranslations('MiscComponents')
  usePageTitle(t('title'))

  const settingsAdminItemsPerPage = useSettingsStore((state) => state.adminItemsPerPage);
  const setSettingsAdminItemsPerPage = useSettingsStore((state) => state.setAdminItemsPerPage)

  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(settingsAdminItemsPerPage);
  const [records, setRecords] = useState<WatchedChannel[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<WatchedChannel>>({
    columnAccessor: "edges.channel.name",
    direction: "asc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [activeWatchedChannel, setActiveWatchedChannel] = useState<WatchedChannel | null>(null);
  const [drawerEditMode, setDrawerEditMode] = useState<WatchedChannelEditMode>(WatchedChannelEditMode.Create)

  const [channelDrawerOpened, { open: openChannelDrawer, close: closeChannelDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);

  const axiosPrivate = useAxiosPrivate()

  const { data: channels, isPending, isError } = useGetWatchedChannesl(axiosPrivate)

  useEffect(() => {
    setPerPage(settingsAdminItemsPerPage);
  }, [settingsAdminItemsPerPage]);

  useEffect(() => {
    if (!channels) return;

    let filteredData = [...channels] as WatchedChannel[];

    // Apply search filter
    if (debouncedQuery) {
      filteredData = filteredData.filter((channel) => {
        return (
          channel.id.toString().includes(debouncedQuery) ||
          channel.edges.channel.name.toLowerCase().includes(debouncedQuery.toLowerCase()) ||
          channel.edges.channel.display_name.toLowerCase().includes(debouncedQuery.toLowerCase())
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
  }, [channels, page, perPage, sortStatus, debouncedQuery, initialRecords]);

  const openDrawer = (mode: WatchedChannelEditMode, watchedChannel: WatchedChannel) => {
    setActiveWatchedChannel(watchedChannel);
    setDrawerEditMode(WatchedChannelEditMode.Edit)
    openChannelDrawer()
  };

  const handleDeleteModal = (watchedChannel: WatchedChannel) => {
    setActiveWatchedChannel(watchedChannel);
    openDeleteModal()
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
            <Button
              onClick={() => {
                setDrawerEditMode(WatchedChannelEditMode.Create)
                setActiveWatchedChannel(null)
                openChannelDrawer()
              }}
              mr={5}
              variant="default"
            >
              {t('addButton')}
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
          <DataTable<WatchedChannel>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={records}
            columns={[
              { accessor: "id", title: t('columns.id') },
              {
                accessor: "edges.channel.display_name",
                title: t('columns.channel'),
                sortable: true,
              },
              {
                accessor: "watch_live",
                title: t('columns.watchLive'),
                sortable: true,
                render: ({ watch_live }) => {
                  return watch_live ? "✅" : "❌";
                },
              },
              {
                accessor: "is_live",
                title: t('columns.isLive'),
                sortable: true,
                render: ({ is_live }) => {
                  return is_live ? "✅" : "❌";
                },
              },
              {
                accessor: "watch_vod",
                title: t('columns.watchVideos'),
                sortable: true,
                render: ({ watch_vod }) => {
                  return watch_vod ? "✅" : "❌";
                },
              },
              {
                accessor: "watch_clips",
                title: t('columns.watchClips'),
                sortable: true,
                render: ({ watch_clips }) => {
                  return watch_clips ? "✅" : "❌";
                },
              },

              {
                accessor: "actions",
                title: t('columns.actions'),
                render: (watched) => (
                  <Group>
                    <ActionIcon
                      onClick={() => openDrawer(WatchedChannelEditMode.Edit, watched)}
                      className={classes.actionButton}
                      variant="light"
                    >
                      <IconPencil size={18} />
                    </ActionIcon>
                    <ActionIcon
                      onClick={() => handleDeleteModal(watched)}
                      className={classes.actionButton}
                      variant="light" color="red"
                    >
                      <IconTrash size={18} />
                    </ActionIcon>
                  </Group>
                ),
              },
            ]}
            totalRecords={channels?.length ?? 0}
            page={page}
            recordsPerPage={perPage}
            onPageChange={(p) => setPage(p)}
            recordsPerPageOptions={[20, 40, 100]}
            onRecordsPerPageChange={(value) => {
              setPerPage(value);
              setSettingsAdminItemsPerPage(value);
              setPage(1);
            }}
            sortStatus={sortStatus}
            onSortStatusChange={setSortStatus}
            recordsPerPageLabel={miscT('recordsPerPageLabel')}
          />
        </Box>
      </Container>

      <Drawer opened={channelDrawerOpened} onClose={closeChannelDrawer} position="right" title={t('drawer')}>
        <AdminWatchedChannelDrawerContent mode={drawerEditMode} watchedChannel={activeWatchedChannel} handleClose={closeChannelDrawer} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('deleteModal')}>
        <DeleteWatchedChannelModalContent watchedChannel={activeWatchedChannel} handleClose={closeDeleteModal} />
      </Modal>

    </div>
  );
}

export default AdminWatchChannelsPage;