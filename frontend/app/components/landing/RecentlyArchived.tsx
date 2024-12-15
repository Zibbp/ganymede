import { rem, SimpleGrid, useMantineTheme } from "@mantine/core"
import { useMediaQuery } from "@mantine/hooks"
import VideoCard from "../videos/Card"
import { Carousel } from "@mantine/carousel";
import { useFetchVideosFilter } from "@/app/hooks/useVideos"

type Props = {
  count: number
}

const RecentlyArchived = ({ count }: Props) => {
  const theme = useMantineTheme()
  const isMobile = useMediaQuery(`(max-width: ${theme.breakpoints.sm})`);

  const { data, isPending, isError } = useFetchVideosFilter({ limit: count, offset: 0 })

  if (isPending) return (<div></div>)
  if (isError) return <div>Error loading recently archived videos</div>

  return (
    <div>
      {isMobile ? (
        <Carousel
          slideSize={{ base: '100%', sm: '50%' }}
          slideGap={{ base: rem(4), sm: 'xl' }}
          align="start"
          slidesToScroll={isMobile ? 1 : 2}
          controlSize={40}
          withIndicators
        >
          {data.data && data.data.map((item) => (
            <Carousel.Slide key={item.id}>
              <VideoCard video={item} showProgress={true} showChannel={true} showMenu={true} />
            </Carousel.Slide>
          ))}
        </Carousel>
      ) : (
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="xs" verticalSpacing="xs">
          {data.data && data.data.map((item) => (
            <VideoCard key={item.id} video={item} showProgress={true} showChannel={true} showMenu={true} />
          ))}
        </SimpleGrid>
      )}
    </div>
  );
}

export default RecentlyArchived;