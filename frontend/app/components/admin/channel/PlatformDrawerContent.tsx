import { useArchiveChannel } from "@/app/hooks/useArchive";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Button, TextInput } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";
import { z } from "zod";

type Props = {
  handleClose: () => void;
}

const PlatformChannelDrawerContent = ({ handleClose }: Props) => {
  const t = useTranslations('AdminChannelsComponents')
  const useArchiveChannelMutate = useArchiveChannel()
  const axiosPrivate = useAxiosPrivate()
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    channel_name: z.string().min(2, { message: t('validation.channelName') })
  })

  const form = useForm({
    mode: "controlled",
    initialValues: {
      channel_name: ""
    },

    validate: zodResolver(schema),
  })

  const handleSubmitForm = async () => {
    const formValues = form.getValues()

    // archive platform channel
    try {
      setLoading(true)
      await useArchiveChannelMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        channel_name: formValues.channel_name
      })

      showNotification({
        message: t('addedNotification')
      })

      handleClose()
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <form onSubmit={form.onSubmit(() => {
        handleSubmitForm()
      })}>
        <TextInput
          label={t('platformChannelNameLabel')}
          placeholder="ganymede"
          key={form.key('channel_name')}
          {...form.getInputProps('channel_name')}
        />

        <Button mt={10} type="submit" loading={loading} fullWidth>{t('addButton')}</Button>
      </form>
    </div>
  );
}

export default PlatformChannelDrawerContent;