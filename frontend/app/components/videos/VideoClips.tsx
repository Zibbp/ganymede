import { Video } from "@/app/hooks/useVideos";
import { Carousel } from "@mantine/carousel";
import { Title } from "@mantine/core";
import VideoCard from "./Card";
import { useTranslations } from "next-intl";

interface Params {
  clips: Video[];
}

const VideoPageClips = ({ clips }: Params) => {
  const t = useTranslations("VideoComponents");

  return (
    <div>
      <Title my={5}>{t('videoClipsTitle')}</Title>
      <Carousel
        withIndicators
        slideSize={{ base: '100%', sm: '50%', md: '33.333333%', lg: '25%', xl: '16.666666%' }}
        slideGap={{ base: 0, sm: 'md' }}
        emblaOptions={{
          align: "start",
          loop: true,
        }}
      >
        {clips.map((video) => (
          <Carousel.Slide key={video.id}>
            <VideoCard
              key={video.id}
              video={video}
              showChannel={false}
              showMenu={true}
              showProgress={true}
            />
          </Carousel.Slide>
        ))}
      </Carousel>
    </div>
  );
}

export default VideoPageClips;