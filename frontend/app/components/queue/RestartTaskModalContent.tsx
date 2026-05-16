import { Button, Code, Switch } from "@mantine/core";
import { Queue, QueueTask, useStartQueueTask } from "@/app/hooks/useQueue";
import { useState } from "react";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";

interface Params {
  queue: Queue;
  task: QueueTask;
  closeModal: () => void;
}

const QueueRestartTaskModalContent = ({ queue, task, closeModal }: Params) => {
  const t = useTranslations('QueueComponents')
  const [checked, setChecked] = useState(true);
  const [isLoading, setIsLoading] = useState(false);

  const axiosPrivate = useAxiosPrivate()

  const useStartQueueTaskMutate = useStartQueueTask()

  const restartTask = async () => {
    try {
      setIsLoading(true)

      await useStartQueueTaskMutate.mutateAsync({
        axiosPrivate: axiosPrivate, queueId: queue.id, taskName: task, continueWithSubsequent: checked
      })

      showNotification({
        message: t('taskRestartedNotification')
      })

      closeModal()

    } catch (error) {
      console.error(error)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div>
      <div>
        {t.rich('taskRestartText', {
          taskName: (chunks) => <code>{task}</code>,
        })}
      </div>
      <div>
        <Switch
          mt={10}
          label={t('continueWithSubsequentTasksLabel')}
          checked={checked}
          onChange={(event) => setChecked(event.currentTarget.checked)}
        />
      </div>
      <div>
        <Button
          onClick={() => restartTask()}
          fullWidth
          radius="md"
          mt="sm"
          size="md"
          color="green"
          loading={isLoading}
        >
          {t('restartTaskButton')}
        </Button>
      </div>
    </div>
  );
}

export default QueueRestartTaskModalContent;