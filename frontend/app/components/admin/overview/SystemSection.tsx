import { Paper, Title, Text, SimpleGrid, Alert, Button, Anchor, Group } from "@mantine/core";
import { IconCpu, IconDeviceDesktop, IconDatabase, IconUsers } from "@tabler/icons-react";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { GanymedeSystemOverview } from "@/app/hooks/useAdmin";
import { formatBytes } from "@/app/util/util";
import StatCard from "./StatCard";

interface SystemSectionProps {
  system?: GanymedeSystemOverview;
  loading: boolean;
  error: boolean;
  onRetry: () => void;
}

const SystemSection = ({ system, loading, error, onRetry }: SystemSectionProps) => {
  const t = useTranslations("AdminOverviewPage");

  return (
    <Paper shadow="xs" withBorder p="xl">
      <Group justify="space-between" align="center">
        <Title order={4}>{t("system.title")}</Title>
        <Anchor component={Link} href="/admin/info" size="sm">
          {t("system.viewInfo")}
        </Anchor>
      </Group>
      {error ? (
        <Alert color="red" mt="md" title={t("error")}>
          <Button variant="subtle" size="xs" onClick={onRetry}>
            {t("retry")}
          </Button>
        </Alert>
      ) : (
        <SimpleGrid cols={{ base: 1, xs: 2, md: 4 }} pt="md">
          <StatCard
            icon={<IconDatabase size={18} stroke={1.5} />}
            color="blue"
            label={t("system.usedStorage")}
            value={formatBytes(system?.videos_directory_used_space ?? 0, 2)}
            loading={loading}
            description={t("system.freeStorage", { value: formatBytes(system?.videos_directory_free_space ?? 0, 2) })}
          />
          <StatCard icon={<IconCpu size={18} stroke={1.5} />} color="grape" label={t("system.cpuCores")} value={system?.cpu_cores ?? 0} loading={loading} />
          <StatCard icon={<IconDeviceDesktop size={18} stroke={1.5} />} color="cyan" label={t("system.totalMemory")} value={formatBytes(system?.memory_total ?? 0, 0)} loading={loading} />
          <StatCard icon={<IconUsers size={18} stroke={1.5} />} color="green" label={t("system.users")} value={system?.user_count ?? 0} loading={loading} />
        </SimpleGrid>
      )}
      {!error && (
        <Text size="xs" c="dimmed" mt="md">
          {t("system.refreshNote")}
        </Text>
      )}
    </Paper>
  );
};

export default SystemSection;
