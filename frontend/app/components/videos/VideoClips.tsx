import { Video } from "@/app/hooks/useVideos";
import { Carousel } from "@mantine/carousel";
import { Title } from "@mantine/core";
import VideoCard from "./Card";

interface Params {
  clips: Video[];
}

const VideoPageClips = ({ clips }: Params) => {

  return (
    <div>
      <Title my={5}>Video Clips</Title>
      <Carousel
        withIndicators
        slideSize={{ base: '100%', sm: '50%', md: '33.333333%', lg: '25%', xl: '16.666666%' }}
        slideGap={{ base: 0, sm: 'md' }}
        loop
        align="start"
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