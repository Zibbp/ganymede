'use client';
import { Container, SimpleGrid } from "@mantine/core";
import ChannelCard from "../components/channel/Card";
import { useFetchChannels } from "../hooks/useChannels";
import GanymedeLoadingText from "../components/utils/GanymedeLoadingText";
import { useEffect } from "react";
import { useTranslations } from "next-intl";

const ChannelsPage = () => {
  const t = useTranslations("ChannelsPage");

  useEffect(() => {
    document.title = t('title');
  }, []);

  const { data: channels, isPending, isError } = useFetchChannels()

  if (isPending) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isError) return <div>{t('error')}</div>

  return (
    <Container size="7xl" px="xl" mt={10}>
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