import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Channel, useCreateChannel, useEditChannel, useUpdateChannelImage } from "@/app/hooks/useChannels";
import { ActionIcon, Button, NumberInput, TextInput, Tooltip, Text, Divider, Checkbox } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { IconHelpCircle } from "@tabler/icons-react";
import { z } from "zod";

type Props = {
  channel: Channel | null
  mode: ChannelEditMode
  handleClose: () => void;
}

export enum ChannelEditMode {
  Create = "create",
  Edit = "edit",
}

const schema = z.object({
  display_name: z.string().min(2, { message: "Display name should have at least 2 characters" }),
  name: z.string().min(2, { message: "Name should have at least 2 characters" }),
  image_path: z.string().min(3, { message: "Image path should have at least 3 characters" }),
  retention: z.boolean(),
  retention_days: z.number().min(1)
})

const AdminChannelDrawerContent = ({ channel, mode, handleClose }: Props) => {
  const axiosPrivate = useAxiosPrivate()
  const useCreateChannelMutate = useCreateChannel()
  const useEditChannelMutate = useEditChannel()
  const useUpdateChannelImageMutate = useUpdateChannelImage()

  const form = useForm({
    mode: "controlled",
    initialValues: {
      id: channel?.id || "",
      external_id: channel?.ext_id || "",
      name: channel?.name || "",
      display_name: channel?.display_name || "",
      image_path: channel?.image_path || "",
      retention: channel?.retention || false,
      retention_days: channel?.retention_days || 7,
    },

    validate: zodResolver(schema),
  })

  const handleSubmitForm = async () => {
    const formValues = form.getValues()
    console.debug(`Admin channel form submit - ${formValues}`)

    // @ts-expect-error partial channel
    const submitChannel: Channel = {
      id: formValues.id,
      ext_id: formValues.external_id,
      name: formValues.name,
      display_name: formValues.display_name,
      image_path: formValues.image_path,
      retention: formValues.retention,
      retention_days: formValues.retention_days,
    }

    // create channel
    if (mode == ChannelEditMode.Create) {
      try {
        await useCreateChannelMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          channel: submitChannel
        })

        showNotification({
          message: "Channel created"
        })

        handleClose()
      } catch (error) {
        console.error(error)
      }
    }

    // edit channel
    if (mode == ChannelEditMode.Edit) {
      try {
        if (!channel) return;
        await useEditChannelMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          channelId: channel.id,
          channel: submitChannel
        })

        showNotification({
          message: "Channel edited"
        })

        handleClose()
      } catch (error) {
        console.error(error)
      }
    }
  }

  const handleUpdateChannelImage = async () => {
    if (!channel || !channel.id) return

    try {
      await useUpdateChannelImageMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        channelId: channel.id
      })

      showNotification({
        message: "Channel image updated from platform"
      })
    } catch (error) {
      console.error(error)
    }
  }

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
        />
        <TextInput
          label="External Platform ID"
          placeholder="123456789"
          key={form.key('external_id')}
          {...form.getInputProps('external_id')}
        />
        <TextInput
          withAsterisk
          label="Name"
          placeholder="ganymede"
          key={form.key('name')}
          {...form.getInputProps('name')}
        />
        <TextInput
          withAsterisk
          label="Display Name"
          placeholder="Ganymede"
          key={form.key('display_name')}
          {...form.getInputProps('display_name')}
        />
        <TextInput
          withAsterisk
          label="Image Path"
          placeholder="/data/videos/ganymede/profile.jpg"
          key={form.key('image_path')}
          {...form.getInputProps('image_path')}
        />

        <div style={{ display: "flex" }}>
          <Text>Retention settings</Text>
          <Tooltip
            multiline
            label="If this setting is enabled channel videos will be deleted (including files) after the specified number of days. 'Lock' a video to prevent it from being deleted."
          >
            <ActionIcon variant="transparent">
              <IconHelpCircle size="1.125rem" />
            </ActionIcon>
          </Tooltip>
        </div>

        <Checkbox
          label="Enable Video Retention"
          key={form.key('retention')}
          {...form.getInputProps('retention', { type: "checkbox" })}
        />

        {form.values.retention && (
          <Text c="red" mt={5}>
            Videos will be deleted after {form.values.retention_days} days!
          </Text>
        )}

        <NumberInput
          disabled={!form.values.retention}
          label="Retention Days"
          placeholder="7"
          key={form.key('retention_days')}
          {...form.getInputProps('retention_days')}
        />

        <Button mt={10} type="submit" fullWidth>{mode == ChannelEditMode.Create ? 'Create Channel' : 'Edit Channel'}</Button>
      </form>
      {channel && (
        <div>
          <Divider mt={10} />
          <Button mt={10} fullWidth variant="default" onClick={handleUpdateChannelImage}>Update Image from Platform</Button>
        </div>
      )}
    </div>
  );
}

export default AdminChannelDrawerContent;