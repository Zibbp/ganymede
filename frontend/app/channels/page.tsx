'use client';
import { Container, SimpleGrid } from "@mantine/core";
import ChannelCard from "../components/channel/Card";
import { useFetchChannels } from "../hooks/useChannels";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";

const ChannelsPage = () => {

  const { data: channels, isPending, isError } = useFetchChannels()

  if (isPending) return (
    <GanymedeLoadingText message="Loading Channels" />
  )
  if (isError) return <div>Error loading channels</div>

  return (
    <Container size="7xl" px="xl" >
      <SimpleGrid
        cols={{ base: 1, sm: 3, lg: 6, xl: 8 }}
        verticalSpacing={{ base: 'md', sm: 'xl' }}
      >
        {channels.map((channel) => (
          <ChannelCard key={channel.id} channel={channel} />
        ))}
      </SimpleGrid>
    </Container>
  );
}

export default ChannelsPage;