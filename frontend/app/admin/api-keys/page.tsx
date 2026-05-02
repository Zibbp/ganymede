"use client"
import {
  ActionIcon,
  Badge,
  Box,
  Button,
  Container,
  CopyButton,
  Drawer,
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
import { IconCheck, IconCopy, IconPlus, IconSearch, IconTrash } from "@tabler/icons-react";
import dayjs from "dayjs";
import classes from "./AdminApiKeysPage.module.css";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { ApiKey, ApiKeyTier, useGetApiKeys } from "@/app/hooks/useApiKeys";
import AdminApiKeyDrawerContent from "@/app/components/admin/api-key/DrawerContent";
import DeleteApiKeyModalContent from "@/app/components/admin/api-key/DeleteModalContent";
import ShowOnceModalContent from "@/app/components/admin/api-key/ShowOnceModal";
import { useTranslations } from "next-intl";
import { usePageTitle } from "@/app/util/util";
import useSettingsStore from "@/app/store/useSettingsStore";

// scopeBadgeColor picks a badge color from a scope's tier so admin
// destructive perms are red, write is yellow, read is blue, and any
// unrecognised tier falls back to gray.
const scopeBadgeColor = (scope: string): string => {
  const tier = scope.split(":", 2)[1] ?? "";
  switch (tier as ApiKeyTier) {
    case ApiKeyTier.Admin:
      return "red";
    case ApiKeyTier.Write:
      return "yellow";
    case ApiKeyTier.Read:
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
  const [showOnceSecret, setShowOnceSecret] = useState<string | null>(null);

  const [createDrawerOpened, { open: openCreateDrawer, close: closeCreateDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [showOnceOpened, { open: openShowOnceModal, close: closeShowOnceModalRaw }] = useDisclosure(false);

  // Wrap close so the secret is wiped from React state as soon as the
  // modal is dismissed — keeps the value out of memory longer than the
  // user actually needs it.
  const closeShowOnceModal = () => {
    setShowOnceSecret(null);
    closeShowOnceModalRaw();
  };

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
          (k.description ?? "").toLowerCase().includes(q) ||
          (k.scopes ?? []).some((s) => s.toLowerCase().includes(q))
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

  // Called by the create drawer once the API has minted a key. We surface
  // the full secret in the show-once modal exactly here; the secret is
  // never persisted in the table, query cache, or Zustand store.
  const handleCreated = (secret: string) => {
    closeCreateDrawer();
    setShowOnceSecret(secret);
    openShowOnceModal();
  };

  if (isPending) return <GanymedeLoadingText message={t("loading")} />;
  if (isError) return <div>{t("error")}</div>;

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2}>
          <Title>{t("header")}</Title>
          <Button leftSection={<IconPlus size={16} />} onClick={openCreateDrawer}>
            {t("createButton")}
          </Button>
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
                accessor: "scopes",
                title: t("columns.scopes"),
                // A key can hold multiple scopes; render one badge per
                // scope so admins can see the full grant at a glance.
                render: ({ scopes }) => (
                  <Group gap={4}>
                    {(scopes ?? []).map((s) => (
                      <Badge
                        key={s}
                        color={scopeBadgeColor(s)}
                        variant="light"
                        ff="monospace"
                      >
                        {s}
                      </Badge>
                    ))}
                  </Group>
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

      <Drawer
        opened={createDrawerOpened}
        onClose={closeCreateDrawer}
        position="right"
        title={t("drawer")}
      >
        <AdminApiKeyDrawerContent onCreated={handleCreated} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t("deleteModal")}>
        {activeKey && (
          <DeleteApiKeyModalContent apiKey={activeKey} handleClose={closeDeleteModal} />
        )}
      </Modal>

      <Modal
        opened={showOnceOpened}
        onClose={closeShowOnceModal}
        title={t("showOnceModal")}
        closeOnClickOutside={false}
        closeOnEscape={false}
        withCloseButton={false}
        size="lg"
      >
        {showOnceSecret && (
          <ShowOnceModalContent secret={showOnceSecret} handleClose={closeShowOnceModal} />
        )}
      </Modal>
    </div>
  );
};

export default AdminApiKeysPage;
