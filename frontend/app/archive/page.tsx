"use client"
import { Card, Center, Container, Title, Text, TextInput, Divider, Select, Group, Switch, Button } from "@mantine/core";
import { useInputState } from "@mantine/hooks";
import { useEffect, useState } from "react";
import { Channel, useFetchChannels } from "../hooks/useChannels";
import { showNotification } from "@mantine/notifications";
import { useArchiveVideo, VideoQuality } from "../hooks/useArchive";
import { useAxiosPrivate } from "../hooks/useAxios";

interface SelectOption {
  label: string;
  value: string;
}

const ArchivePage = () => {
  // State management with proper typing
  const [archiveInput, setArchiveInput] = useInputState("");
  const [archiveSubmitLoading, setArchiveSubmitLoading] = useState(false);
  const [archiveChat, setArchiveChat] = useInputState(true);
  const [renderChat, setRenderChat] = useInputState(true);
  const [archiveQuality, setArchiveQuality] = useInputState<VideoQuality>(VideoQuality.Best);
  const [channelData, setChannelData] = useState<SelectOption[]>([]);
  const [channelId, setChannelId] = useState("");

  const axiosPrivate = useAxiosPrivate();
  const useArchiveVideoMutate = useArchiveVideo();
  const { data: channels, isPending: channelsIsPending } = useFetchChannels();

  // Quality options using the enum
  const qualityOptions: SelectOption[] = Object.entries(VideoQuality).map(([key, value]) => ({
    label: key.replace('Quality', ''),
    value: value
  }));

  // Effect to transform channel data
  useEffect(() => {
    if (!channels) return;

    const transformedChannels: SelectOption[] = channels.map((channel: Channel) => ({
      label: channel.name,
      value: channel.id,
    }));

    setChannelData(transformedChannels);
  }, [channels]);

  const archiveVideo = async () => {
    try {
      // Input validation
      if (!archiveInput && !channelId) {
        showNotification({
          title: "Input Required",
          message: "Please enter a video ID or select a channel",
          color: "red",
        });
        return;
      }

      if (archiveInput && channelId) {
        showNotification({
          title: "Invalid Selection",
          message: "Please either enter an ID or select a channel (not both)",
          color: "red",
        });
        return;
      }

      setArchiveSubmitLoading(true);

      await useArchiveVideoMutate.mutateAsync({
        axiosPrivate,
        video_id: archiveInput,
        channel_id: channelId,
        quality: archiveQuality,
        archive_chat: archiveChat,
        render_chat: renderChat,
      });

      setArchiveInput("")

      showNotification({
        title: "Success",
        message: "Video added to archive queue",
        color: "green",
      });

    } catch (error) {
      console.error(error)
    } finally {
      setArchiveSubmitLoading(false);
    }
  };

  return (
    <div>
      <Container size="md" mt={20}>
        <Center>
          <div style={{ width: "100%" }}>
            <Card
              shadow="sm"
              p="lg"
              radius="md"
              withBorder
            >
              <Center>
                <div>
                  <Title>Archive</Title>
                </div>
              </Center>
              <Center mb={10}>
                <Text>
                  Enter a video ID or select a channel to archive a livestream
                </Text>
              </Center>
              <TextInput
                value={archiveInput}
                onChange={setArchiveInput}
                placeholder="Video ID or URL"
                disabled={channelsIsPending}
                className="mb-4"
              />
              <Divider my="xs" label="Or" labelPosition="center" />
              <Select
                placeholder="Select Channel"
                data={channelData}
                value={channelId}
                onChange={(value) => setChannelId(value || "")}
                searchable
                mb={"md"}
                disabled={channelsIsPending}
              />
              <Group mt={5} mb={10}>
                <Select
                  placeholder="Resolution"
                  value={archiveQuality}
                  onChange={(value) => setArchiveQuality(value as VideoQuality)}
                  data={qualityOptions}
                  className="w-1/3"
                />
                <Switch
                  checked={archiveChat}
                  onChange={setArchiveChat}
                  label="Archive Chat"
                  color="violet"
                />
                <Switch
                  checked={renderChat}
                  onChange={setRenderChat}
                  label="Render Chat"
                  color="violet"
                />
              </Group>
              <Button
                onClick={archiveVideo}
                fullWidth
                radius="md"
                size="md"
                color="violet"
                loading={archiveSubmitLoading}
                disabled={channelsIsPending || (!archiveInput && !channelId)}
              >
                Archive
              </Button>
            </Card>
            {/* {twitchVodInfo?.id && <VodPreview video={twitchVodInfo} />} */}
          </div>
        </Center>
      </Container>
    </div>
  );
}

export default ArchivePage;