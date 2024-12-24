import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { } from "@/app/hooks/useChannels";
import { useDeleteWatchedChannel, WatchedChannel } from "@/app/hooks/useWatchedChannels";
import { Button, Code, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";

type Props = {
  watchedChannel: WatchedChannel | null
  handleClose: () => void;
}


const DeleteWatchedChannelModalContent = ({ watchedChannel, handleClose }: Props) => {
  const [loading, setLoading] = useState(false)

  const deleteWatchedChannelMutate = useDeleteWatchedChannel()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteChannel = async () => {
    if (!watchedChannel || !watchedChannel.id) return
    try {
      setLoading(true)

      await deleteWatchedChannelMutate.mutateAsync({ axiosPrivate: axiosPrivate, watchedChannelId: watchedChannel.id })

      showNotification({
        message: "Watched channel deleted"
      })

      handleClose()
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Text>Are you sure you want to delete this watched channel?</Text>
      <Code block>{JSON.stringify(watchedChannel, null, 2)}</Code>
      <Button mt={5} color="red" onClick={handleDeleteChannel} loading={loading} fullWidth>Delete Watched Channel</Button>
    </div>
  );
}

export default DeleteWatchedChannelModalContent;