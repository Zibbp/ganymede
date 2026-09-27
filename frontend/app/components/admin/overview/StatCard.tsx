import { Paper, Group, ThemeIcon, Text, Skeleton } from "@mantine/core";
import { ReactNode } from "react";
import classes from "./StatCard.module.css";

interface StatCardProps {
  icon: ReactNode;
  color: string;
  label: string;
  value: ReactNode;
  description?: string;
  loading?: boolean;
}

// Small KPI card used across the admin overview dashboard.
const StatCard = ({ icon, color, label, value, description, loading }: StatCardProps) => {
  return (
    <Paper withBorder p="md" radius="md">
      <Group justify="space-between">
        <Text size="xs" c="dimmed" className={classes.title}>
          {label}
        </Text>
        <ThemeIcon color={color} variant="light" size="md" radius="md">
          {icon}
        </ThemeIcon>
      </Group>
      <Group align="flex-end" gap="xs" mt="md">
        {loading ? (
          <Skeleton height={24} width={80} />
        ) : (
          <Text className={classes.value}>{value}</Text>
        )}
      </Group>
      {description && (
        <Text fz="xs" c="dimmed" mt={7}>
          {description}
        </Text>
      )}
    </Paper>
  );
};

export default StatCard;
