"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useGetGanymedeStorageDistribution, useGetGanymedeSystemOverview, useGetGanymedeVideoStatistics } from "@/app/hooks/useAdmin";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Container, Title, Text, Paper, SimpleGrid, Group, RingProgress, Stack, Box, useMantineTheme } from "@mantine/core";
import classes from "./AdminOverviewPage.module.css"
import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { formatBytes, usePageTitle } from "@/app/util/util";
import { IconCpu, IconDatabase, IconDeviceDesktop, IconUser, IconVideo } from "@tabler/icons-react";
import { PieChart } from "@mantine/charts";
const colors = [
  'indigo.6', 'yellow.6', 'teal.6', 'gray.6',
  'red.6', 'green.6', 'blue.6', 'cyan.6', 'pink.6',
  'orange.6', 'violet.6', 'lime.6'
];

// hashStringToIndex is a utility function to convert a string to an index based on its hash.
function hashStringToIndex(str: string, max: number): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = (hash << 5) - hash + str.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash) % max;
}

const AdminOverviewPage = () => {
  const t = useTranslations("AdminOverviewPage");
  const theme = useMantineTheme()
  usePageTitle(t('title'));
  const axiosPrivate = useAxiosPrivate()

  const { data: videoStatistics, isPending: isPendingVideoStatistics, isError: isErrorVideoStatistics } = useGetGanymedeVideoStatistics(axiosPrivate)
  const { data: systemOverview, isPending: isPendingSystemOverview, isError: isErrorSystemOverview } = useGetGanymedeSystemOverview(axiosPrivate)
  const { data: storageDistribution, isPending: isPendingStorageDistribution, isError: isErrorStorageDistribution } = useGetGanymedeStorageDistribution(axiosPrivate)

  const [channelVideoMapChartData, setChannelVideoMapChartData] = useState<
    { name: string; value: number; color: string }[]
  >([]);
  const [videoTypesMapChartData, setVideoTypesMapChartData] = useState<
    { name: string; value: number; color: string }[]
  >([]);
  const [channelStorageMapChartData, setChannelStorageMapChartData] = useState<
    { name: string; value: number; color: string }[]
  >([]);

  useEffect(() => {
    if (!videoStatistics?.channel_videos || !videoStatistics?.video_types) return;

    const mapped = Object.entries(videoStatistics.channel_videos).map(
      ([name, value]) => ({
        name,
        value,
        color: colors[hashStringToIndex(name, colors.length)]
      })
    );

    setChannelVideoMapChartData(mapped);

    const videoTypesMapped = Object.entries(videoStatistics.video_types).map(
      ([name, value]) => ({
        name,
        value,
        color: colors[hashStringToIndex(name, colors.length)]
      })
    );
    setVideoTypesMapChartData(videoTypesMapped);
  }, [videoStatistics]);

  useEffect(() => {
    if (!storageDistribution?.storage_distribution) return;

    const mapped = Object.entries(storageDistribution.storage_distribution)
      .filter(([, value]) => value !== 0)
      .map(([name, value]) => ({
        name,
        value: value,
        color: colors[hashStringToIndex(name, colors.length)]
      }));
    setChannelStorageMapChartData(mapped);
  }, [storageDistribution]);

  if (isPendingStorageDistribution || isPendingVideoStatistics || isPendingSystemOverview) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isErrorStorageDistribution || isErrorVideoStatistics || isErrorSystemOverview) return <div>{t('error')}</div>


  return (
    <Container mt={10} size={"7xl"}>
      {/* System Overview */}
      <Paper shadow="xs" withBorder p="xl">
        <Title order={4} >
          {t('system.title')}
          {/* Storage overview */}
          <SimpleGrid cols={{ base: 1, xs: 2, md: 3 }} pt={20}>
            {/* Used storage */}
            <Paper withBorder p="md" radius="md" key="Used Storage">
              <Group justify="space-between">
                <Text size="xs" c="dimmed" className={classes.title}>
                  {t('system.usedStorageText')}
                </Text>
                <IconDatabase className={classes.icon} size={22} stroke={1.5} />
              </Group>
              <Group align="flex-end" gap="xs" mt={15}>
                <Text className={classes.value}>{formatBytes(systemOverview.videos_directory_used_space ?? 0, 2)}</Text>
              </Group>
              <Text fz="xs" c="dimmed" mt={7}>
                {t('system.usedStorageDescription')}
              </Text>
            </Paper>
            {/* Free storage */}
            <Paper withBorder p="md" radius="md" key="Available Storage">
              <Group justify="space-between">
                <Text size="xs" c="dimmed" className={classes.title}>
                  {t('system.availableStorageText')}
                </Text>
                <IconDatabase className={classes.icon} size={22} stroke={1.5} />
              </Group>
              <Group align="flex-end" gap="xs" mt={15}>
                <Text className={classes.value}>{formatBytes(systemOverview.videos_directory_free_space ?? 0, 2)}</Text>
              </Group>
              <Text fz="xs" c="dimmed" mt={7}>
                {t('system.availableStorageDescription')}
              </Text>
            </Paper>
            {/* Usage */}
            <Paper withBorder p="md" radius="md" key="Storage Usage">
              <Group justify="space-between">
                <Text size="xs" c="dimmed" className={classes.title}>
                  {t('system.storageUsedText')}
                </Text>
                <IconDatabase className={classes.icon} size={22} stroke={1.5} />
              </Group>
              <Group align="flex-end" >
                <RingProgress
                  label={
                    <Text c="blue" fw={700} ta="center" size="xl">
                      {(((systemOverview.videos_directory_used_space ?? 0) / Math.max((systemOverview.videos_directory_used_space ?? 0) + (systemOverview.videos_directory_free_space ?? 0), 1)) * 100).toFixed(2)}%
                    </Text>
                  }
                  sections={[
                    {
                      value: ((systemOverview.videos_directory_used_space ?? 0) / Math.max((systemOverview.videos_directory_used_space ?? 0) + (systemOverview.videos_directory_free_space ?? 0), 1)) * 100,
                      color: 'blue',
                    },
                  ]}
                  size={120}
                />
              </Group>
            </Paper>
          </SimpleGrid>

          {/* Resources overview */}
          <SimpleGrid cols={{ base: 1, xs: 2, md: 2 }} pt={15}>
            {/* CPU Cores */}
            <Paper withBorder p="md" radius="md" key="CPU Cores">
              <Group justify="space-between">
                <Text size="xs" c="dimmed" className={classes.title}>
                  {t('system.cpuCoresText')}
                </Text>
                <IconCpu className={classes.icon} size={22} stroke={1.5} />
              </Group>
              <Group align="flex-end" gap="xs" mt={15}>
                <Text className={classes.value}>{systemOverview.cpu_cores}</Text>
              </Group>
            </Paper>
            {/* Total memory */}
            <Paper withBorder p="md" radius="md" key="Total Memory">
              <Group justify="space-between">
                <Text size="xs" c="dimmed" className={classes.title}>
                  {t('system.totalMemoryText')}
                </Text>
                <IconDeviceDesktop className={classes.icon} size={22} stroke={1.5} />
              </Group>
              <Group align="flex-end" gap="xs" mt={15}>
                <Text className={classes.value}>{formatBytes(systemOverview.memory_total ?? 0, 0)}</Text>
              </Group>
            </Paper>
          </SimpleGrid>
        </Title>
      </Paper>

      {/* Video Statistics */}
      <Paper shadow="xs" withBorder p="xl" mt={15}>
        <Title order={4} >
          {t('videoStatistics.title')}</Title>
        {/* Video statistics */}
        <SimpleGrid cols={{ base: 1, xs: 2, md: 2 }} pt={20}>
          {/* Total videos */}
          <Paper withBorder p="md" radius="md" key="Used Storage">
            <Group justify="space-between">
              <Text size="xs" c="dimmed" className={classes.title}>
                {t('videoStatistics.totalVideosText')}
              </Text>
              <IconVideo className={classes.icon} size={22} stroke={1.5} />
            </Group>
            <Group align="flex-end" gap="xs" mt={15}>
              <Text className={classes.value}>{videoStatistics?.video_count}</Text>
            </Group>
          </Paper>
          {/* Total channels */}
          <Paper withBorder p="md" radius="md" key="Available Storage">
            <Group justify="space-between">
              <Text size="xs" c="dimmed" className={classes.title}>
                {t('videoStatistics.totalChannelsText')}
              </Text>
              <IconUser className={classes.icon} size={22} stroke={1.5} />
            </Group>
            <Group align="flex-end" gap="xs" mt={15}>
              <Text className={classes.value}>{videoStatistics?.channel_count}</Text>
            </Group>
          </Paper>
        </SimpleGrid>

        <SimpleGrid cols={{ base: 1, xs: 2, md: 2 }} pt={20}>
          <Box>
            {/* Channel videos map */}
            <Title order={4} mt={10} mb={0}>{t('videoStatistics.channelVideosMapTitle')}</Title>
            <Group mt="xs" gap="xs" justify="center" >
              <PieChart
                withLabelsLine
                labelsPosition="outside"
                labelsType="value"
                withLabels
                size={250}
                withTooltip
                tooltipDataSource="segment"
                mx="auto"
                data={channelVideoMapChartData} />
              <Stack gap={4} ml={0}>
                {[...channelVideoMapChartData]
                  .sort((a, b) => b.value - a.value)
                  .map((item: { name: string; value: number; color: string }) => (
                    <Group key={item.name} gap={6} align="center">
                      <Box
                        w={16}
                        h={16}
                        bg={theme.colors[item.color.split(".")[0]][parseInt(item.color.split(".")[1])]}
                      />
                      <Text size="sm">{item.name} ({item.value})</Text>
                    </Group>
                  ))}
              </Stack>
            </Group>
          </Box>

          <Box>
            {/* Video types map */}
            <Title order={4} mt={10} mb={0}>{t('videoStatistics.videoTypeMapTitle')}</Title>
            <Group mt="xs" gap="xs" justify="center" >
              <PieChart
                withLabelsLine
                labelsPosition="outside"
                labelsType="value"
                withLabels
                size={250}
                withTooltip
                tooltipDataSource="segment"
                mx="auto"
                data={videoTypesMapChartData} />
              <Stack gap={4} ml={0}>
                {[...videoTypesMapChartData]
                  .sort((a, b) => b.value - a.value)
                  .map((item: { name: string; value: number; color: string }) => (
                    <Group key={item.name} gap={6} align="center">
                      <Box
                        w={16}
                        h={16}
                        bg={theme.colors[item.color.split(".")[0]][parseInt(item.color.split(".")[1])]}
                      />
                      <Text size="sm">{item.name} ({item.value})</Text>
                    </Group>
                  ))}
              </Stack>
            </Group>
          </Box>
        </SimpleGrid>
      </Paper>

      {/* Storage Distribution */}
      <Paper shadow="xs" withBorder p="xl" mt={15} mb={15}>
        <Title order={4} >
          {t('storageDistribution.title')}</Title>
        <SimpleGrid cols={{ base: 1, xs: 2, md: 2 }} pt={20}>
          <Box>
            {/* Channel storage usage map */}
            <Title order={4} mt={10} mb={0}>{t('storageDistribution.channelStorageDistributionText')}</Title>
            <Group mt="xs" gap="xs" justify="center" >
              <PieChart
                withLabelsLine
                labelsPosition="outside"
                labelsType="percent"
                withLabels
                size={250}
                mx="auto"
                data={channelStorageMapChartData} />
              <Stack gap={4} ml={0}>
                {[...channelStorageMapChartData]
                  .sort((a, b) => b.value - a.value)
                  .map((item: { name: string; value: number; color: string }) => (
                    <Group key={item.name} gap={6} align="center">
                      <Box
                        w={16}
                        h={16}
                        bg={theme.colors[item.color.split(".")[0]][parseInt(item.color.split(".")[1])]}
                      />
                      <Text size="sm">{item.name} ({formatBytes(item.value ?? 0, 1)})</Text>
                    </Group>
                  ))}
              </Stack>
            </Group>
          </Box>
        </SimpleGrid>
      </Paper>
    </Container >
  );
}

export default AdminOverviewPage;