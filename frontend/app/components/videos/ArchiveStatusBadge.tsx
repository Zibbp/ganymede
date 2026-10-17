import { useRecordingStopping } from "@/app/store/useRecordingStopStore";
import { ArchiveStatus } from "@/app/util/archiveStatus";
import { Badge } from "@mantine/core";
import { useTranslations } from "next-intl";

export default function ArchiveStatusBadge({ videoId, status }: { videoId: string; status: ArchiveStatus }) {
  const t = useTranslations("ArchiveStatus");
  const stopping = useRecordingStopping(videoId, status);
  const colors: Record<ArchiveStatus, string> = {
    queued: "gray", running: "blue", finalizing: "orange", completed: "green", failed: "red",
  };
  return <Badge color={stopping ? "orange" : colors[status]} variant="light">{t(stopping ? "stopping" : status)}</Badge>;
}
