import { useArchiveChannel } from "@/app/hooks/useArchive";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Button, TextInput } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";
import { z } from "zod";

type Props = {
  handleClose: () => void;
}

const schema = z.object({
  channel_name: z.string().min(2, { message: "Channel name should have at least 2 characters" })
})

const PlatformChannelDrawerContent = ({ handleClose }: Props) => {
  const useArchiveChannelMutate = useArchiveChannel()
  const axiosPrivate = useAxiosPrivate()
  const [loading, setLoading] = useState(false)

  const form = useForm({
    mode: "controlled",
    initialValues: {
      channel_name: ""
    },

    validate: zodResolver(schema),
  })

  const handleSubmitForm = async () => {
    const formValues = form.getValues()
    console.debug(`Admin platform channel form submit - ${formValues}`)

    // archive platform channel
    try {
      setLoading(true)
      await useArchiveChannelMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        channel_name: formValues.channel_name
      })

      showNotification({
        message: "Channel added"
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
          label="Platform Channel Name"
          placeholder="ganymede"
          key={form.key('channel_name')}
          {...form.getInputProps('channel_name')}
        />

        <Button mt={10} type="submit" loading={loading} fullWidth>Add Channel</Button>
      </form>
    </div>
  );
}

export default PlatformChannelDrawerContent;