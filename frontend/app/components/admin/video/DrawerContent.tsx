import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Channel, useFetchChannels } from "@/app/hooks/useChannels";
import { CreateVodRequest, Platform, useCreateVideo, useEditVideo, Video, VideoType } from "@/app/hooks/useVideos";
import { Select, Button, NumberInput, TextInput, Checkbox, Flex } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useEffect, useState } from "react";
import { z } from "zod";

type Props = {
  video: Video | null
  mode: VideoEditMode
  handleClose: () => void;
}

export enum VideoEditMode {
  Create = "create",
  Edit = "edit",
}


interface SelectOption {
  label: string;
  value: string;
}

const schema = z.object({
  title: z.string().min(1, { message: "Title should have at least 1 characters" }),
  ext_id: z.string().min(1, { message: "External ID should have at least 1 characters" }),
  channel_id: z.string().min(1, { message: "Select a channel" }),
  type: z.nativeEnum(VideoType, { message: "Select a video type" }),
  platform: z.nativeEnum(Platform, { message: "Select a platform" }),
  duration: z.number().min(1, { message: "Duration should be at least 1 second" }),
  views: z.number().min(1),
  resolution: z.string().min(1),
  streamed_at: z.string().datetime(),
  web_thumbnail_path: z.string().min(1),
  video_path: z.string().min(1)
})

