import { useDisclosure } from "@mantine/hooks";
import { useState } from "react";
import QueueTimelineBullet from "./TimelineBullet";
import { Timeline, Text, Modal } from "@mantine/core";
import { Queue, QueueLogType, QueueTask } from "@/app/hooks/useQueue";
import { openQueueTaskLog } from "@/app/util/queue";
import classes from "./Timeline.module.css"
import QueueRestartTaskModalContent from "./RestartTaskModalContent";
import { useTranslations } from "next-intl";

interface Params {
  queue: Queue;
}

const QueueChatTimeline = ({ queue }: Params) => {
  const t = useTranslations('QueueComponents')
  const [restartTaskModalOpened, { open: openRestartTaskModal, close: closeRestartTaskModal }] = useDisclosure(false);

  const [restartTaskName, setRestartTaskName] = useState<QueueTask | null>(null);

  const restartTask = (task: QueueTask) => {
    setRestartTaskName(task);
    openRestartTaskModal();
  };

  return (
    <div>
      <Timeline
        active={0}
        bulletSize={24}
        color="dark"
        align="right"
        lineWidth={3}
      >
        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_chat_download} />}
          title={t('chatDownloadTitle')}
        >
          <Text size="sm">
            {!queue.live_archive && (
              <>
                <span>
                  <span
                    className={classes.restartText}
                    onClick={() => restartTask(QueueTask.TaskChatDownload)}
                  >
                    {t('restartButton')}
                  </span>
                  <span> - </span>
                </span>
                <span
                  className={classes.restartText}
                  onClick={() => openQueueTaskLog(queue.id, QueueLogType.Chat)}
                >
                  {t('logsButton')}
                </span>
              </>
            )}
          </Text>
        </Timeline.Item>

        {queue.live_archive && (
          <Timeline.Item
            bullet={<QueueTimelineBullet status={queue.task_chat_convert} />}
            title={t('chatConvertTitle')}
          >
            <Text size="sm">
              <span>
                <span
                  className={classes.restartText}
                  onClick={() => restartTask(QueueTask.TaskChatConvert)}
                >
                  {t('restartButton')}
                </span>
                <span> - </span>
              </span>
              <span
                className={classes.restartText}
                onClick={() => openQueueTaskLog(queue.id, QueueLogType.ChatConvert)}
              >
                {t('logsButton')}
              </span>
            </Text>
          </Timeline.Item>
        )}

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_chat_render} />}
          title={t('chatRenderTitle')}
        >
          <Text size="sm">
            <span>
              <span
                className={classes.restartText}
                onClick={() => restartTask(QueueTask.TaskChatRender)}
              >
                {t('restartButton')}
              </span>
              <span> - </span>
            </span>
            <span
              className={classes.restartText}
              onClick={() => openQueueTaskLog(queue.id, QueueLogType.ChatRender)}
            >
              {t('logsButton')}
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_chat_move} />}
          title={t('chatMoveTitle')}
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskChatMove)}
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

export default QueueChatTimeline;