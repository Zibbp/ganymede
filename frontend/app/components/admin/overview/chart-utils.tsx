import { Box, Group, Text } from "@mantine/core";
import type { MantineColor } from "@mantine/core";

// Fixed palette that stays readable in both light and dark color schemes.
export const CHART_COLORS: MantineColor[] = [
  "blue.6",
  "teal.6",
  "violet.6",
  "orange.6",
  "pink.6",
  "cyan.6",
  "lime.6",
  "yellow.6",
  "red.6",
  "indigo.6",
];

export interface NamedValue {
  name: string;
  value: number;
  color: string;
}

// Sort entries descending and bucket everything past `limit` into an "Other" entry
// so charts stay readable on instances with many channels.
export const toTopEntries = (
  record: Record<string, number> | undefined,
  limit: number,
  otherName: string,
  opts?: { hideZero?: boolean }
): NamedValue[] => {
  if (!record) return [];
  let entries = Object.entries(record);
  if (opts?.hideZero) {
    entries = entries.filter(([, value]) => value !== 0);
  }
  entries.sort((a, b) => b[1] - a[1]);
  const top = entries.slice(0, limit).map(([name, value], index) => ({
    name,
    value,
    color: CHART_COLORS[index % CHART_COLORS.length],
  }));
  const rest = entries.slice(limit);
  if (rest.length > 0) {
    top.push({
      name: otherName,
      value: rest.reduce((sum, [, value]) => sum + value, 0),
      color: "gray.5",
    });
  }
  return top;
};

interface ChartLegendProps {
  data: NamedValue[];
  formatValue?: (value: number) => string;
}

// Compact legend rendered next to DonutChart (v9 has no built-in legend).
const ChartLegend = ({ data, formatValue }: ChartLegendProps) => {
  return (
    <Box>
      {[...data]
        .sort((a, b) => b.value - a.value)
        .map((item) => (
          <Group key={item.name} gap={6} align="center" wrap="nowrap">
            <Box w={12} h={12} bg={item.color} style={{ borderRadius: 3, flexShrink: 0 }} />
            <Text size="sm" truncate>
              {item.name} ({formatValue ? formatValue(item.value) : item.value})
            </Text>
          </Group>
        ))}
    </Box>
  );
};

export default ChartLegend;
