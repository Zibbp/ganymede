import { useAddVideoToPlaylist, useGetPlaylists, useRemoveVideoFromPlaylist } from "@/app/hooks/usePlaylist";
import { useGetPlaylistsForVideo } from "@/app/hooks/useVideos";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { ActionIcon, Button, Divider, Flex, Text, Select } from "@mantine/core";
import { useEffect, useState } from "react";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { showNotification } from "@mantine/notifications";
import { useQueryClient } from "@tanstack/react-query";
import { IconTrash } from "@tabler/icons-react";
import { useTranslations } from "next-intl";
interface Params {
  videoId: string;
}

interface FormattedPlaylist {
  label: string;
  value: string;
}

const PlaylistManageDrawerContent = ({ videoId }: Params) => {
  const t = useTranslations('PlaylistComponents')
  const [playlistsFormatted, setPlaylistsFormatted] = useState<FormattedPlaylist[]>([]);
  const [selectedPlaylistValue, setSelectedPlaylistValue] = useState<string | null>(null);
  const queryClient = useQueryClient()

  const axiosPrivate = useAxiosPrivate();

  const useAddVideoToPlaylistMutate = useAddVideoToPlaylist()
  const useRemoveVideoFromPlaylistMutate = useRemoveVideoFromPlaylist()

  const { data: videoPlaylists, isPending: isVideoPlaylistsPending, isError: isVideoPlaylistsError } = useGetPlaylistsForVideo(videoId)

  const { data: playlists, isPending: isPlaylistsPending, isError: isPlaylistsError } = useGetPlaylists()

  useEffect(() => {
    setPlaylistsFormatted([])
    if (!playlists) return
    playlists.forEach((playlist) => {
      setPlaylistsFormatted(prevPlaylists => {
        const currentPlaylists = prevPlaylists || [];
        return [...currentPlaylists, {
          label: playlist.name,
          value: playlist.id
        }];
      });
    })
  }, [playlists])


  const addVideoToPlayist = async () => {
    if (!selectedPlaylistValue) return;
    try {
      await useAddVideoToPlaylistMutate.mutateAsync({ axiosPrivate, playlistId: selectedPlaylistValue, videoId: videoId })

      queryClient.invalidateQueries({ queryKey: ["video", "playlists", videoId] })

      showNotification({
        message: t('videoAddedToPlaylistNotification')
      })


    } catch (error) {
      console.error(error)
    }
  }

  const removeVideoFromPlaylist = async (playlistId: string) => {
    try {
      await useRemoveVideoFromPlaylistMutate.mutateAsync({ axiosPrivate, playlistId: playlistId, videoId: videoId })

      queryClient.invalidateQueries({ queryKey: ["video", "playlists", videoId] })

      showNotification({
        message: t('videoRemovedFromPlaylistNotification')
      })
    } catch (error) {
      console.error(error)
    }
  }


  if (isVideoPlaylistsPending || isPlaylistsPending) {
    return <GanymedeLoadingText message={t('loading')} />;
  }

  if (isVideoPlaylistsError || isPlaylistsError) {
    return <div>{t('errorLoading')}</div>;
  }

  return (
    <div>
      <Select
        data={playlistsFormatted}
        value={selectedPlaylistValue}
        onChange={setSelectedPlaylistValue}
        searchable
        placeholder={t('addVideoPlaylistsPlaceholder')}
        w="100%"
      />

      <Button mt={10} onClick={addVideoToPlayist} fullWidth>{t('addVideoToPlaylistButton')}</Button>

      <Divider my="md" />

      {videoPlaylists.map((playlist) => (
        <div key={playlist.id}>

          <Flex>
            <ActionIcon variant="light" color="red" aria-label="Settings" mr={5} my={5} onClick={() => removeVideoFromPlaylist(playlist.id)}>
              <IconTrash style={{ width: '70%', height: '70%' }} stroke={1.5} />
            </ActionIcon>
            <Text>{playlist.name}</Text>
          </Flex>

        </div>
      ))}

    </div>
  );
}

export default PlaylistManageDrawerContent;