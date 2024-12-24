import { useGetVideoChatHistogram } from "@/app/hooks/useVideos";
import { BarChart } from '@mantine/charts';
import { Title } from "@mantine/core";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";

type Props = {
  videoId: string;
}

const VideoChatHistogram = ({ videoId }: Props) => {
  const { data, isPending, isError } = useGetVideoChatHistogram(videoId)

  if (isPending) {
    return <GanymedeLoadingText message="Loading Chat Histogram" />
  }
  if (isError) {
    return <div>Error loading chat histogram</div>
  }

  const secondsToHHMM = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
  };

  const result = Object.entries(data).map(([time, count]) => ({
    Time: secondsToHHMM(parseInt(time)),
    Messages: count,
  }));

  if (!result.length) {
    return <div>Error loading chat histogram</div>
  }

  return (
    <div>
      <Title my={5}>Chat Histogram</Title>
      <BarChart
        h={300}
        data={result}
        dataKey="Time"
        series={[
          { name: 'Messages', color: 'violet.6' },
        ]}
        tickLine="y"
      />
    </div>
  );
}

export default VideoChatHistogram;