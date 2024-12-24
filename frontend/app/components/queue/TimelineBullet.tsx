import { QueueTaskStatus } from "@/app/hooks/useQueue";
import { Loader, ThemeIcon } from "@mantine/core";
import { IconCircleCheck, IconCircleX, IconHourglass } from "@tabler/icons-react";

interface Props {
  status: QueueTaskStatus
}

const QueueTimelineBullet = ({ status }: Props) => {
  if (status == QueueTaskStatus.Running) {
    return (
      <div style={{ marginTop: "6px", marginLeft: "0.3px" }}>
        <Loader size="sm" color="green" />
      </div>
    );
  }
  if (status == QueueTaskStatus.Success) {
    return (
      <div style={{ marginTop: "5px" }}>
        <ThemeIcon radius="xl" color="green">
          <IconCircleCheck />
        </ThemeIcon>
      </div>
    );
  }
  if (status == QueueTaskStatus.Failed) {
    return (
      <div style={{ marginTop: "5px" }}>
        <ThemeIcon radius="xl" color="red">
          <IconCircleX />
        </ThemeIcon>
      </div>
    );
  }
  if (status == QueueTaskStatus.Pending) {
    return (
      <div style={{ marginTop: "5px" }}>
        <ThemeIcon radius="xl" color="blue">
          <IconHourglass />
        </ThemeIcon>
      </div>
    );
  }
};

export default QueueTimelineBullet;