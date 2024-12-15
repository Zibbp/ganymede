import { Menu, rem, ActionIcon, Modal, Drawer } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  IconPhoto,
  IconTrash,
  IconMenu2,
  IconPlaylistAdd,
  IconInfoCircle,
  IconHourglassHigh,
  IconHourglassEmpty,
  IconLockOpen,
  IconShare,
  IconLock,
} from '@tabler/icons-react';
import VideoInfoModalContent from './modals/InfoModalContent';
import { useGenerateStaticThumbnail, useLockVideo, Video } from '@/app/hooks/useVideos';
import PlaylistManageDrawerContent from '../playlist/ManageDrawerContent';
import { useAxiosPrivate } from '@/app/hooks/useAxios';
import { useDeletePlayback, useMarkVideoAsWatched } from '@/app/hooks/usePlayback';
import { showNotification } from '@mantine/notifications';
import { useEffect, useRef } from 'react';
import videoEventBusInstance from '@/app/util/VideoEventBus';
import DeleteVideoModalContent from '../admin/video/DeleteModalContent';
import useAuthStore from '@/app/store/useAuthStore';
import { UserRole } from '@/app/hooks/useAuthentication';

type Props = {
  video: Video
}

const VideoMenu = ({ video }: Props) => {
  const [infoModalOpened, { open: infoModalOpen, close: infoModalClose }] = useDisclosure(false);
  const [playlistsDrawerOpened, { open: openPlaylistDrawer, close: closePlaylistDrawer }] = useDisclosure(false);
  const axiosPrivate = useAxiosPrivate()
  const isLocked = useRef(false);
  const { hasPermission } = useAuthStore()

  useEffect(() => {
    if (video.locked) {
      isLocked.current = true;
    }
  }, [video]);

  const markAsWatchedMutate = useMarkVideoAsWatched()
  const deletePlaybackMutate = useDeletePlayback()
  const lockVideoMutate = useLockVideo()
  const generateStaticThumbnailMutate = useGenerateStaticThumbnail()
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);

  const handleMarkAsWatched = async () => {
    try {
      await markAsWatchedMutate.mutateAsync({
        axiosPrivate,
        videoId: video.id
      })
      showNotification({
        message: "Video marked as watched"
      })
    } catch (error) {
      console.error(error)
    }
  }
  const handleMarkAsUnWatched = async () => {
    try {
      await deletePlaybackMutate.mutateAsync({
        axiosPrivate,
        videoId: video.id
      })
      showNotification({
        message: "Video marked as unwatched"
      })
    } catch (error) {
      console.error(error)
    }
  }
  const handleLockVideo = async (lock: boolean) => {
    try {
      await lockVideoMutate.mutateAsync({
        axiosPrivate,
        videoId: video.id,
        locked: lock
      })
      showNotification({
        message: `Video has been ${lock ? "locked" : "unlocked"}`
      })
      if (lock == true) {
        isLocked.current = true;
      } else {
        isLocked.current = false;
      }
    } catch (error) {
      console.error(error)
    }
  }
  const handleGenerateStaticThumbnail = async () => {
    try {
      await generateStaticThumbnailMutate.mutateAsync({
        axiosPrivate,
        videoId: video.id
      })
      showNotification({
        message: `Queued task to generate static thumbnail`
      })
    } catch (error) {
      console.error(error)
    }
  }

  const handleShareVideo = () => {
    let shareUrl: string = "";
    const url = window.location.origin;

    // check if we are on a vod page
    if (window.location.pathname.includes("/videos/") && window.location.pathname.includes(video.id)) {
      // get the current time
      const { time } = videoEventBusInstance.getData()
      const roundedTime = Math.ceil(time);
      // create the url
      shareUrl = `${url}/videos/${video.id}?t=${roundedTime}`;
    } else {
      // create the url
      shareUrl = `${url}/videos/${video.id}`;
    }

    // copy the url to the clipboard
    try {
      navigator.clipboard.writeText(shareUrl);
      showNotification({
        title: "Copied to Clipboard",
        message: "The video url has been copied to your clipboard",
      });

    } catch (err) {
      console.error(err);
      showNotification({
        title: "Error",
        message: "The clipboard API requires HTTPS, falling back to a prompt",
        color: "red",
      });
      prompt("Copy to clipboard: Ctrl+C, Enter", shareUrl);
    }
  }

  return (
    <div>
      <Menu shadow="md" width={200} position="top-end">
        {/* @ts-expect-error valid */}
        <Menu.Target>
          <ActionIcon color="gray" variant="subtle">
            <IconMenu2 size="1rem" />
          </ActionIcon>
        </Menu.Target>

        <Menu.Dropdown>
          <Menu.Item onClick={openPlaylistDrawer} leftSection={<IconPlaylistAdd style={{ width: rem(14), height: rem(14) }} />}>
            Playlists
          </Menu.Item>
          <Menu.Item leftSection={<IconInfoCircle style={{ width: rem(14), height: rem(14) }} />}
            onClick={infoModalOpen}>
            Info
          </Menu.Item>
          <Menu.Item leftSection={<IconHourglassHigh style={{ width: rem(14), height: rem(14) }} />} onClick={handleMarkAsWatched}>
            Mark as Watched
          </Menu.Item>
          <Menu.Item leftSection={<IconHourglassEmpty style={{ width: rem(14), height: rem(14) }} />} onClick={handleMarkAsUnWatched}>
            Mark as Unwatched
          </Menu.Item>
          {isLocked.current ? (
            <Menu.Item leftSection={<IconLockOpen style={{ width: rem(14), height: rem(14) }} />} onClick={() => handleLockVideo(false)}>
              Unlock
            </Menu.Item>
          ) : (
            <Menu.Item leftSection={<IconLock style={{ width: rem(14), height: rem(14) }} />} onClick={() => handleLockVideo(true)}>
              Lock
            </Menu.Item>
          )}
          <Menu.Item leftSection={<IconPhoto style={{ width: rem(14), height: rem(14) }} />} onClick={handleGenerateStaticThumbnail}>
            Regenerate Thumbnail
          </Menu.Item>
          <Menu.Item leftSection={<IconShare style={{ width: rem(14), height: rem(14) }} />} onClick={handleShareVideo}>
            Share
          </Menu.Item>

          {hasPermission(UserRole.Admin) && (
            <>
              <Menu.Divider />
              <Menu.Item
                color="red"
                leftSection={<IconTrash style={{ width: rem(14), height: rem(14) }} />}
                onClick={openDeleteModal}
              >
                Delete
              </Menu.Item>
            </>
          )}

        </Menu.Dropdown>
      </Menu>

      <Modal
        opened={infoModalOpened}
        onClose={infoModalClose}
        size="xl"
        title="Video Information"
      >
        <VideoInfoModalContent video={video} />
      </Modal>

      <Drawer opened={playlistsDrawerOpened} onClose={closePlaylistDrawer} position="right" title="Manage Playlists">
        <PlaylistManageDrawerContent videoId={video.id} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title="Delete Video">
        <DeleteVideoModalContent video={video} handleClose={closeDeleteModal} />
      </Modal>
    </div>
  );
}

export default VideoMenu;