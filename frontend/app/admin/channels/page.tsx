"use client"
import { Channel, useFetchChannels } from "@/app/hooks/useChannels";
import { ActionIcon, Container, Group, TextInput, Title, Box, Button, Drawer, Modal, Text } from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useEffect, useState } from "react";
import sortBy from "lodash/sortBy";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { IconPencil, IconSearch, IconTrash } from "@tabler/icons-react";
import dayjs from "dayjs";

import classes from "./AdminChannelsPage.module.css"
import AdminChannelDrawerContent, { ChannelEditMode } from "@/app/components/admin/channel/DrawerContent";
import PlatformChannelDrawerContent from "@/app/components/admin/channel/PlatformDrawerContent";
import DeleteChannelModalContent from "@/app/components/admin/channel/DeleteModalContent";
import { useTranslations } from "next-intl";
import { formatBytes, usePageTitle } from "@/app/util/util";

const AdminChannelsPage = () => {
  const t = useTranslations("AdminChannelsPage");
  usePageTitle(t('title'))
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [records, setRecords] = useState<Channel[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<Channel>>({
    columnAccessor: "name",
    direction: "asc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [activeChannel, setActiveChannel] = useState<Channel | null>(null);
  const [drawerEditMode, setDrawerEditMode] = useState<ChannelEditMode>(ChannelEditMode.Create)

  const [channelDrawerOpened, { open: openChannelDrawer, close: closeChannelDrawer }] = useDisclosure(false);
  const [platformChannelDrawerOpened, { open: openPlatformChannelDrawer, close: closePlatformChannelDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);


  const { data: channels, isPending, isError } = useFetchChannels()

  useEffect(() => {
    if (!channels) return;

    let filteredData = [...channels] as Channel[];

    // Apply search filter
    if (debouncedQuery) {
      filteredData = filteredData.filter((channel) => {
        return (
          channel.id.toString().includes(debouncedQuery) ||
          channel.name.toLowerCase().includes(debouncedQuery.toLowerCase()) ||
          channel.display_name.toLowerCase().includes(debouncedQuery.toLowerCase())
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
  }, [channels, page, perPage, sortStatus, debouncedQuery, initialRecords]);

  const openDrawer = (mode: ChannelEditMode, channel: Channel) => {
    setActiveChannel(channel);
    setDrawerEditMode(ChannelEditMode.Edit)
    openChannelDrawer()
    // setDrawerOpened(true);
  };

  const handleDeleteModal = (channel: Channel) => {
    setActiveChannel(channel);
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
                setDrawerEditMode(ChannelEditMode.Create)
                setActiveChannel(null)
                openChannelDrawer()
              }}
              mr={5}
              variant="default"
            >
              {t('createButton')}
            </Button>
            <Button
              onClick={openPlatformChannelDrawer}
              color="violet"
            >
              {t('addTwitchChannel')}
            </Button>
          </Box>
        </Group>
        <Text>{t('storageSizeDisclaimerText')}</Text>


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
          <DataTable<Channel>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={records}
            columns={[
              { accessor: "id", title: t('columns.id') },
              { accessor: "ext_id", title: t('columns.externalId') },
              { accessor: "name", title: t('columns.name'), sortable: true },
              { accessor: "display_name", title: t('columns.displayName'), sortable: true },
              {
                accessor: "storage_size_bytes", title: t('columns.storageSize'), sortable: true,
                render: ({ storage_size_bytes }) => {
                  return formatBytes(storage_size_bytes ?? 0, 2);
                },
              },
              {
                accessor: "retention",
                title: t('columns.videoRetention'),
                sortable: false,
                render: ({ retention }) => (
                  retention ? (
                    <div>✅</div>
                  ) : (
                    <div>❌</div>
                  )
                ),
              },
              {
                accessor: "created_at",
                title: t('columns.createdAt'),
                sortable: true,
                render: ({ created_at }) => (
                  <div>{dayjs(created_at).format("YYYY/MM/DD")}</div>
                ),
              },
              {
                accessor: "actions",
                title: t('columns.actions'),
                render: (channel) => (
                  <Group>
                    <ActionIcon
                      onClick={() => openDrawer(ChannelEditMode.Edit, channel)}
                      className={classes.actionButton}
                      variant="light"
                    >
                      <IconPencil size={18} />
                    </ActionIcon>
                    <ActionIcon
                      onClick={() => handleDeleteModal(channel)}
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
            totalRecords={channels?.length ?? 0}
            page={page}
            recordsPerPage={perPage}
            onPageChange={(p) => setPage(p)}
            recordsPerPageOptions={[20, 40, 100]}
            onRecordsPerPageChange={setPerPage}
            sortStatus={sortStatus}
            onSortStatusChange={setSortStatus}
          />
        </Box>
      </Container>

      <Drawer opened={channelDrawerOpened} onClose={closeChannelDrawer} position="right" title={t('drawer')}>
        <AdminChannelDrawerContent mode={drawerEditMode} channel={activeChannel} handleClose={closeChannelDrawer} />
      </Drawer>
      <Drawer opened={platformChannelDrawerOpened} onClose={closePlatformChannelDrawer} position="right" title={t('platformDrawer')}>
        <PlatformChannelDrawerContent handleClose={closePlatformChannelDrawer} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('deleteModal')}>
        <DeleteChannelModalContent channel={activeChannel} handleClose={closeDeleteModal} />
      </Modal>

    </div>
  );
}

export default AdminChannelsPage;