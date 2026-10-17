import { Paper, Title, Text, SimpleGrid, Group, Box, Skeleton, Alert, Button, Center } from "@mantine/core";
import { DonutChart, BarsList } from "@mantine/charts";
import { IconVideo, IconUsers, IconClock, IconEye } from "@tabler/icons-react";
import { useMemo } from "react";
import { useTranslations } from "next-intl";
import { GanymedeVideoStatistics } from "@/app/hooks/useAdmin";
import { durationToTime, prettyNumber } from "@/app/util/util";
import StatCard from "./StatCard";
import ChartLegend, { toTopEntries } from "./chart-utils";

interface LibrarySectionProps {
  stats?: GanymedeVideoStatistics;
  loading: boolean;
  error: boolean;
  onRetry: () => void;
}

const LibrarySection = ({ stats, loading, error, onRetry }: LibrarySectionProps) => {
  const t = useTranslations("AdminOverviewPage");

  const channelVideoData = useMemo(
    () => toTopEntries(stats?.channel_videos, 8, t("library.otherChannels")),
    [stats, t]
  );
  const videoTypeData = useMemo(
    () => toTopEntries(stats?.video_types, 8, t("library.otherChannels")),
    [stats, t]
  );

  return (
    <Paper shadow="xs" withBorder p="xl">
      <Title order={4}>{t("library.title")}</Title>
      {error ? (
        <Alert color="red" mt="md" title={t("error")}>
          <Button variant="subtle" size="xs" onClick={onRetry}>
            {t("retry")}
          </Button>
        </Alert>
      ) : (
        <>
          <SimpleGrid cols={{ base: 1, xs: 2, md: 4 }} pt="md">
            <StatCard icon={<IconVideo size={18} stroke={1.5} />} color="blue" label={t("library.totalVideos")} value={stats?.video_count ?? 0} loading={loading} />
            <StatCard icon={<IconUsers size={18} stroke={1.5} />} color="teal" label={t("library.totalChannels")} value={stats?.channel_count ?? 0} loading={loading} />
            <StatCard icon={<IconClock size={18} stroke={1.5} />} color="violet" label={t("library.totalDuration")} value={durationToTime(stats?.total_duration_seconds ?? 0)} loading={loading} />
            <StatCard icon={<IconEye size={18} stroke={1.5} />} color="orange" label={t("library.totalViews")} value={prettyNumber((stats?.total_views ?? 0) + (stats?.total_local_views ?? 0))} loading={loading} description={t("library.totalViewsDescription")} />
          </SimpleGrid>
          <SimpleGrid cols={{ base: 1, md: 2 }} pt="xl" spacing="xl">
            <Box>
              <Text fw={600} mb="xs">{t("library.videosPerChannel")}</Text>
              {loading ? (
                <Skeleton height={180} />
              ) : channelVideoData.length === 0 ? (
                <Text size="sm" c="dimmed">{t("library.noData")}</Text>
              ) : (
                <BarsList data={channelVideoData} barsLabel={t("library.channel")} valueLabel={t("library.videos")} />
              )}
            </Box>
            <Box>
              <Text fw={600} mb="xs">{t("library.videoTypes")}</Text>
              {loading ? (
                <Skeleton height={180} />
              ) : videoTypeData.length === 0 ? (
                <Text size="sm" c="dimmed">{t("library.noData")}</Text>
              ) : (
                <Group align="center" justify="center" gap="xl">
                  <DonutChart data={videoTypeData} size={170} thickness={28} tooltipDataSource="segment" chartLabel={stats?.video_count ?? 0} />
                  <ChartLegend data={videoTypeData} />
                </Group>
              )}
            </Box>
          </SimpleGrid>
          {(stats && (stats.video_count ?? 0) === 0 && !loading) && (
            <Center mt="md">
              <Text size="sm" c="dimmed">{t("library.empty")}</Text>
            </Center>
          )}
        </>
      )}
    </Paper>
  );
};

export default LibrarySection;
