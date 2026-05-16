import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Channel, useCreateChannel, useEditChannel, useUpdateChannelImage } from "@/app/hooks/useChannels";
import { ActionIcon, Button, NumberInput, TextInput, Tooltip, Text, Divider, Checkbox } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { IconHelpCircle } from "@tabler/icons-react";
import { useTranslations } from "next-intl";
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

const AdminChannelDrawerContent = ({ channel, mode, handleClose }: Props) => {
  const t = useTranslations('AdminChannelsComponents')
  const axiosPrivate = useAxiosPrivate()
  const useCreateChannelMutate = useCreateChannel()
  const useEditChannelMutate = useEditChannel()
  const useUpdateChannelImageMutate = useUpdateChannelImage()

  const schema = z.object({
    display_name: z.string().min(2, { message: t('validation.displayName') }),
    name: z.string().min(2, { message: t('validation.name') }),
    image_path: z.string().min(3, { message: t('validation.imagePath') }),
    retention: z.boolean(),
    retention_days: z.number().min(1)
  })

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
          message: t('createNotification')
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
          message: t('editNotification')
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
        message: t('imageUpdateNotification')
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
          placeholder={t('idLabel')}
          key={form.key('id')}
          {...form.getInputProps('id')}
        />
        <TextInput
          label={t('extIdLabel')}
          placeholder="123456789"
          key={form.key('external_id')}
          {...form.getInputProps('external_id')}
        />
        <TextInput
          withAsterisk
          label={t('nameLabel')}
          placeholder="ganymede"
          key={form.key('name')}
          {...form.getInputProps('name')}
        />
        <TextInput
          withAsterisk
          label={t('displayNameLabel')}
          placeholder="Ganymede"
          key={form.key('display_name')}
          {...form.getInputProps('display_name')}
        />
        <TextInput
          withAsterisk
          label={t('imagePathLabel')}
          placeholder="/data/videos/ganymede/profile.jpg"
          key={form.key('image_path')}
          {...form.getInputProps('image_path')}
        />

        <div style={{ display: "flex" }}>
          <Text>{t('videoRetentionSettingsHeader')}</Text>
          <Tooltip
            multiline
            label={t('videoRetentionLabel')}
          >
            <ActionIcon variant="transparent">
              <IconHelpCircle size="1.125rem" />
            </ActionIcon>
          </Tooltip>
        </div>

        <Checkbox
          label={t('videoRetentionLabel')}
          key={form.key('retention')}
          {...form.getInputProps('retention', { type: "checkbox" })}
        />

        {form.values.retention && (
          <Text c="red" mt={5}>
            {t('videoRetentionWarning', { number: form.values.retention_days })}
          </Text>
        )}

        <NumberInput
          disabled={!form.values.retention}
          label={t('videoRetentionDaysLabel')}
          placeholder="7"
          key={form.key('retention_days')}
          {...form.getInputProps('retention_days')}
        />

        <Button mt={10} type="submit" fullWidth>{mode == ChannelEditMode.Create ? t('submitButton') : t('editButton')}</Button>
      </form>
      {channel && (
        <div>
          <Divider mt={10} />
          <Button mt={10} fullWidth variant="default" onClick={handleUpdateChannelImage}>{t('imageUpdateButton')}</Button>
        </div>
      )}
    </div>
  );
}

export default AdminChannelDrawerContent;