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
} from '@tabler/icons-react';
import VideoInfoModalContent from './modals/InfoModalContent';
import { Video } from '@/app/hooks/useVideos';
import PlaylistManageDrawerContent from '../playlist/ManageDrawerContent';

type Props = {
  video: Video
}

const VideoMenu = ({ video }: Props) => {
  const [infoModalOpened, { open: infoModalOpen, close: infoModalClose }] = useDisclosure(false);
  const [playlistsDrawerOpened, { open: openPlaylistDrawer, close: closePlaylistDrawer }] = useDisclosure(false);

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
          <Menu.Item leftSection={<IconHourglassHigh style={{ width: rem(14), height: rem(14) }} />}>
            Mark as Watched
          </Menu.Item>
          <Menu.Item leftSection={<IconHourglassEmpty style={{ width: rem(14), height: rem(14) }} />}>
            Mark as Unwatched
          </Menu.Item>
          <Menu.Item leftSection={<IconLockOpen style={{ width: rem(14), height: rem(14) }} />}>
            Lock
          </Menu.Item>
          <Menu.Item leftSection={<IconPhoto style={{ width: rem(14), height: rem(14) }} />}>
            Regenerate Thumbnail
          </Menu.Item>
          <Menu.Item leftSection={<IconShare style={{ width: rem(14), height: rem(14) }} />}>
            Share
          </Menu.Item>


          <Menu.Divider />

          <Menu.Item
            color="red"
            leftSection={<IconTrash style={{ width: rem(14), height: rem(14) }} />}
          >
            Delete
          </Menu.Item>
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
    </div>
  );
}

export default VideoMenu;