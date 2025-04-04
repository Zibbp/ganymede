import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Channel, useDeleteChannel } from "@/app/hooks/useChannels";
import { Button, Code, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  channel: Channel | null
  handleClose: () => void;
}


const DeleteChannelModalContent = ({ channel, handleClose }: Props) => {
  const t = useTranslations('AdminChannelsComponents')
  const [loading, setLoading] = useState(false)

  const deleteChannelMutate = useDeleteChannel()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteChannel = async () => {
    if (!channel || !channel.id) return
    try {
      setLoading(true)

      await deleteChannelMutate.mutateAsync({ axiosPrivate: axiosPrivate, channelId: channel.id })

      showNotification({
        message: t('deleteNotification')
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
      <Text>{t('deleteConfirmText')}</Text>
      <Code block>{JSON.stringify(channel, null, 2)}</Code>
      <Button mt={5} color="red" onClick={handleDeleteChannel} loading={loading} fullWidth>{t('deleteButton')}</Button>
    </div>
  );
}

export default DeleteChannelModalContent;