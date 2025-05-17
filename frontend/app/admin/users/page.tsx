"use client"
import { ActionIcon, Container, Group, TextInput, Title, Box, Drawer, Modal, Tooltip, Text } from "@mantine/core";
import { useDebouncedValue, useDisclosure } from "@mantine/hooks";
import { DataTable, DataTableSortStatus } from "mantine-datatable";
import { useEffect, useState } from "react";
import sortBy from "lodash/sortBy";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { IconPencil, IconSearch, IconTrash } from "@tabler/icons-react";
import dayjs from "dayjs";
import classes from "./AdminUsersPage.module.css"
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { User } from "@/app/hooks/useAuthentication";
import { useGetUsers } from "@/app/hooks/useUsers";
import AdminUserDrawerContent from "@/app/components/admin/user/DrawerContent";
import DeleteUserModalContent from "@/app/components/admin/user/DeleteModalContent";
import { useTranslations } from "next-intl";
import { usePageTitle } from "@/app/util/util";

const AdminUsersPage = () => {
  const t = useTranslations('AdminUsersPage')
  usePageTitle(t('title'))
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);
  const [records, setRecords] = useState<User[]>([]);
  const [initialRecords, setInitialRecords] = useState(false);
  const [sortStatus, setSortStatus] = useState<DataTableSortStatus<User>>({
    columnAccessor: "name",
    direction: "asc",
  });
  const [query, setQuery] = useState("");
  const [debouncedQuery] = useDebouncedValue(query, 500);
  const [activeUser, setActiveUser] = useState<User | null>(null);

  const [userDrawerOpened, { open: openUserDrawer, close: closeUserDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);

  const axiosPrivate = useAxiosPrivate()
  const { data: users, isPending, isError } = useGetUsers(axiosPrivate)

  useEffect(() => {
    if (!users || users.length == 0) return;

    let filteredData = [...users] as User[];

    // Apply search filter
    if (debouncedQuery) {
      filteredData = filteredData.filter((user) => {
        return (
          user.id.toString().includes(debouncedQuery) ||
          user.username.includes(debouncedQuery)
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
  }, [users, page, perPage, sortStatus, debouncedQuery, initialRecords]);

  const openDrawer = (user: User) => {
    setActiveUser(user);
    openUserDrawer()
  };

  const handleDeleteModal = (user: User) => {
    setActiveUser(user);
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
          <DataTable<User>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={records}
            columns={[
              {
                accessor: "id", title: t('columns.id'),
                render: ({ id }) => (
                  <Tooltip label={id}>
                    <Text lineClamp={1}>{id}</Text>
                  </Tooltip>
                ),
              },
              {
                accessor: "username",
                title: t('columns.username'),
                sortable: true
              },
              {
                accessor: "role",
                title: t('columns.role'),
                sortable: true
              },
              {
                accessor: "oauth",
                title: t('columns.authMethod'),
                sortable: true,
                render: ({ oauth }) => {
                  return oauth ? "SSO" : "Local";
                },
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
                width: "150px",
                render: (user) => (
                  <Group>
                    <ActionIcon
                      onClick={() => openDrawer(user)}
                      className={classes.actionButton}
                      variant="light"
                    >
                      <IconPencil size={18} />
                    </ActionIcon>
                    <ActionIcon
                      onClick={() => handleDeleteModal(user)}
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
            totalRecords={users?.length ?? 0}
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

      <Drawer opened={userDrawerOpened} onClose={closeUserDrawer} position="right" title={t('drawer')}>
        {activeUser && (
          <AdminUserDrawerContent user={activeUser} handleClose={closeUserDrawer} />
        )}
      </Drawer>


      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('deleteModal')}>
        {activeUser && (
          <DeleteUserModalContent user={activeUser} handleClose={closeDeleteModal} />
        )}
      </Modal>

    </div>
  );
}

export default AdminUsersPage;