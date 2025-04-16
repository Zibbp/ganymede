import { useAddVideoToPlaylist, useGetPlaylists } from "@/app/hooks/usePlaylist";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { Button, Select } from "@mantine/core";
import { useEffect, useState } from "react";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { showNotification } from "@mantine/notifications";
import { useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { Video } from "@/app/hooks/useVideos";

interface Params {
  videos: Video[];
  handleClose: () => void;
}

interface FormattedPlaylist {
  label: string;
  value: string;
}

const PlaylistBulkAddModalContent = ({ videos, handleClose }: Params) => {
  const t = useTranslations('PlaylistComponents')
  const [playlistsFormatted, setPlaylistsFormatted] = useState<FormattedPlaylist[]>([]);
  const [selectedPlaylistValue, setSelectedPlaylistValue] = useState<string | null>(null);
  const [loading, setLoading] = useState<boolean>(false)
  const queryClient = useQueryClient()

  const axiosPrivate = useAxiosPrivate();

  const useAddVideoToPlaylistMutate = useAddVideoToPlaylist()

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
      setLoading(true)
      if (videos && videos.length > 0) {
        await Promise.all(
          videos.map((video) =>
            useAddVideoToPlaylistMutate.mutateAsync({ axiosPrivate, playlistId: selectedPlaylistValue, videoId: video.id })
          )
        )
      }

      queryClient.invalidateQueries({ queryKey: ["playlist_videos"] })

      showNotification({
        message: t('videosAddedToPlaylistNotification')
      })

      handleClose()
    } catch (error) {
      showNotification({
        title: t('notificationError'),
        message: error instanceof Error ? error.message : String(error),
      });
      console.error(error);
    } finally {
      setLoading(false)
    }
  }



  if (isPlaylistsPending) {
    return <GanymedeLoadingText message={t('loading')} />;
  }

  if (isPlaylistsError) {
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

      <Button mt={10} onClick={addVideoToPlayist} loading={loading} fullWidth>{t('addVideosToPlaylistButton')}</Button>

    </div>
  );
}

export default PlaylistBulkAddModalContent;