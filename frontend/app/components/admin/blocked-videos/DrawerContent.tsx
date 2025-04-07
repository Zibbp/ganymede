import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { BlockedVideo, useBlockVideo } from "@/app/hooks/useBlockedVideos";
import { Button, TextInput } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { z } from "zod";

type Props = {
  blockedVideo: BlockedVideo | null
  handleClose: () => void;
}

const schema = z.object({
  id: z.string().min(1, { message: "ID should have at least 2 characters" }),
})

const AdminBlockedVideosDrawerContent = ({ blockedVideo, handleClose }: Props) => {
  const t = useTranslations('AdminBlockedVideosComponents')
  const axiosPrivate = useAxiosPrivate()
  const useBlockVideoMutate = useBlockVideo()

  const form = useForm({
    mode: "controlled",
    initialValues: {
      id: blockedVideo?.id || "",
    },

    validate: zodResolver(schema),
  })

  const handleSubmitForm = async () => {
    const formValues = form.getValues()

    // create channel
    try {
      await useBlockVideoMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        id: formValues.id
      })

      showNotification({
        message: t('blockedNotification')
      })

      handleClose()
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
          label="ID"
          placeholder={t('idPlaceholder')}
          key={form.key('id')}
          {...form.getInputProps('id')}
        />
        <Button mt={10} type="submit" fullWidth>{t('blockButton')}</Button>
      </form>
    </div>
  );
}

export default AdminBlockedVideosDrawerContent;