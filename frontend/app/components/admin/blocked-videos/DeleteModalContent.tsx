import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { BlockedVideo, useUnblockVideo } from "@/app/hooks/useBlockedVideos";
import { Button, Code, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";

type Props = {
  blockedVideo: BlockedVideo
  handleClose: () => void;
}


const DeleteBlockedVideoModalContent = ({ blockedVideo, handleClose }: Props) => {
  const [loading, setLoading] = useState(false)

  const unblockVideoMutate = useUnblockVideo()
  const axiosPrivate = useAxiosPrivate()

  const handleUnblockVideo = async () => {
    if (!blockedVideo || !blockedVideo.id) return
    try {
      setLoading(true)

      await unblockVideoMutate.mutateAsync({ axiosPrivate: axiosPrivate, videoId: blockedVideo.id })

      showNotification({
        message: "Video unblocked"
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
      <Text>Are you sure you want to unblock this video?</Text>
      <Code block>{JSON.stringify(blockedVideo, null, 2)}</Code>
      <Button mt={5} color="red" onClick={handleUnblockVideo} loading={loading} fullWidth>Unblock Video</Button>
    </div>
  );
}

export default DeleteBlockedVideoModalContent;