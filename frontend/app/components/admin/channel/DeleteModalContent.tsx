import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Channel, useDeleteChannel } from "@/app/hooks/useChannels";
import { Button, Code, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";

type Props = {
  channel: Channel | null
  handleClose: () => void;
}


const DeleteChannelModalContent = ({ channel, handleClose }: Props) => {
  const [loading, setLoading] = useState(false)

  const deleteChannelMutate = useDeleteChannel()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteChannel = async () => {
    if (!channel || !channel.id) return
    try {
      setLoading(true)

      await deleteChannelMutate.mutateAsync({ axiosPrivate: axiosPrivate, channelId: channel.id })

      showNotification({
        message: "Channel deleted"
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
      <Text>Are you sure you want to delete this channel?</Text>
      <Code block>{JSON.stringify(channel, null, 2)}</Code>
      <Button mt={5} color="red" onClick={handleDeleteChannel} loading={loading} fullWidth>Delete Channel</Button>
    </div>
  );
}

export default DeleteChannelModalContent;