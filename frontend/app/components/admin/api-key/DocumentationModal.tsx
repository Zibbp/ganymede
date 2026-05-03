import {
  Accordion,
  Alert,
  Badge,
  Code,
  Group,
  List,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { IconInfoCircle } from "@tabler/icons-react";
import { useTranslations } from "next-intl";
import { ApiKeyTier } from "@/app/hooks/useApiKeys";
import {
  ResourceDocs,
  SCOPE_DOCS,
  SESSION_ONLY_NOTES,
  TierDocs,
} from "./scopeDocs";

// tierBadgeColor mirrors the badges on the list page so the visual
// language for read/write/admin is consistent.
const tierBadgeColor = (tier: ApiKeyTier): string => {
  switch (tier) {
    case ApiKeyTier.Admin:
      return "red";
    case ApiKeyTier.Write:
      return "yellow";
    case ApiKeyTier.Read:
      return "blue";
    default:
      return "gray";
  }
};

// orderedTiers ensures the modal always renders read → write → admin,
// independent of object property iteration order.
const orderedTiers: ApiKeyTier[] = [
  ApiKeyTier.Read,
  ApiKeyTier.Write,
  ApiKeyTier.Admin,
];

const TierBlock = ({
  tier,
  docs,
  resourceKey,
}: {
  tier: ApiKeyTier;
  docs: TierDocs;
  resourceKey: string;
}) => (
  <Stack gap={4} mt="xs">
    <Group gap="xs">
      <Badge color={tierBadgeColor(tier)} variant="light" ff="monospace">
        {`${resourceKey}:${tier}`}
      </Badge>
      <Text size="sm">{docs.summary}</Text>
    </Group>
    {docs.routes.length > 0 && (
      <List spacing={2} size="xs" withPadding>
        {docs.routes.map((r) => (
          <List.Item key={r}>
            <Code>{r}</Code>
          </List.Item>
        ))}
      </List>
    )}
  </Stack>
);

const ResourcePanel = ({ docs }: { docs: ResourceDocs }) => (
  <Accordion.Item value={String(docs.resource)}>
    <Accordion.Control>
      <Group gap="xs">
        <Text fw={600}>{docs.label}</Text>
      </Group>
    </Accordion.Control>
    <Accordion.Panel>
      <Text size="sm" c="dimmed" mb="xs">
        {docs.description}
      </Text>
      {orderedTiers.map((tier) => {
        const tierDocs = docs.tiers[tier];
        if (!tierDocs) return null;
        return (
          <TierBlock
            key={tier}
            tier={tier}
            docs={tierDocs}
            resourceKey={String(docs.resource)}
          />
        );
      })}
    </Accordion.Panel>
  </Accordion.Item>
);

const DocumentationModalContent = () => {
  const t = useTranslations("AdminApiKeyComponents");

  return (
    <Stack gap="md">
      <Alert
        color="blue"
        variant="light"
        icon={<IconInfoCircle size={18} />}
        title={t("docs.scopeFormatTitle")}
      >
        <Text size="sm">{t("docs.scopeFormatBody")}</Text>
      </Alert>

      <Accordion variant="separated" multiple>
        {SCOPE_DOCS.map((docs) => (
          <ResourcePanel key={String(docs.resource)} docs={docs} />
        ))}
      </Accordion>

      <div>
        <Title order={5} mb={6}>
          {t("docs.sessionOnlyTitle")}
        </Title>
        <Text size="sm" c="dimmed" mb="xs">
          {t("docs.sessionOnlyBody")}
        </Text>
        <List spacing={4} size="sm">
          {SESSION_ONLY_NOTES.map((note) => (
            <List.Item key={note.area}>
              <Text size="sm">
                <Code>{note.area}</Code> — {note.reason}
              </Text>
            </List.Item>
          ))}
        </List>
      </div>
    </Stack>
  );
};

export default DocumentationModalContent;
