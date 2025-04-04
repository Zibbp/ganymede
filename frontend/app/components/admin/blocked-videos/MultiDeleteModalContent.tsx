import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { BlockedVideo, useUnblockVideo } from "@/app/hooks/useBlockedVideos";
import { Button, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  blockedVideos: BlockedVideo[]
  handleClose: () => void;
}

const MultiDeleteBlockedVideoModalContent = ({ blockedVideos, handleClose }: Props) => {
  const t = useTranslations('AdminBlockedVideosComponents')
  const [loading, setLoading] = useState(false)
  const unblockVideoMutate = useUnblockVideo()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteQueue = async () => {
    setLoading(true)
    try {
      await Promise.all(blockedVideos.map(async (blockedVideo) => {

        await unblockVideoMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          videoId: blockedVideo.id,
        })
      }))

      showNotification({
        message: t('multiUnblockedNotification')
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
      <Text>{t('multiUnblockText', { length: blockedVideos.length })}</Text>
      <Button mt={5} color="red" onClick={handleDeleteQueue} loading={loading} fullWidth>{t('multiUnblockButton')}</Button>
    </div>
  );
}

export default MultiDeleteBlockedVideoModalContent;