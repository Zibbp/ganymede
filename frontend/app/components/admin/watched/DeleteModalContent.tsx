import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useBulkDeleteWatchedChannels, useDeleteWatchedChannel, WatchedChannel } from "@/app/hooks/useWatchedChannels";
import { Button, Code, List, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  watchedChannel: WatchedChannel | null
  // Set for bulk delete.
  watchedChannels?: WatchedChannel[]
  handleClose: () => void;
}


const DeleteWatchedChannelModalContent = ({ watchedChannel, watchedChannels, handleClose }: Props) => {
  const t = useTranslations('AdminWatchedComponents')
  const [loading, setLoading] = useState(false)

  const deleteWatchedChannelMutate = useDeleteWatchedChannel()
  const bulkDeleteWatchedChannelsMutate = useBulkDeleteWatchedChannels()
  const axiosPrivate = useAxiosPrivate()

  const isBulk = !!watchedChannels && watchedChannels.length > 0

  const handleDeleteChannel = async () => {
    try {
      setLoading(true)

      if (isBulk) {
        const result = await bulkDeleteWatchedChannelsMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          watchedChannelIds: watchedChannels.map((wc) => wc.id),
        })

        if (result.failed > 0) {
          showNotification({
            message: t('bulkDeletePartialNotification', { succeeded: result.succeeded, failed: result.failed }),
            color: result.succeeded > 0 ? "yellow" : "red",
          })
          if (result.succeeded === 0) return
        } else {
          showNotification({
            message: t('bulkDeleteNotification', { count: result.succeeded })
          })
        }
      } else {
        if (!watchedChannel || !watchedChannel.id) return

        await deleteWatchedChannelMutate.mutateAsync({ axiosPrivate: axiosPrivate, watchedChannelId: watchedChannel.id })

        showNotification({
          message: t('deleteNotification')
        })
      }

      handleClose()
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      {isBulk ? (
        <>
          <Text>{t('bulkDeleteConfirmText', { count: watchedChannels.length })}</Text>
          <List mt={5} size="sm" mah={300} style={{ overflowY: "auto" }}>
            {watchedChannels.map((wc) => (
              <List.Item key={wc.id}>{wc.edges.channel.display_name}</List.Item>
            ))}
          </List>
        </>
      ) : (
        <>
          <Text>{t('deleteConfirmText')}</Text>
          <Code block>{JSON.stringify(watchedChannel, null, 2)}</Code>
        </>
      )}
      <Button mt={5} color="red" onClick={handleDeleteChannel} loading={loading} fullWidth>
        {isBulk ? t('bulkDeleteButton', { count: watchedChannels.length }) : t('deleteButton')}
      </Button>
    </div>
  );
}

export default DeleteWatchedChannelModalContent;
