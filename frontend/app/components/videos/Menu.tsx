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
  IconMovie,
} from '@tabler/icons-react';
import VideoInfoModalContent from './modals/InfoModalContent';
import { useGenerateSpriteThumbnails, useGenerateStaticThumbnail, useLockVideo, Video } from '@/app/hooks/useVideos';
import PlaylistManageDrawerContent from '../playlist/ManageDrawerContent';
import { useAxiosPrivate } from '@/app/hooks/useAxios';
import { useDeletePlayback, useMarkVideoAsWatched } from '@/app/hooks/usePlayback';
import { showNotification } from '@mantine/notifications';
import { useEffect, useRef } from 'react';
import videoEventBusInstance from '@/app/util/VideoEventBus';
import DeleteVideoModalContent from '../admin/video/DeleteModalContent';
import useAuthStore from '@/app/store/useAuthStore';
import { UserRole } from '@/app/hooks/useAuthentication';
import { useTranslations } from 'next-intl';

type Props = {
  video: Video
}

const VideoMenu = ({ video }: Props) => {
  const t = useTranslations('VideoComponents')
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
  const generateSpriteThumbnailsMutate = useGenerateSpriteThumbnails()
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);

  const handleMarkAsWatched = async () => {
    try {
      await markAsWatchedMutate.mutateAsync({
        axiosPrivate,
        videoId: video.id
      })
      showNotification({
        message: t('markedAsWatchedNotification')
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
        message: t('markedAsUnwatchedNotification')
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
        message: `${t('videoLockedNotification', { status: lock ? t('locked') : t('unlocked') })}}`
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
        message: t('generateStaticThumbnailsNotification')
      })
    } catch (error) {
      console.error(error)
    }
  }
  const handleGenerateSpriteThumbnails = async () => {
    try {
      await generateSpriteThumbnailsMutate.mutateAsync({
        axiosPrivate,
        videoId: video.id
      })
      showNotification({
        message: t('generateSpriteThumbnailsNotification')
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
        title: t('copiedToClipboardText'),
        message: t('copiedToClipboardMessage'),
      });

    } catch (err) {
      console.error(err);
      showNotification({
        title: t('error'),
        message: t('clipboardAPIErrorNotification'),
        color: "red",
      });
      prompt(t('clipboardPromptText'), shareUrl);
    }
  }

  return (
    <div>
      <Menu shadow="md" width={200} position="top-end">
        <Menu.Target>
          <ActionIcon color="gray" variant="subtle">
            <IconMenu2 size="1rem" />
          </ActionIcon>
        </Menu.Target>

        <Menu.Dropdown>
          <Menu.Item onClick={openPlaylistDrawer} leftSection={<IconPlaylistAdd style={{ width: rem(14), height: rem(14) }} />}>
            {t('menu.playlists')}
          </Menu.Item>
          <Menu.Item leftSection={<IconInfoCircle style={{ width: rem(14), height: rem(14) }} />}
            onClick={infoModalOpen}>
            {t('menu.info')}
          </Menu.Item>
          <Menu.Item leftSection={<IconHourglassHigh style={{ width: rem(14), height: rem(14) }} />} onClick={handleMarkAsWatched}>
            {t('menu.markAsWatched')}
          </Menu.Item>
          <Menu.Item leftSection={<IconHourglassEmpty style={{ width: rem(14), height: rem(14) }} />} onClick={handleMarkAsUnWatched}>
            {t('menu.markAsUnwatched')}
          </Menu.Item>
          {isLocked.current ? (
            <Menu.Item leftSection={<IconLockOpen style={{ width: rem(14), height: rem(14) }} />} onClick={() => handleLockVideo(false)}>
              {t('menu.unlock')}
            </Menu.Item>
          ) : (
            <Menu.Item leftSection={<IconLock style={{ width: rem(14), height: rem(14) }} />} onClick={() => handleLockVideo(true)}>
              {t('menu.lock')}
            </Menu.Item>
          )}
          <Menu.Item leftSection={<IconPhoto style={{ width: rem(14), height: rem(14) }} />} onClick={handleGenerateStaticThumbnail}>
            {t('menu.regenerateThumbnails')}
          </Menu.Item>
          <Menu.Item leftSection={<IconMovie style={{ width: rem(14), height: rem(14) }} />} onClick={handleGenerateSpriteThumbnails}>
            {t('menu.generateSpriteThumbnails')}
          </Menu.Item>
          <Menu.Item leftSection={<IconShare style={{ width: rem(14), height: rem(14) }} />} onClick={handleShareVideo}>
            {t('menu.share')}
          </Menu.Item>

          {hasPermission(UserRole.Admin) && (
            <>
              <Menu.Divider />
              <Menu.Item
                color="red"
                leftSection={<IconTrash style={{ width: rem(14), height: rem(14) }} />}
                onClick={openDeleteModal}
              >
                {t('menu.delete')}
              </Menu.Item>
            </>
          )}

        </Menu.Dropdown>
      </Menu>

      <Modal
        opened={infoModalOpened}
        onClose={infoModalClose}
        size="xl"
        title={t('videoInformationModalTitle')}
      >
        <VideoInfoModalContent video={video} />
      </Modal>

      <Drawer opened={playlistsDrawerOpened} onClose={closePlaylistDrawer} position="right" title={t('managePlaylistsDrawerTitle')}>
        <PlaylistManageDrawerContent videoId={video.id} />
      </Drawer>

      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t('deleteVideoModalTitle')}>
        <DeleteVideoModalContent video={video} handleClose={closeDeleteModal} />
      </Modal>
    </div>
  );
}

export default VideoMenu;