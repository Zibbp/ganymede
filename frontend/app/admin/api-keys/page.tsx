"use client"
import {
  ActionIcon,
  Badge,
  Box,
  Container,
  CopyButton,
  Group,
  Modal,
  TextInput,
  Title,
  Tooltip,
  Text,
} from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useEffect, useMemo, useState } from "react";
import sortBy from "lodash/sortBy";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { IconCheck, IconCopy, IconSearch, IconTrash } from "@tabler/icons-react";
import dayjs from "dayjs";
import classes from "./AdminApiKeysPage.module.css";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { ApiKey, ApiKeyScope, useGetApiKeys } from "@/app/hooks/useApiKeys";
import DeleteApiKeyModalContent from "@/app/components/admin/api-key/DeleteModalContent";
import { useTranslations } from "next-intl";
import { usePageTitle } from "@/app/util/util";
import useSettingsStore from "@/app/store/useSettingsStore";

// scopeBadgeColor maps an API key scope to a Mantine badge color so the
// permission level is visually distinct in the list at a glance.
const scopeBadgeColor = (scope: ApiKeyScope): string => {
  switch (scope) {
    case ApiKeyScope.Admin:
      return "red";
    case ApiKeyScope.Write:
      return "yellow";
    case ApiKeyScope.Read:
      return "blue";
    default:
      return "gray";
  }
};

const AdminApiKeysPage = () => {
  const t = useTranslations("AdminApiKeysPage");
  const miscT = useTranslations("MiscComponents");
  usePageTitle(t("title"));

  const settingsAdminItemsPerPage = useSettingsStore(
    (state) => state.adminItemsPerPage
  );
  const setSettingsAdminItemsPerPage = useSettingsStore(
    (state) => state.setAdminItemsPerPage
  );

  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(settingsAdminItemsPerPage);
  const [records, setRecords] = useState<ApiKey[]>([]);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<ApiKey>>({
    columnAccessor: "created_at",
    direction: "desc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [activeKey, setActiveKey] = useState<ApiKey | null>(null);

  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);

  const axiosPrivate = useAxiosPrivate();
  const { data: apiKeys, isPending, isError } = useGetApiKeys(axiosPrivate);

  useEffect(() => {
    setPerPage(settingsAdminItemsPerPage);
  }, [settingsAdminItemsPerPage]);

  useEffect(() => {
    if (!apiKeys) return;
    let filtered = [...apiKeys];

    if (debouncedQuery) {
      const q = debouncedQuery.toLowerCase();
      filtered = filtered.filter(
        (k) =>
          k.name.toLowerCase().includes(q) ||
          k.prefix.toLowerCase().includes(q) ||
          (k.description ?? "").toLowerCase().includes(q)
      );
    }

    const sorted = sortBy(filtered, sortStatus.columnAccessor);
    filtered = sortStatus.direction === "desc" ? sorted.reverse() : sorted;

    const from = (page - 1) * perPage;
    setRecords(filtered.slice(from, from + perPage));
  }, [apiKeys, page, perPage, sortStatus, debouncedQuery]);

  const totalRecords = useMemo(() => apiKeys?.length ?? 0, [apiKeys]);

  const handleDeleteModal = (key: ApiKey) => {
    setActiveKey(key);
    openDeleteModal();
  };

  if (isPending) return <GanymedeLoadingText message={t("loading")} />;
  if (isError) return <div>{t("error")}</div>;

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2}>
          <Title>{t("header")}</Title>
        </Group>

        <Box mt={5}>
          <TextInput
            placeholder={t("searchPlaceholder")}
            leftSection={<IconSearch size={16} />}
            value={query}
            onChange={(e) => setQuery(e.currentTarget.value)}
            mb={10}
          />

          <DataTable<ApiKey>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={records}
            columns={[
              {
                accessor: "name",
                title: t("columns.name"),
                sortable: true,
              },
              {
                accessor: "scope",
                title: t("columns.scope"),
                sortable: true,
                render: ({ scope }) => (
                  <Badge color={scopeBadgeColor(scope)} variant="light">
                    {scope}
                  </Badge>
                ),
              },
              {
                accessor: "prefix",
                title: t("columns.prefix"),
                render: ({ prefix }) => (
                  <Group gap="xs">
                    <Text ff="monospace">{prefix}</Text>
                    <CopyButton value={prefix} timeout={1500}>
                      {({ copied, copy }) => (
                        <Tooltip label={copied ? t("copied") : t("copyPrefix")}>
                          <ActionIcon size="sm" variant="subtle" onClick={copy}>
                            {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
                          </ActionIcon>
                        </Tooltip>
                      )}
                    </CopyButton>
                  </Group>
                ),
              },
              {
                accessor: "last_used_at",
                title: t("columns.lastUsed"),
                sortable: true,
                render: ({ last_used_at }) =>
                  last_used_at ? dayjs(last_used_at).format("YYYY/MM/DD HH:mm") : t("never"),
              },
              {
                accessor: "created_at",
                title: t("columns.createdAt"),
                sortable: true,
                render: ({ created_at }) => dayjs(created_at).format("YYYY/MM/DD"),
              },
              {
                accessor: "actions",
                title: t("columns.actions"),
                width: "100px",
                render: (key) => (
                  <Group>
                    <Tooltip label={t("revoke")}>
                      <ActionIcon
                        onClick={() => handleDeleteModal(key)}
                        className={classes.actionButton}
                        variant="light"
                        color="red"
                      >
                        <IconTrash size={18} />
                      </ActionIcon>
                    </Tooltip>
                  </Group>
                ),
              },
            ]}
            totalRecords={totalRecords}
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
            recordsPerPageLabel={miscT("recordsPerPageLabel")}
          />
        </Box>
      </Container>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t("deleteModal")}>
        {activeKey && (
          <DeleteApiKeyModalContent apiKey={activeKey} handleClose={closeDeleteModal} />
        )}
      </Modal>
    </div>
  );
};

export default AdminApiKeysPage;
