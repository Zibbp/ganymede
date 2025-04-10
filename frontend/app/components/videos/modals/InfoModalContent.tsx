import { useFetchVideo, Video } from "@/app/hooks/useVideos";
import { Center, Code } from "@mantine/core";
import GanymedeLoadingText from "../../utils/GanymedeLoadingText";
import { useTranslations } from "next-intl";

type Props = {
  video: Video
}

const VideoInfoModalContent = ({ video }: Props) => {
  const t = useTranslations('VideoComponents')

  const { data, isPending, isError } = useFetchVideo({ id: video.id, with_channel: true, with_chapters: true, with_muted_segments: true })

  if (isPending) {
    return (
      <GanymedeLoadingText message={t('loadingInformation')} />
    );
  }

  if (isError) {
    return (
      <Center>
        <div>{t('errorLoadingInformation')}</div>
      </Center>
    );
  }

  return (
    <div>
      <Code block>{JSON.stringify(data, null, 2)}</Code>
    </div>
  );
}

export default VideoInfoModalContent;