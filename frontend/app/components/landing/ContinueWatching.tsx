import { useAxiosPrivate } from "@/app/hooks/useAxios"
import { useGetLastPlaybackVideos } from "@/app/hooks/usePlayback"
import { rem, SimpleGrid, useMantineTheme } from "@mantine/core"
import { useMediaQuery } from "@mantine/hooks"
import VideoCard from "../videos/Card"
import { Carousel } from "@mantine/carousel";
import { useTranslations } from "next-intl"

type Props = {
  count: number
}

const ContinueWatching = ({ count }: Props) => {
  const axiosPrivate = useAxiosPrivate()
  const theme = useMantineTheme()
  const isMobile = useMediaQuery(`(max-width: ${theme.breakpoints.sm})`);

  const t = useTranslations("LandingContinueWatchingComponent")

  const { data, isPending, isError } = useGetLastPlaybackVideos(axiosPrivate, count)

  if (isPending) return (<div></div>)
  if (isError) return <div>{t('error')}</div>


  return (
    <div>
      {isMobile ? (
        <Carousel
          slideSize={{ base: '100%', sm: '50%' }}
          slideGap={{ base: rem(4), sm: 'xl' }}
          controlSize={40}
          withIndicators
          emblaOptions={{
            align: 'start',
            slidesToScroll: isMobile ? 1 : 2
          }}
        >
          {data.data && data.data.map((item) => (
            <Carousel.Slide key={item.vod.id}>
              <VideoCard video={item.vod} showProgress={true} showChannel={true} showMenu={true} />
            </Carousel.Slide>
          ))}
        </Carousel>
      ) : (
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="xs" verticalSpacing="xs">
          {data.data && data.data.map((item) => (
            <VideoCard key={item.vod.id} video={item.vod} showProgress={true} showChannel={true} showMenu={true} />
          ))}
        </SimpleGrid>
      )}
    </div>
  );
}

export default ContinueWatching;