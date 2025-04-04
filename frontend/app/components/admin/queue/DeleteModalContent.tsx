import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Queue, useDeleteQueue } from "@/app/hooks/useQueue";
import { Button, Code, Flex, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  queue: Queue
  handleClose: () => void;
}


const DeleteQueueModalContent = ({ queue, handleClose }: Props) => {
  const t = useTranslations('AdminQueueComponents')
  const [loading, setLoading] = useState(false)

  const deleteQueueMutate = useDeleteQueue()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteQueue = async () => {
    try {
      setLoading(true)

      await deleteQueueMutate.mutateAsync({ axiosPrivate: axiosPrivate, queueId: queue.id })

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
      <Flex>
        ID: <Code ml={3}>{queue.id}</Code>
      </Flex>

      <Text fs={"italic"} fz={"sm"}>
        {t('deleteConfirmText2')}
      </Text>

      <Button mt={5} color="red" onClick={handleDeleteQueue} loading={loading} fullWidth>{t('deleteButton')}</Button>
    </div>
  );
}

export default DeleteQueueModalContent;