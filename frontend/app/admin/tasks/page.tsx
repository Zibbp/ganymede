"use client"
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Card, Container, Title, Text, Tooltip, ActionIcon, Group, Box } from "@mantine/core";
import classes from "./AdminTasksPage.module.css"
import { useEffect, useState } from "react";
import { IconPlayerPlay } from "@tabler/icons-react";
import { Task, useStartTask } from "@/app/hooks/useTasks";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { usePageTitle } from "@/app/util/util";

const AdminTasksPage = () => {
  const t = useTranslations('AdminTasksPage')
  usePageTitle(t('title'))
  const axiosPrivate = useAxiosPrivate()
  const [loading, setLoading] = useState(false)

  const startTaskMutate = useStartTask()

  const startTask = async (task: Task) => {
    try {
      setLoading(true)

      await startTaskMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        task: task
      })

      showNotification({
        message: t('taskStartedNotification')
      })
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Container mt={15}>
        <Card withBorder p="xl" radius={"sm"}>

          <Title>{t('header')}</Title>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('checkWatchedChannelsLive')}</Text>
              <Text size="xs">{t('checkWatchedChannelsLiveDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.CheckLive)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('checkWatchedChannelsVideo')}</Text>
              <Text size="xs">{t('checkWatchedChannelsVideoDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.CheckVod)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('checkWatchedChannelsClips')}</Text>
              <Text size="xs">{t('checkWatchedChannelsClipsDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.CheckClips)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('storageTemplateMigration')}</Text>
              <Text size="xs">{t('storageTemplateMigrationDescription')} <a className={classes.link} target="_blank" href="https://github.com/Zibbp/ganymede/wiki/Storage-Templates-and-Migration">Documentation</a>.</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.StorageMigration)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('pruneVideos')}</Text>
              <Text size="xs">{t('pruneVideosDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.PruneVideo)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('jwks')}</Text>
              <Text size="xs">{t('jwksDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.GetJWKS)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('saveChaptersForVideos')}</Text>
              <Text size="xs">{t('saveChaptersForVideosDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.SaveChapters)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('updateLiveStreamIds')}</Text>
              <Text size="xs">{t('updateLiveStreamIdsDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.UpdateStreamVodIds)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('generateSpriteThumbnails')}</Text>
              <Text size="xs">{t('generateSpriteThumbnailsDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.GenerateSpriteThumbnails)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5} wrap="nowrap">
            <Box>
              <Text fw={"bold"}>{t('updateVideoStorageUsage')}</Text>
              <Text size="xs">{t('updateVideoStorageUsageDescription')}</Text>
            </Box>
            <Tooltip label={t('startTaskButton')}>
              <ActionIcon
                onClick={() => startTask(Task.UpdateVideoStorageUsage)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>


        </Card>
      </Container>
    </div>
  );
}

export default AdminTasksPage;