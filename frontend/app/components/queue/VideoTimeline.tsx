import { Timeline, Text, Modal } from "@mantine/core";
import { useState } from "react";
import classes from "./Timeline.module.css"
import { useDisclosure } from "@mantine/hooks";
import QueueTimelineBullet from "./TimelineBullet";
import QueueRestartTaskModalContent from "./RestartTaskModalContent";
import { Queue, QueueLogType, QueueTask } from "@/app/hooks/useQueue";
import { openQueueTaskLog } from "@/app/util/queue";
import { useTranslations } from "next-intl";

interface Params {
  queue: Queue;
}

const QueueVideoTimeline = ({ queue }: Params) => {
  const t = useTranslations('QueueComponents')
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
          title={t('videoDownloadTitle')}
        >
          <Text size="sm">
            {!queue.live_archive && (
              <span>
                <span
                  className={classes.restartText}
                  onClick={() => restartTask(QueueTask.TaskVideoDownload)}
                >
                  {t('restartButton')}
                </span>
                <span> - </span>
              </span>
            )}
            <span
              className={classes.restartText}
              onClick={() => openQueueTaskLog(queue.id, QueueLogType.Video)}
            >
              {t('logsButton')}
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_video_convert} />}
          title={t('videoConvertTitle')}
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVideoConvert)}
            >
              {t('restartButton')}
            </span>
            <span> - </span>
            <span
              className={classes.restartText}
              onClick={() => openQueueTaskLog(queue.id, QueueLogType.VideoConvert)}
            >
              {t('logsButton')}
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_video_move} />}
          title={t('videoMoveTitle')}
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVideoMove)}
            >
              {t('restartButton')}
            </span>
          </Text>
        </Timeline.Item>
      </Timeline>
      <Modal
        opened={restartTaskModalOpened}
        onClose={closeRestartTaskModal}
        title={t('restartQueueTaskModalTitle')}
      >
        {restartTaskName !== null && (
          <QueueRestartTaskModalContent queue={queue} task={restartTaskName} closeModal={closeRestartTaskModal} />
        )}
      </Modal>
    </div>
  );
}

export default QueueVideoTimeline;