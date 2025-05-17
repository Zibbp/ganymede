"use client"
import { Text, ActionIcon, Button, Container, Group, Title, Drawer, Modal, Code } from "@mantine/core";
import { Playlist, useDeletePlaylist, useGetPlaylists } from "../hooks/usePlaylist";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import useAuthStore from "../store/useAuthStore";
import { IconEdit, IconTrash } from "@tabler/icons-react";
import { UserRole } from "../hooks/useAuthentication";
import { DataTable } from 'mantine-datatable';
import Link from "next/link"
import { useDisclosure } from "@mantine/hooks";
import PlaylistEditForm, { PlaylistEditFormMode } from "../components/playlist/EditForm";
import { useEffect, useState } from "react";
import { showNotification } from "@mantine/notifications";
import { useAxiosPrivate } from "../hooks/useAxios";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { usePageTitle } from "../util/util";


const PlaylistsPage = () => {
  const t = useTranslations("PlaylistsPage");
  usePageTitle(t('title'));

  const hasPermission = useAuthStore(state => state.hasPermission);

  const { data: playlists, isPending, isError } = useGetPlaylists()

  const [playlist, setPlaylist] = useState<Playlist | null>(null);
  const [playlistEditMode, setPlaylistEditMode] = useState<PlaylistEditFormMode>(PlaylistEditFormMode.Edit)

  const [playlistDrawerOpened, { open: openPlaylistDrawer, close: closePlaylistDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);

  const axiosPrivate = useAxiosPrivate();
  const deletePlaylistMutation = useDeletePlaylist()
  const [deleteButtonLoading, setDeleteButtonLoading] = useState(false);

  const queryClient = useQueryClient()

  const handleDelete = async (id: string) => {
    try {
      setDeleteButtonLoading(true)

      await deletePlaylistMutation.mutateAsync({ axiosPrivate, id })

      setDeleteButtonLoading(false)

      showNotification({
        message: t('deleted')
      })

      queryClient.invalidateQueries({ queryKey: ["queue"] })

      closeDeleteModal()

    } catch (error) {
      console.error(t('errorDelete'), error)
      setDeleteButtonLoading(false)
    }
  };

  const closeDrawerCallback = () => {
    closePlaylistDrawer()
  }

  if (isPending) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isError) return <div>{t('error')}</div>

  return (
    <div>
      <Container size={"7xl"}>
        <Group justify="space-between">
          <Title>{t('title')}</Title>
          {hasPermission(UserRole.Editor) && (
            <div>
              <Button variant="default" onClick={() => {
                setPlaylistEditMode(PlaylistEditFormMode.Create)
                setPlaylist(null)
                openPlaylistDrawer()
              }}>{t('create')}</Button>
            </div>
          )}
        </Group>

        {/* table */}
        <DataTable
          highlightOnHover
          columns={[
            {
              accessor: "name",
              title: t('playlist.name'),
              render: ({ name, id }) => (
                <Link href={`/playlists/${id}`}>
                  <Text>{name}</Text>
                </Link>
              ),
            },
            {
              accessor: "description",
              title: t('playlist.description'),
              render: ({ description }) => (
                <Text lineClamp={1}>{description}</Text>
              ),
            },
            {
              accessor: '',
              textAlign: 'right',
              render: (playlist) => hasPermission(UserRole.Editor) && (
                <Group gap={4} justify="right" wrap="nowrap">
                  <ActionIcon
                    size="sm"
                    variant="subtle"
                    color="blue"
                    onClick={() => {
                      setPlaylistEditMode(PlaylistEditFormMode.Edit)
                      setPlaylist(playlist)
                      openPlaylistDrawer()
                    }}
                    title={t('playlist.edit')}
                    aria-label="Edit"
                  >
                    <IconEdit size={16} />
                  </ActionIcon>
                  <ActionIcon
                    size="sm"
                    variant="subtle"
                    color="red"
                    onClick={() => {
                      setPlaylist(playlist)
                      openDeleteModal()
                    }}
                    title={t('playlist.delete')}
                    arria-label="Delete"
                  >
                    <IconTrash size={16} />
                  </ActionIcon>
                </Group>
              )
            },
          ]}
          records={playlists}
        />

      </Container>

      <Drawer opened={playlistDrawerOpened} onClose={closePlaylistDrawer} position="right" title={t('title')}>
        <PlaylistEditForm mode={playlistEditMode} playlist={playlist} handleClose={closeDrawerCallback} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('deletePlaylist')} centered>
        <div>
          <Code block>{JSON.stringify(playlist, null, 2)}</Code>
          <Button mt={10} fullWidth color="red" variant="filled" loading={deleteButtonLoading} onClick={() => handleDelete(playlist?.id || "")}>{t('playlist.delete')}</Button>
        </div>
      </Modal>
    </div>
  );
}

export default PlaylistsPage;