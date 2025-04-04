import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Queue, useDeleteQueue } from "@/app/hooks/useQueue";
import { Button, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  queues: Queue[]
  handleClose: () => void;
}

const MultiDeleteQueueModalContent = ({ queues, handleClose }: Props) => {
  const t = useTranslations('AdminQueueComponents')
  const [loading, setLoading] = useState(false)
  const deleteQueueMutate = useDeleteQueue()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteQueue = async () => {
    setLoading(true)
    try {
      await Promise.all(queues.map(async (queue) => {

        await deleteQueueMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          queueId: queue.id,
        })
      }))

      showNotification({
        message: t('multiDeleteNotification')
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
      <Text>{t('multiDeleteConfirmText', { number: queues.length })}</Text>
      <Button mt={5} color="red" onClick={handleDeleteQueue} loading={loading} fullWidth>{t('multiDeleteButton')}</Button>
    </div>
  );
}

export default MultiDeleteQueueModalContent;