import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Playlist, useCreatePlaylist, useEditPlaylist } from "@/app/hooks/usePlaylist";
import { Button, TextInput } from "@mantine/core";
import { useForm } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";

type Props = {
  playlist: Playlist | null
  mode: PlaylistEditFormMode
  handleClose: () => void;
}

export enum PlaylistEditFormMode {
  Create = "create",
  Edit = "edit",
}

const PlaylistEditForm = ({ playlist, mode, handleClose }: Props) => {
  const [editButtonLoading, setEditButtonLoading] = useState(false)

  const axiosPrivate = useAxiosPrivate();

  const createPlaylistMutation = useCreatePlaylist()
  const editPlaylistMutation = useEditPlaylist()


  const form = useForm({
    mode: 'uncontrolled',
    initialValues: {
      name: playlist?.name || "",
      description: playlist?.description || "",
    },
  })

  const handleSubmit = async (values: typeof form.values) => {
    try {
      setEditButtonLoading(true)

      if (mode == PlaylistEditFormMode.Edit) {
        await editPlaylistMutation.mutateAsync({
          axiosPrivate,
          id: playlist?.id || "",
          name: values.name,
          description: values.description
        })
      } else {
        await createPlaylistMutation.mutateAsync({ axiosPrivate, name: values.name, description: values.description })
      }

      setEditButtonLoading(false)

      showNotification({
        message: `Playlist ${mode == PlaylistEditFormMode.Edit ? "edited" : "created"}`
      })

      handleClose()

    } catch (error) {
      console.error("Error creating playlist", error)
      setEditButtonLoading(false)
    }
  };

  return (
    <div>
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <TextInput
          label="Name"
          description="Playlist name"
          key={form.key('name')}
          {...form.getInputProps('name')}
          required
        />
        <TextInput
          label="Description"
          description="Playlist description"
          key={form.key('description')}
          {...form.getInputProps('description')}
        />
        <Button mt={10} type="submit" fullWidth loading={editButtonLoading}>{mode == PlaylistEditFormMode.Create ? 'Create Playlist' : 'Edit Playlist'}</Button>
      </form>
    </div>
  );
}

export default PlaylistEditForm;