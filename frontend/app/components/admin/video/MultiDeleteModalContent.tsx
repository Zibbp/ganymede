import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useBlockVideo } from "@/app/hooks/useBlockedVideos";
import { useDeleteVideo, Video } from "@/app/hooks/useVideos";
import { Button, Text, Flex, Checkbox } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  videos: Video[]
  handleClose: () => void;
}

const MultiDeleteVideoModalContent = ({ videos, handleClose }: Props) => {
  const t = useTranslations('AdminVideoComponents')
  const [loading, setLoading] = useState(false)
  const [deleteFiles, setDeleteFiles] = useState(false)
  const [blockVideo, setBlockVideo] = useState(false)
  const deleteVideoMutate = useDeleteVideo()
  const blockVideoMutate = useBlockVideo()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteVideo = async () => {
    setLoading(true)
    try {
      await Promise.all(videos.map(async (video) => {
        if (blockVideo) {
          await blockVideoMutate.mutateAsync({
            axiosPrivate: axiosPrivate,
            id: video.ext_id
          })
        }
        await deleteVideoMutate.mutateAsync({
          axiosPrivate: axiosPrivate,
          id: video.id,
          deleteFiles: deleteFiles
        })
      }))

      showNotification({
        message: t('multiDeleteNotification')
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
      <Text>{t('multiDeleteConfirmText', { number: videos.length })}</Text>
      <Flex my={10}>
        <Checkbox
          label={t('deleteFilesLabel')}
          checked={deleteFiles}
          onChange={(event) => setDeleteFiles(event.currentTarget.checked)}
          mr={10}
        />
        <Checkbox
          label={t('blockExtIdLabel')}
          checked={blockVideo}
          onChange={(event) => setBlockVideo(event.currentTarget.checked)}
        />
      </Flex>
      <Button mt={5} color="red" onClick={handleDeleteVideo} loading={loading} fullWidth>{t('multiDeleteButton')}</Button>
    </div>
  );
}

export default MultiDeleteVideoModalContent;