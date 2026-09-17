import { Video } from "@/app/hooks/useVideos";
import { escapeURL } from "@/app/util/util";
import { BackgroundImage, Overlay, Stack, Text, Title } from "@mantine/core";
import { useTranslations } from "next-intl";
import { env } from "next-runtime-env";
import classes from "./RecordingNotice.module.css";

export default function RecordingNotice({ video }: { video: Video }) {
  const t = useTranslations("ArchiveStatus");

  return (
    <BackgroundImage
      src={`${env("NEXT_PUBLIC_CDN_URL") ?? ""}${escapeURL(video.web_thumbnail_path)}`}
      className={classes.container}
    >
      <Overlay color="#000" backgroundOpacity={0.75} zIndex={0} />
      <Stack align="center" gap="sm" p="xl" pos="relative" role="status">
        <Title order={2}>{t(video.status)}</Title>
        <Text ta="center" maw={480}>{t("livePreviewUnavailable")}</Text>
      </Stack>
    </BackgroundImage>
  );
}
