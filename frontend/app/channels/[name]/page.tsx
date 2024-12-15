"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import ChannelVideos from "@/app/components/videos/ChannelVideos";
import { useFetchChannelByName } from "@/app/hooks/useChannels";
import { Center, Container, Title } from "@mantine/core";
import React, { useEffect } from "react";

interface Params {
  name: string;
}

const ChannelPage = ({ params }: { params: Promise<Params> }) => {
  const { name } = React.use(params);
  useEffect(() => {
    document.title = `${name}`;
  }, [name]);

  const { data: channel, isPending, isError } = useFetchChannelByName(name)

  if (isPending) {
    return (
      <GanymedeLoadingText message="Loading Channel" />
    );
  }

  if (isError) {
    return (
      <Center>
        <div>Error loading channel</div>
      </Center>
    );
  }
  return (
    <div>
      <Container size="xl" px="xl" fluid={true}>
        <Center>
          <Title>{channel.display_name}</Title>
        </Center>
        <ChannelVideos channel={channel} />
      </Container>
    </div>
  );
}
export default ChannelPage;