"use client"
import LibrarySection from "@/app/components/admin/overview/LibrarySection";
import StorageSection from "@/app/components/admin/overview/StorageSection";
import SystemSection from "@/app/components/admin/overview/SystemSection";
import { useGetGanymedeStorageDistribution, useGetGanymedeSystemOverview, useGetGanymedeVideoStatistics } from "@/app/hooks/useAdmin";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { usePageTitle } from "@/app/util/util";
import { ActionIcon, Container, Group, Stack, Title, Tooltip } from "@mantine/core";
import { IconRefresh } from "@tabler/icons-react";
import { useTranslations } from "next-intl";

const AdminOverviewPage = () => {
  const t = useTranslations("AdminOverviewPage");
  usePageTitle(t('title'));
  const axiosPrivate = useAxiosPrivate()

  const videoStatsQuery = useGetGanymedeVideoStatistics(axiosPrivate)
  const systemQuery = useGetGanymedeSystemOverview(axiosPrivate)
  const storageQuery = useGetGanymedeStorageDistribution(axiosPrivate)

  const isRefreshing = videoStatsQuery.isFetching || systemQuery.isFetching || storageQuery.isFetching;

  const handleRefresh = () => {
    videoStatsQuery.refetch();
    systemQuery.refetch();
    storageQuery.refetch();
  };

  // Storage section needs both the filesystem totals (system) and the
  // per-channel breakdown; either query failing puts it in an error state.
  const storageError = systemQuery.isError || storageQuery.isError;
  const storageLoading = systemQuery.isPending || storageQuery.isPending;

  return (
    <Container size="7xl" mt={10} mb={15}>
      <Group justify="space-between" align="center" mb="md">
        <Title order={3}>{t('header')}</Title>
        <Tooltip label={t('refresh')}>
          <ActionIcon variant="subtle" onClick={handleRefresh} loading={isRefreshing} aria-label={t('refresh')}>
            <IconRefresh size={18} stroke={1.5} />
          </ActionIcon>
        </Tooltip>
      </Group>
      <Stack gap="md">
        <LibrarySection
          stats={videoStatsQuery.data}
          loading={videoStatsQuery.isPending}
          error={videoStatsQuery.isError}
          onRetry={() => videoStatsQuery.refetch()}
        />
        <StorageSection
          system={systemQuery.data}
          storage={storageQuery.data}
          loading={storageLoading}
          error={storageError}
          onRetry={handleRefresh}
        />
        <SystemSection
          system={systemQuery.data}
          loading={systemQuery.isPending}
          error={systemQuery.isError}
          onRetry={() => systemQuery.refetch()}
        />
      </Stack>
    </Container>
  );
}

export default AdminOverviewPage;
