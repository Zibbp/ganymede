"use client"
import { Timeline, Text, Modal } from "@mantine/core";
import classes from "./Timeline.module.css"
import QueueTimelineBullet from "./TimelineBullet";
import { Queue, QueueTask } from "@/app/hooks/useQueue";
import { useDisclosure } from "@mantine/hooks";
import { useState } from "react";
import QueueRestartTaskModalContent from "./RestartTaskModalContent";

interface Params {
  queue: Queue;
}

const QueueGeneralTimeline = ({ queue }: Params) => {
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
          title="Create Folder"
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVodCreateFolder)}
            >
              restart
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={<QueueTimelineBullet status={queue.task_vod_save_info} />}
          title="Save Information"
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVodSaveInfo)}
            >
              restart
            </span>
          </Text>
        </Timeline.Item>

        <Timeline.Item
          bullet={
            <QueueTimelineBullet status={queue.task_vod_download_thumbnail} />
          }
          title="Download Thumbnails"
        >
          <Text size="sm">
            <span
              className={classes.restartText}
              onClick={() => restartTask(QueueTask.TaskVodDownloadThumbnail)}
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

export default QueueGeneralTimeline;