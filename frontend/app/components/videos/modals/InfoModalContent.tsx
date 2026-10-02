import { useFetchVideo, useGetVideoFFprobe, Video } from "@/app/hooks/useVideos";
import { Center, Code, Tabs } from "@mantine/core";
import GanymedeLoadingText from "../../utils/GanymedeLoadingText";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  video: Video
}

const VideoInfoModalContent = ({ video }: Props) => {
  const t = useTranslations('VideoComponents')
  const [activeSection, setActiveSection] = useState<string | null>('information');

  const { data, isPending, isError } = useFetchVideo({ id: video.id, with_channel: true, with_chapters: true, with_muted_segments: true })
  const {
    data: ffprobeData,
    isPending: isPendingFFprobe,
    isError: isErrorFFprobe,
  } = useGetVideoFFprobe(video.id, activeSection === 'ffprobe');

  const renderInformation = () => {
    if (isPending) {
      return <GanymedeLoadingText message={t('loadingInformation')} />;
    }

    if (isError) {
      return (
        <Center>
          <div>{t('errorLoadingInformation')}</div>
        </Center>
      );
    }

    return <Code block>{JSON.stringify(data, null, 2)}</Code>;
  };

  const renderFFprobe = () => {
    if (isPendingFFprobe) {
      return <GanymedeLoadingText message={t('loadingInformation')} />;
    }

    if (isErrorFFprobe) {
      return (
        <Center>
          <div>{t('errorLoadingInformation')}</div>
        </Center>
      );
    }

    return <Code block>{JSON.stringify(ffprobeData, null, 2)}</Code>;
  };

  return (
    <Tabs value={activeSection} onChange={setActiveSection}>
      <Tabs.List>
        <Tabs.Tab value="information">
          {t("videoInformationModal.informationTitle")}
        </Tabs.Tab>
        <Tabs.Tab value="ffprobe">
          {t("videoInformationModal.ffprobeTitle")}
        </Tabs.Tab>
      </Tabs.List>

      <Tabs.Panel value="information" pt="md">
        {renderInformation()}
      </Tabs.Panel>
      <Tabs.Panel value="ffprobe" pt="md">
        {renderFFprobe()}
      </Tabs.Panel>
    </Tabs>
  );
}

export default VideoInfoModalContent;
