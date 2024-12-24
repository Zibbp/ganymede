import { Button, Code, Switch } from "@mantine/core";
import { Queue, QueueTask, useStartQueueTask } from "@/app/hooks/useQueue";
import { useState } from "react";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { showNotification } from "@mantine/notifications";

interface Params {
  queue: Queue;
  task: QueueTask;
  closeModal: () => void;
}

const QueueRestartTaskModalContent = ({ queue, task, closeModal }: Params) => {
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
        message: "Restarted queue task"
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
        Restart queue task <Code>{task}</Code>?
      </div>
      <div>
        <Switch
          mt={10}
          label="Continue with subsequent tasks"
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
          Restart Task
        </Button>
      </div>
    </div>
  );
}

export default QueueRestartTaskModalContent;