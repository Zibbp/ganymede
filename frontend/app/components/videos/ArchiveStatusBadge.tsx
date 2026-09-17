import { ArchiveStatus } from "@/app/util/archiveStatus";
import { Badge } from "@mantine/core";
import { useTranslations } from "next-intl";

export default function ArchiveStatusBadge({ status }: { status: ArchiveStatus }) {
  const t = useTranslations("ArchiveStatus");
  const colors: Record<ArchiveStatus, string> = {
    queued: "gray", running: "blue", finalizing: "orange", completed: "green", failed: "red",
  };
  return <Badge color={colors[status]} variant="light">{t(status)}</Badge>;
}
