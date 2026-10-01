import { useAddVideoToPlaylist, useGetPlaylists, useRemoveVideoFromPlaylist } from "@/app/hooks/usePlaylist";
import { useGetPlaylistsForVideo } from "@/app/hooks/useVideos";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { Checkbox, ScrollArea, Stack, Text, TextInput } from "@mantine/core";
import { useMemo, useState } from "react";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { showNotification } from "@mantine/notifications";
import { useQueryClient } from "@tanstack/react-query";
import { IconSearch } from "@tabler/icons-react";
import { useTranslations } from "next-intl";

interface Params {
  videoId: string;
}

const PlaylistManageDrawerContent = ({ videoId }: Params) => {
  const t = useTranslations('PlaylistComponents')
  const [filter, setFilter] = useState<string>("");
  // Optimistic membership state for playlists with an in-flight request.
  // Key is the playlist ID, value is the membership the user just requested.
  const [pendingMembership, setPendingMembership] = useState<Record<string, boolean>>({});
  const queryClient = useQueryClient()

  const axiosPrivate = useAxiosPrivate();

  const useAddVideoToPlaylistMutate = useAddVideoToPlaylist()
  const useRemoveVideoFromPlaylistMutate = useRemoveVideoFromPlaylist()

  const { data: videoPlaylists, isPending: isVideoPlaylistsPending, isError: isVideoPlaylistsError } = useGetPlaylistsForVideo(videoId)

  const { data: playlists, isPending: isPlaylistsPending, isError: isPlaylistsError } = useGetPlaylists()

  const memberPlaylistIds = useMemo(
    () => new Set((videoPlaylists ?? []).map((playlist) => playlist.id)),
    [videoPlaylists]
  )

  const filteredPlaylists = useMemo(() => {
    if (!playlists) return []
    const query = filter.trim().toLowerCase()
    if (query === "") return playlists
    return playlists.filter((playlist) => playlist.name.toLowerCase().includes(query))
  }, [playlists, filter])

  const clearPendingMembership = (playlistId: string) => {
    setPendingMembership((previous) => {
      const next = { ...previous };
      delete next[playlistId];
      return next;
    })
  }

  const togglePlaylistMembership = async (playlistId: string, isMember: boolean) => {
    // Ignore clicks while a request for this playlist is still running
    if (playlistId in pendingMembership) return;

    setPendingMembership((previous) => ({ ...previous, [playlistId]: !isMember }))

    try {
      if (isMember) {
        await useRemoveVideoFromPlaylistMutate.mutateAsync({ axiosPrivate, playlistId, videoId })
      } else {
        await useAddVideoToPlaylistMutate.mutateAsync({ axiosPrivate, playlistId, videoId })
      }

      // Wait for the refetch so the optimistic state is only dropped once the real data is in
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["video", "playlists", videoId] }),
        queryClient.invalidateQueries({ queryKey: ["playlist_videos"] }),
      ])

      showNotification({
        message: isMember ? t('videoRemovedFromPlaylistNotification') : t('videoAddedToPlaylistNotification')
      })
    } catch (error) {
      showNotification({
        title: t('notificationError'),
        message: error instanceof Error ? error.message : String(error),
        color: "red",
      })
      console.error(error)
    } finally {
      clearPendingMembership(playlistId)
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
      <TextInput
        value={filter}
        onChange={(event) => setFilter(event.currentTarget.value)}
        placeholder={t('searchPlaylistsPlaceholder')}
        leftSection={<IconSearch size={16} stroke={1.5} />}
        mb="md"
        w="100%"
      />

      {filteredPlaylists.length === 0 ? (
        <Text c="dimmed" ta="center" py="md">{t('noPlaylistsFound')}</Text>
      ) : (
        <ScrollArea.Autosize mah="calc(100vh - 160px)" type="auto" mx={-6}>
          <Stack gap={0} px={6}>
            {filteredPlaylists.map((playlist) => {
              const isMember = pendingMembership[playlist.id] ?? memberPlaylistIds.has(playlist.id)
              return (
                <Checkbox
                  key={playlist.id}
                  label={playlist.name}
                  checked={isMember}
                  onChange={() => togglePlaylistMembership(playlist.id, isMember)}
                  py={8}
                  styles={{
                    input: { cursor: 'pointer' },
                    label: { cursor: 'pointer' },
                  }}
                />
              )
            })}
          </Stack>
        </ScrollArea.Autosize>
      )}
    </div>
  );
}

export default PlaylistManageDrawerContent;
