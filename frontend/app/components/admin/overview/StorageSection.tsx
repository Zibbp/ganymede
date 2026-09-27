import { Paper, Title, Text, Group, Box, Progress, Skeleton, Alert, Button, Table, Anchor, SimpleGrid } from "@mantine/core";
import { BarsList, DonutChart } from "@mantine/charts";
import { useMemo } from "react";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { GanymedeStorageDistribution, GanymedeSystemOverview } from "@/app/hooks/useAdmin";
import { durationToTime, formatBytes } from "@/app/util/util";
import ChartLegend, { toTopEntries } from "./chart-utils";

interface StorageSectionProps {
  system?: GanymedeSystemOverview;
  storage?: GanymedeStorageDistribution;
  loading: boolean;
  error: boolean;
  onRetry: () => void;
}

const StorageSection = ({ system, storage, loading, error, onRetry }: StorageSectionProps) => {
  const t = useTranslations("AdminOverviewPage");

  const used = system?.videos_directory_used_space ?? 0;
  const free = system?.videos_directory_free_space ?? 0;
  const total = system?.videos_directory_total_space ?? used + free;
  const usedPercent = total > 0 ? (used / total) * 100 : 0;

  const channelStorageData = useMemo(
    () => toTopEntries(storage?.storage_distribution, 8, t("storage.otherChannels"), { hideZero: true }),
    [storage, t]
  );

  const storageByTypeData = useMemo(
    () => toTopEntries(storage?.storage_by_type, 8, t("storage.otherTypes"), { hideZero: true }),
    [storage, t]
  );
  const storageByTypeTotal = useMemo(
    () => storageByTypeData.reduce((sum, item) => sum + item.value, 0),
    [storageByTypeData]
  );

  const largestVideos = useMemo(() => storage?.largest_videos ?? [], [storage]);

  return (
    <Paper shadow="xs" withBorder p="xl">
      <Title order={4}>{t("storage.title")}</Title>
      {error ? (
        <Alert color="red" mt="md" title={t("error")}>
          <Button variant="subtle" size="xs" onClick={onRetry}>
            {t("retry")}
          </Button>
        </Alert>
      ) : (
        <>
          <Box pt="md">
            <Group justify="space-between" mb={6}>
              <Text size="sm" c="dimmed">
                {t("storage.usedOf", { used: formatBytes(used, 2), total: formatBytes(total, 2) })}
              </Text>
              <Text size="sm" fw={700}>
                {usedPercent.toFixed(1)}%
              </Text>
            </Group>
            {loading ? (
              <Skeleton height={12} radius="xl" />
            ) : (
              <Progress value={usedPercent} size="lg" radius="xl" color={usedPercent > 90 ? "red" : usedPercent > 75 ? "orange" : "blue"} />
            )}
            <Group gap="xl" mt="xs">
              <Text size="xs" c="dimmed">{t("storage.free", { value: formatBytes(free, 2) })}</Text>
              <Text size="xs" c="dimmed">{t("storage.total", { value: formatBytes(total, 2) })}</Text>
            </Group>
          </Box>
          <SimpleGrid cols={{ base: 1, md: 2 }} pt="lg" spacing="xl">
            <Box style={{ minWidth: 0 }}>
              <Text fw={600} mb="xs">{t("storage.perChannel")}</Text>
              {loading ? (
                <Skeleton height={180} />
              ) : channelStorageData.length === 0 ? (
                <Text size="sm" c="dimmed">{t("storage.noData")}</Text>
              ) : (
                <BarsList
                  data={channelStorageData}
                  barsLabel={t("storage.channel")}
                  valueLabel={t("storage.size")}
                  valueFormatter={(value) => formatBytes(value, 1)}
                />
              )}
            </Box>
            <Box style={{ minWidth: 0 }}>
              <Text fw={600} mb="xs">{t("storage.byType")}</Text>
              {loading ? (
                <Skeleton height={180} />
              ) : storageByTypeData.length === 0 ? (
                <Text size="sm" c="dimmed">{t("storage.noData")}</Text>
              ) : (
                <Group align="center" justify="center" gap="xl">
                  <DonutChart
                    data={storageByTypeData}
                    size={170}
                    thickness={28}
                    tooltipDataSource="segment"
                    chartLabel={formatBytes(storageByTypeTotal, 1)}
                    valueFormatter={(value) => formatBytes(value, 1)}
                  />
                  <ChartLegend data={storageByTypeData} formatValue={(value) => formatBytes(value, 1)} />
                </Group>
              )}
            </Box>
          </SimpleGrid>
          <Box pt="lg">
            <Text fw={600} mb="xs">{t("storage.largestVideos")}</Text>
            {loading ? (
              <Skeleton height={180} />
            ) : largestVideos.length === 0 ? (
              <Text size="sm" c="dimmed">{t("storage.noLargeVideos")}</Text>
            ) : (
              <Table striped highlightOnHover withTableBorder withColumnBorders={false}>
                <Table.Tbody>
                  {largestVideos.map((video) => (
                    <Table.Tr key={video.id}>
                      <Table.Td>
                        <Anchor component={Link} href={`/videos/${video.id}`} size="sm" lineClamp={1}>
                          {video.title}
                        </Anchor>
                        <Text size="xs" c="dimmed" lineClamp={1}>
                          {video.channel_name} · {durationToTime(video.duration)}
                        </Text>
                      </Table.Td>
                      <Table.Td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                        <Text size="sm">{formatBytes(video.storage_size_bytes, 1)}</Text>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            )}
          </Box>
        </>
      )}
    </Paper>
  );
};

export default StorageSection;
