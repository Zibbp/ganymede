import { useFetchVideo, useGetVideoFFprobe, Video } from "@/app/hooks/useVideos";
import { Center, Code, Title } from "@mantine/core";
import GanymedeLoadingText from "../../utils/GanymedeLoadingText";
import { useTranslations } from "next-intl";

type Props = {
  video: Video
}

const VideoInfoModalContent = ({ video }: Props) => {
  const t = useTranslations('VideoComponents')

  // Get video information and ffprobe data
  const { data, isPending, isError } = useFetchVideo({ id: video.id, with_channel: true, with_chapters: true, with_muted_segments: true })
  const { data: ffprobeData, isPending: isPendingFFprobe, isError: isErrorFFprobe } = useGetVideoFFprobe(video.id);

  if (isPending || isPendingFFprobe) {
    return (
      <GanymedeLoadingText message={t('loadingInformation')} />
    );
  }

  if (isError || isErrorFFprobe) {
    return (
      <Center>
        <div>{t('errorLoadingInformation')}</div>
      </Center>
    );
  }

  return (
    <div>
      <Title order={4}>{t("videoInformationModal.informationTitle")}</Title>
      <Code block>{JSON.stringify(data, null, 2)}</Code>
      <Title order={4} mt={10}>{t("videoInformationModal.ffprobeTitle")}</Title>
      <Code block mt="md">{JSON.stringify(ffprobeData, null, 2)}</Code>
    </div >
  );
}

export default VideoInfoModalContent;