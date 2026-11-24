import { useRecordingStopping } from "@/app/store/useRecordingStopStore";
import { Video } from "@/app/hooks/useVideos";
import { mediaSrc } from "@/app/util/util";
import { BackgroundImage, Overlay, Stack, Text, Title } from "@mantine/core";
import { useTranslations } from "next-intl";
import classes from "./RecordingNotice.module.css";

export default function RecordingNotice({ video }: { video: Video }) {
  const t = useTranslations("ArchiveStatus");
  const stopping = useRecordingStopping(video.id, video.status);

  return (
    <BackgroundImage
      src={mediaSrc(video.web_thumbnail_path)}
      className={classes.container}
    >
      <Overlay color="#000" backgroundOpacity={0.75} zIndex={0} />
      <Stack align="center" gap="sm" p="xl" pos="relative" role="status">
        <Title order={2}>{t(stopping ? "stopping" : video.status)}</Title>
        <Text ta="center" maw={480}>{t("livePreviewUnavailable")}</Text>
      </Stack>
    </BackgroundImage>
  );
}
