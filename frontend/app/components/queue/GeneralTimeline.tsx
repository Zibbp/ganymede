"use client"
import { Timeline, Text, Modal } from "@mantine/core";
import classes from "./Timeline.module.css"
import QueueTimelineBullet from "./TimelineBullet";
import { Queue, QueueTask } from "@/app/hooks/useQueue";
import { useDisclosure } from "@mantine/hooks";
import { useState } from "react";
import QueueRestartTaskModalContent from "./RestartTaskModalContent";
import { useTranslations } from "next-intl";

interface Params {
  queue: Queue;
}

const QueueGeneralTimeline = ({ queue }: Params) => {
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
          bullet={<QueueTimelineBullet status={queue.task_vod_create_folder} />}
          title={t('createFolderTitle')}
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVodCreateFolder)}
            >
              {t('restartButton')}
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_vod_save_info} />}
          title={t('saveInformationTitle')}
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVodSaveInfo)}
            >
              {t('restartButton')}
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={
            <QueueTimelineBullet status={queue.task_vod_download_thumbnail} />
          }
          title={t('downloadThumbnailsTitle')}
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVodDownloadThumbnail)}
            >
              {t('restartButton')}
            </span>
          </Text>
        </Timeline.Item>
      </Timeline>
      <Modal
        opened={restartTaskModalOpened}
        onClose={closeRestartTaskModal}
        title={t("restartQueueTaskModalTitle")}
      >
        {restartTaskName !== null && (
          <QueueRestartTaskModalContent queue={queue} task={restartTaskName} closeModal={closeRestartTaskModal} />
        )}
      </Modal>
    </div>
  );
}

export default QueueGeneralTimeline;