const AdminVideoDrawerContent = ({ video, mode, handleClose }: Props) => {
  const axiosPrivate = useAxiosPrivate()
  const [loading, setLoading] = useState(false)

  const form = useForm({
    mode: "controlled",
    initialValues: {
      id: video?.id || "",
      ext_id: video?.ext_id || "",
      platform: video?.platform || Platform.Twitch,
      type: video?.type || "",
      title: video?.title || "",
      duration: video?.duration || 0,
      views: video?.views || 0,
      resolution: video?.resolution || "",
      thumbnail_path: video?.thumbnail_path || "",
      web_thumbnail_path: video?.web_thumbnail_path || "",
      video_path: video?.video_path || "",
      chat_path: video?.chat_path || "",
      chat_video_path: video?.chat_video_path || "",
      info_path: video?.info_path || "",
      processing: video?.processing ?? false,
      streamed_at: video?.streamed_at || "",
      channel_id: video?.edges.channel.id || "",
      caption_path: video?.caption_path || "",
      locked: video?.locked ?? false
    },

    validate: zodResolver(schema),
  })

  const editVideoMutate = useEditVideo()
  const createVideoMutate = useCreateVideo()

  const handleSubmitForm = async () => {
    setLoading(true)
    const formValues = form.getValues()
    console.debug(`Admin video form submit - ${formValues}`)

    try {

      const req: CreateVodRequest = {
        ...formValues,
        type: formValues.type as VideoType,
        streamed_at: formValues.streamed_at instanceof Date
          ? formValues.streamed_at.toISOString()
          : formValues.streamed_at,
      };

      if (mode === VideoEditMode.Create) {
        await createVideoMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          videoData: req
        })
      } else {
        await editVideoMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          videoData: req,
          videoId: req.id
        })
      }

      showNotification({
        message: `Video ${mode == VideoEditMode.Create ? 'created' : 'edited'}`
      })

      handleClose()

    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }

  }

  const { data: channels } = useFetchChannels();
  const [channelSelect, setChannelSelect] = useState<SelectOption[]>([]);

  useEffect(() => {
    if (!channels) return;

    const transformedChannels: SelectOption[] = channels.map((channel: Channel) => ({
      label: channel.name,
      value: channel.id,
    }));

    setChannelSelect(transformedChannels);
  }, [channels]);

  const selectorVideoTypes = Object.values(VideoType).map((type) => ({
    value: type,
    label: type.charAt(0).toUpperCase() + type.slice(1),
  }));

  const selectorVideoPlatforms = Object.values(Platform).map((type) => ({
    value: type,
    label: type.charAt(0).toUpperCase() + type.slice(1),
  }));

  return (
    <div>
      <form onSubmit={form.onSubmit(() => {
        handleSubmitForm()
      })}>
        <TextInput
          disabled={true}
          label="ID"
          placeholder="Auto generated"
          key={form.key('id')}
          {...form.getInputProps('id')}
          withAsterisk
        />
        <TextInput
          label="External Platform ID"
          placeholder="123456789"
          key={form.key('ext_id')}
          {...form.getInputProps('ext_id')}
          withAsterisk
        />

        <Select
          label="Channel"
          data={channelSelect}
          key={form.key('channel_id')}
          {...form.getInputProps('channel_id')}
          searchable
          withAsterisk
        />

        <Flex
          my={10}
          gap="md"
          justify="flex-start"
          align="center"
          direction="row"
        >

          <Checkbox
            label="Is Processing"
            key={form.key('processing')}
            {...form.getInputProps('processing', { type: "checkbox" })}
          />
          <Checkbox
            label="Locked"
            key={form.key('locked')}
            {...form.getInputProps('locked', { type: "checkbox" })}
          />

        </Flex>


        <TextInput
          withAsterisk
          label="Title"
          placeholder="An awesome title"
          key={form.key('title')}
          {...form.getInputProps('title')}
        />

        <Select
          label="Type"
          data={selectorVideoTypes}
          key={form.key('type')}
          {...form.getInputProps('type')}
          searchable
          withAsterisk
        />

        <Select
          label="Platform"
          data={selectorVideoPlatforms}
          key={form.key('platform')}
          {...form.getInputProps('platform')}
          searchable
          withAsterisk
        />

        <Flex
          gap="md"
          justify="flex-start"
          align="center"
          direction="row"
        >
          <NumberInput
            label="Duration (seconds)"
            placeholder="0"
            key={form.key('duration')}
            {...form.getInputProps('duration')}
            min={0}
            withAsterisk
          />
          <NumberInput
            label="View Count"
            placeholder="0"
            key={form.key('views')}
            {...form.getInputProps('views')}
            min={0}
            withAsterisk
          />
        </Flex>

        <TextInput
          withAsterisk
          label="Resolution"
          placeholder="best"
          key={form.key('resolution')}
          {...form.getInputProps('resolution')}
        />

        <TextInput
          withAsterisk
          label="Streamed At"
          placeholder="2022-01-01"
          key={form.key('streamed_at')}
          {...form.getInputProps('streamed_at')}
        />

        <TextInput
          label="Thumbnail Path"
          placeholder="/data/videos/channel/123_456/123-thumbnail.jpg"
          key={form.key('thumbnail_path')}
          {...form.getInputProps('thumbnail_path')}
        />
        <TextInput
          label="Web Thumbnail Path"
          placeholder="/data/videos/channel/123_456/123-web_thumbnail.jpg"
          key={form.key('web_thumbnail_path')}
          {...form.getInputProps('web_thumbnail_path')}
          withAsterisk
        />
        <TextInput
          label="Video Path"
          placeholder="/data/videos/channel/123_456/123-video.mp4"
          key={form.key('video_path')}
          {...form.getInputProps('video_path')}
          withAsterisk
        />
        <TextInput
          label="Chat Path"
          placeholder="/data/videos/channel/123_456/123-chat.json"
          key={form.key('chat_path')}
          {...form.getInputProps('chat_path')}
        />
        <TextInput
          label="Chat Video Path"
          placeholder="/data/videos/channel/123_456/123-chat.mp4"
          key={form.key('chat_video_path')}
          {...form.getInputProps('chat_video_path')}
        />
        <TextInput
          label="Caption Path"
          placeholder="/data/videos/channel/123_456/123.vtt"
          key={form.key('caption_path')}
          {...form.getInputProps('caption_path')}
        />
        <TextInput
          label="Info Path"
          placeholder="/data/videos/channel/123_456/123-info.json"
          key={form.key('info_path')}
          {...form.getInputProps('info_path')}
        />



        <Button mt={10} type="submit" loading={loading} fullWidth>{mode == VideoEditMode.Create ? 'Create Video' : 'Edit Video'}</Button>
      </form>
    </div>
  );
}

export default AdminVideoDrawerContent;