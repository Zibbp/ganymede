import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { BlockedVideo, useUnblockVideo } from "@/app/hooks/useBlockedVideos";
import { Button, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";

type Props = {
  blockedVideos: BlockedVideo[]
  handleClose: () => void;
}

const MultiDeleteBlockedVideoModalContent = ({ blockedVideos, handleClose }: Props) => {
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
        message: "Unblocked videos deleted"
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
      <Text>Are you sure you want to unblock the {blockedVideos.length} selected blocked videos?</Text>
      <Button mt={5} color="red" onClick={handleDeleteQueue} loading={loading} fullWidth>Unblock Videos</Button>
    </div>
  );
}

export default MultiDeleteBlockedVideoModalContent;