import { useGetVideoChatHistogram } from "@/app/hooks/useVideos";
import { BarChart } from '@mantine/charts';
import { Title } from "@mantine/core";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { RefObject } from "react";
import { MediaPlayerInstance } from "@vidstack/react";
import { useTranslations } from "next-intl";

type Props = {
  videoId: string;
  playerRef: RefObject<MediaPlayerInstance | null>;
}

const VideoChatHistogram = ({ videoId, playerRef }: Props) => {
  const t = useTranslations('VideoComponents')
  const { data, isPending, isError } = useGetVideoChatHistogram(videoId)

  if (isPending) {
    return <GanymedeLoadingText message={t('loadingChatHistogram')} />
  }
  if (isError) {
    return <div>{t('errorLoadingChatHistogram')}</div>
  }

  const secondsToHHMM = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
  };

  const HHMMToseconds = (str: string): number => {
    const p = str.split(':');
    return Number.parseInt(p[0]) * 3600 + Number.parseInt(p[1]) * 60;
  }

  const result = Object.entries(data).map(([time, count]) => ({
    Time: secondsToHHMM(parseInt(time)),
    Messages: count,
  }));

  if (!result.length) {
    return <div>{t('errorLoadingChatHistogram')}</div>
  }

  return (
    <div>
      <Title my={5}>{t('chatHistogramTitle')}</Title>
      <BarChart
        h={300}
        data={result}
        dataKey="Time"
        barChartProps={{
          style: { cursor: "pointer" },
          onClick: (data) => playerRef.current!.currentTime = HHMMToseconds(data.activeLabel!)
        }}
        series={[
          { name: 'Messages', color: 'violet.6' },
        ]}
        tickLine="y"
      />
    </div>
  );
}

export default VideoChatHistogram;