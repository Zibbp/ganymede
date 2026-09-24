"use client"
import { ActionIcon, Box, Button, Code, Container, Group, Loader, Modal, Text, TextInput, Title } from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { useQueryClient } from "@tanstack/react-query";
import { showNotification } from "@mantine/notifications";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useEffect, useState } from "react";
import sortBy from "lodash/sortBy";
import dayjs from "dayjs";
import { IconFileImport, IconRefresh, IconSearch, IconTrash } from "@tabler/icons-react";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import StorageActionModalContent, { StorageAction } from "@/app/components/admin/storage/ActionModalContent";
import { StorageFinding, useGetStorageFindings } from "@/app/hooks/useStorage";
import { Task, useStartTask } from "@/app/hooks/useTasks";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import useSettingsStore from "@/app/store/useSettingsStore";
import { formatBytes, usePageTitle } from "@/app/util/util";
import { useTranslations } from "next-intl";

import classes from "./AdminStoragePage.module.css";

const AdminStoragePage = () => {
  const t = useTranslations("AdminStoragePage");
  const miscT = useTranslations("MiscComponents");
  usePageTitle(t('title'))

  const settingsAdminItemsPerPage = useSettingsStore((state) => state.adminItemsPerPage);
  const setSettingsAdminItemsPerPage = useSettingsStore((state) => state.setAdminItemsPerPage)

  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(settingsAdminItemsPerPage);
  const [records, setRecords] = useState<StorageFinding[]>([]);
  const [filteredCount, setFilteredCount] = useState(0);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<StorageFinding>>({
    columnAccessor: "relative_path",
    direction: "asc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [selectedFindings, setSelectedFindings] = useState<StorageFinding[]>([]);
  const [modalAction, setModalAction] = useState<StorageAction>("delete");
  const [modalFindings, setModalFindings] = useState<StorageFinding[]>([]);

  const [modalOpened, { open: openModal, close: closeModal }] = useDisclosure(false);

  // what the last completed scan was when the current one was started, so that its result can
  // be reported whenever it arrives
  const [awaitingScan, setAwaitingScan] = useState<{
    since: number;
    previousCompleted: string | null;
    previousFailed: string | null;
  } | null>(null);

  const axiosPrivate = useAxiosPrivate()

  const queryClient = useQueryClient()

  const { data, isPending, isError } = useGetStorageFindings(axiosPrivate, awaitingScan !== null)
  const startTaskMutate = useStartTask()

  useEffect(() => {
    setPerPage(settingsAdminItemsPerPage);
  }, [settingsAdminItemsPerPage]);

  useEffect(() => {
    if (!data) return;

    let filteredData = [...data.findings];

    if (debouncedQuery) {
      filteredData = filteredData.filter((finding) =>
        finding.relative_path.toLowerCase().includes(debouncedQuery.toLowerCase())
      );
    }

    const sortedData = sortBy(filteredData, sortStatus.columnAccessor);
    filteredData = sortStatus.direction === "desc" ? sortedData.reverse() : sortedData;

    setFilteredCount(filteredData.length);

    // the list can shrink under the current page after an action
    const currentPage = Math.min(page, Math.max(Math.ceil(filteredData.length / perPage), 1));
    if (currentPage !== page) {
      setPage(currentPage);
    }

    const from = (currentPage - 1) * perPage;
    const to = from + perPage;
    setRecords(filteredData.slice(from, to));
  }, [data, page, perPage, sortStatus, debouncedQuery]);

  // an action can succeed for part of a batch, so the selection is reconciled against the
  // refreshed list rather than against what was sent
  useEffect(() => {
    if (!data) return;

    const ids = new Set(data.findings.map((finding) => finding.id));
    setSelectedFindings((current) => current.filter((finding) => ids.has(finding.id)));
  }, [data]);

  // report the result of a scan that was started from this page
  useEffect(() => {
    if (!data || !awaitingScan) return;

    if (data.scan.last_completed_at && data.scan.last_completed_at !== awaitingScan.previousCompleted) {
      setAwaitingScan(null);
      showNotification({
        message: t('scan.finishedNotification', { length: data.findings.length }),
      });
      return;
    }

    if (data.scan.last_failed_at && data.scan.last_failed_at !== awaitingScan.previousFailed) {
      setAwaitingScan(null);
      showNotification({ message: t('scan.failedNotification'), color: "red" });
      return;
    }

    // Without a worker to pick the job up, nothing would ever arrive.
    if (Date.now() - awaitingScan.since > 120000) {
      setAwaitingScan(null);
      showNotification({ message: t('scan.pendingNotification'), color: "yellow" });
    }
  }, [data, awaitingScan, t]);

  const handleAction = (action: StorageAction, findings: StorageFinding[]) => {
    setModalAction(action);
    setModalFindings(findings);
    openModal()
  };

  const handleModalClose = () => {
    closeModal()
    setModalFindings([])
  }

  const startScan = async () => {
    try {
      await startTaskMutate.mutateAsync({ axiosPrivate, task: Task.ReconcileStorage })
      setAwaitingScan({
        since: Date.now(),
        previousCompleted: data?.scan.last_completed_at ?? null,
        previousFailed: data?.scan.last_failed_at ?? null,
      })
      showNotification({ message: t('scanStartedNotification') })
      await queryClient.invalidateQueries({ queryKey: ["storage-findings"] })
    } catch (error) {
      console.error(error)
    }
  }

  if (isPending) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isError) return <div>{t('error')}</div>

  const scanning = data.scan.state !== "idle" || awaitingScan !== null;
  const totalBytes = data.findings.reduce((total, finding) => total + finding.size_bytes, 0);

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2}>
          <Title>{t('header')}</Title>
          <Group>
            {selectedFindings.length >= 1 && (
              <>
                <Button
                  leftSection={<IconFileImport size={16} />}
                  color="violet"
                  variant="light"
                  onClick={() => handleAction("import", selectedFindings)}
                >
                  {t('actions.importSelected', { length: selectedFindings.length })}
                </Button>
                <Button
                  leftSection={<IconTrash size={16} />}
                  color="red"
                  onClick={() => handleAction("delete", selectedFindings)}
                >
                  {t('actions.deleteSelected', { length: selectedFindings.length })}
                </Button>
              </>
            )}
            <Button
              leftSection={scanning ? <Loader size={14} color="white" /> : <IconRefresh size={16} />}
              onClick={startScan}
              loading={startTaskMutate.isPending}
              disabled={scanning}
              variant="default"
            >
              {scanning ? t('scan.running') : t('scan.start')}
            </Button>
          </Group>
        </Group>

        <Text>{t('body')}</Text>
        <Text mt={5}>
          {t('videosDirectory')} <Code>{data.videos_directory}</Code>
        </Text>
        <Text>
          {scanning && (data.scan.started_at
            ? t('scan.runningSince', { time: dayjs(data.scan.started_at).format("YYYY/MM/DD HH:mm") })
            : t('scan.starting'))}
          {!scanning && data.scan.last_completed_at && t('scan.lastCompleted', { time: dayjs(data.scan.last_completed_at).format("YYYY/MM/DD HH:mm") })}
          {!scanning && !data.scan.last_completed_at && (data.findings.length > 0 ? t('scan.unknown') : t('scan.never'))}
        </Text>
        {data.scan.last_failed_at && (!data.scan.last_completed_at || dayjs(data.scan.last_failed_at).isAfter(data.scan.last_completed_at)) && (
          <Text c="red">{t('scan.lastFailed', { time: dayjs(data.scan.last_failed_at).format("YYYY/MM/DD HH:mm") })}</Text>
        )}
        <Text>
          {t('summary', { length: data.findings.length, size: formatBytes(totalBytes, 2) })}
        </Text>

        <Box mt={5}>
          <TextInput
            placeholder={t('search')}
            leftSection={<IconSearch size={16} />}
            value={query}
            onChange={(e) => setQuery(e.currentTarget.value)}
            mb={10}
          />

          <DataTable<StorageFinding>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={records}
            columns={[
              { accessor: "relative_path", title: t('columns.path'), sortable: true },
              {
                accessor: "size_bytes",
                title: t('columns.size'),
                sortable: true,
                render: ({ size_bytes }) => <div>{formatBytes(size_bytes, 2)}</div>,
              },
              {
                accessor: "detected_at",
                title: t('columns.detectedAt'),
                sortable: true,
                render: ({ detected_at }) => (
                  <div>{dayjs(detected_at).format("YYYY/MM/DD HH:mm")}</div>
                ),
              },
              {
                accessor: "actions",
                title: t('columns.actions'),
                render: (finding) => (
                  <Group>
                    <ActionIcon
                      onClick={() => handleAction("import", [finding])}
                      className={classes.actionButton}
                      aria-label={t('actions.import')}
                      variant="light"
                      color="violet"
                    >
                      <IconFileImport size={18} />
                    </ActionIcon>
                    <ActionIcon
                      onClick={() => handleAction("delete", [finding])}
                      className={classes.actionButton}
                      aria-label={t('actions.delete')}
                      variant="light"
                      color="red"
                    >
                      <IconTrash size={18} />
                    </ActionIcon>
                  </Group>
                ),
              },
            ]}
            totalRecords={filteredCount}
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
            selectedRecords={selectedFindings}
            onSelectedRecordsChange={setSelectedFindings}
            recordsPerPageLabel={miscT('recordsPerPageLabel')}
          />
        </Box>
      </Container>

      <Modal opened={modalOpened} onClose={handleModalClose} title={modalAction === "delete" ? t('deleteModal') : t('importModal')}>
        {modalFindings.length >= 1 && (
          <StorageActionModalContent action={modalAction} findings={modalFindings} handleClose={handleModalClose} />
        )}
      </Modal>
    </div>
  );
}

export default AdminStoragePage;
