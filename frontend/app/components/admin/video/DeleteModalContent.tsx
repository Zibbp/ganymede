import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useBlockVideo } from "@/app/hooks/useBlockedVideos";
import { useDeleteVideo, Video } from "@/app/hooks/useVideos";
import { escapeURL } from "@/app/util/util";
import { Button, Code, Text, Image, Flex, Checkbox } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { env } from "next-runtime-env";
import { useState } from "react";

type Props = {
  video: Video
  handleClose: () => void;
}

const DeleteVideoModalContent = ({ video, handleClose }: Props) => {
  const [loading, setLoading] = useState(false)
  const [deleteFiles, setDeleteFiles] = useState(false)
  const [blockVideo, setBlockVideo] = useState(false)

  const deleteVideoMutate = useDeleteVideo()
  const blockVideoMutate = useBlockVideo()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteVideo = async () => {
    try {
      setLoading(true)

      if (blockVideo) {
        await blockVideoMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          id: video.ext_id
        })
      }

      await deleteVideoMutate.mutateAsync({ axiosPrivate: axiosPrivate, id: video.id, deleteFiles: deleteFiles })

      showNotification({
        message: "Video deleted"
      })

      handleClose()
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
      setBlockVideo(false)
      setDeleteFiles(false)
    }
  }

  return (
    <div>
      <Text>Are you sure you want to delete this video?</Text>
      <Code block mb={5}>
        <pre>ID: {video.id}</pre>
        <pre>External ID: {video.ext_id}</pre>
        <pre>Title: {video.title}</pre>
        <pre>Channel: {video.edges.channel.name}</pre>
      </Code>
      <Image
        src={`${(env('NEXT_PUBLIC_CDN_URL') ?? '')}${escapeURL(
          video.web_thumbnail_path
        )}`}
        fallbackSrc="/images/ganymede-thumbnail.webp"
        alt={video.title}
      />

      <Flex my={10}>
        <Checkbox
          label="Delete files"
          checked={deleteFiles}
          onChange={(event) => setDeleteFiles(event.currentTarget.checked)}
          mr={10}
        />
        <Checkbox
          label="Block external video ID"
          checked={blockVideo}
          onChange={(event) => setBlockVideo(event.currentTarget.checked)}
        />

      </Flex>

      <Button mt={5} color="red" onClick={handleDeleteVideo} loading={loading} fullWidth>Delete Video</Button>
    </div>
  );
}

export default DeleteVideoModalContent;