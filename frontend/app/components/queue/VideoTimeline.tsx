import { Timeline, Text, Modal } from "@mantine/core";
import { useState } from "react";
import classes from "./Timeline.module.css"
import { useDisclosure } from "@mantine/hooks";
import QueueTimelineBullet from "./TimelineBullet";
import QueueRestartTaskModalContent from "./RestartTaskModalContent";
import { Queue, QueueLogType, QueueTask } from "@/app/hooks/useQueue";
import { openQueueTaskLog } from "@/app/util/queue";

interface Params {
  queue: Queue;
}

const QueueVideoTimeline = ({ queue }: Params) => {
  const [restartTaskModalOpened, { open: openRestartTaskModal, close: closeRestartTaskModal }] = useDisclosure(false);

  const [restartTaskName, setRestartTaskName] = useState<QueueTask | null>(null);

  const restartTask = (task: QueueTask) => {
    setRestartTaskName(task);
    openRestartTaskModal();
  };

  return (
    <div>
      <Timeline active={0} bulletSize={24} color="dark" lineWidth={3}>
        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_video_download} />}
          title="Video Download"
        >
          <Text size="sm">
            {!queue.live_archive && (
              <span>
                <span
                  className={classes.restartText}
                  onClick={() => restartTask(QueueTask.TaskVideoDownload)}
                >
                  restart
                </span>
                <span> - </span>
              </span>
            )}
            <span
              className={classes.restartText}
              onClick={() => openQueueTaskLog(queue.id, QueueLogType.Video)}
            >
              logs
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_video_convert} />}
          title="Video Convert"
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVideoConvert)}
            >
              restart
            </span>
            <span> - </span>
            <span
              className={classes.restartText}
              onClick={() => openQueueTaskLog(queue.id, QueueLogType.VideoConvert)}
            >
              logs
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_video_move} />}
          title="Video Move"
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVideoMove)}
            >
              restart
            </span>
          </Text>
        </Timeline.Item>
      </Timeline>
      <Modal
        opened={restartTaskModalOpened}
        onClose={closeRestartTaskModal}
        title="Restart Queue Task"
      >
        {restartTaskName !== null && (
          <QueueRestartTaskModalContent queue={queue} task={restartTaskName} closeModal={closeRestartTaskModal} />
        )}
      </Modal>
    </div>
  );
}

export default QueueVideoTimeline;