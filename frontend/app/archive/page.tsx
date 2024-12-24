"use client"
import { Card, Center, Container, Title, Text, TextInput, Divider, Select, Group, Switch, Button, rem } from "@mantine/core";
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

function extractTwitchId(input: string): string {
  // Regex patterns for various inputs
  const clipUrlRegex = /\/clip\/(?<id>[a-zA-Z0-9-_]+)/; // Extract full clip ID from URL
  const videoUrlRegex = /\/videos\/(?<id>\d+)/; // Extract video ID from URL
  const clipIdRegex = /^[a-zA-Z]+(?:[a-zA-Z0-9-]+)?$/; // Match standalone clip IDs
  const videoIdRegex = /^\d+$/; // Match standalone video IDs (numeric)

  // Check for a clip URL
  const clipUrlMatch = input.match(clipUrlRegex);
  if (clipUrlMatch?.groups?.id) {
    return clipUrlMatch.groups.id;
  }

  // Check for a video URL
  const videoUrlMatch = input.match(videoUrlRegex);
  if (videoUrlMatch?.groups?.id) {
    return videoUrlMatch.groups.id;
  }

  // Check for standalone clip IDs
  if (clipIdRegex.test(input)) {
    return input; // If input is already a valid clip ID, return it
  }

  // Check for standalone video IDs
  if (videoIdRegex.test(input)) {
    return input; // If input is already a valid video ID, return it
  }

  return ""
}


const ArchivePage = () => {
  useEffect(() => {
    document.title = "Archive";
  }, []);

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
    label: key.replace('quality', ''),
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
        video_id: extractTwitchId(archiveInput),
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
                  w={rem(200)}
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
          </div>
        </Center>
      </Container>
    </div>
  );
}

export default ArchivePage;