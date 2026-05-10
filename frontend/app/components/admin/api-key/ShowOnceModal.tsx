import {
  ActionIcon,
  Alert,
  Button,
  Code,
  CopyButton,
  Group,
  Stack,
  Text,
  Tooltip,
} from "@mantine/core";
import { IconAlertTriangle, IconCheck, IconCopy } from "@tabler/icons-react";
import { useTranslations } from "next-intl";

type Props = {
  // secret is the full Bearer token. It is shown to the admin exactly
  // once at creation time; subsequent listings only return the prefix.
  secret: string;
  handleClose: () => void;
};

const ShowOnceModalContent = ({ secret, handleClose }: Props) => {
  const t = useTranslations("AdminApiKeyComponents");

  return (
    <Stack gap="md">
      <Alert color="red" icon={<IconAlertTriangle size={18} />}>
        {t("showOnce.warning")}
      </Alert>

      <Text size="sm">
        {t.rich("showOnce.instructions", {
          code: (chunks) => <Code>{chunks}</Code>,
        })}
      </Text>

      <Group gap="xs" wrap="nowrap" align="flex-start">
        <Code block style={{ flex: 1, wordBreak: "break-all" }}>
          {secret}
        </Code>
        <CopyButton value={secret} timeout={2000}>
          {({ copied, copy }) => (
            <Tooltip label={copied ? t("showOnce.copied") : t("showOnce.copy")}>
              <ActionIcon
                size="lg"
                variant="light"
                color={copied ? "teal" : "blue"}
                onClick={copy}
              >
                {copied ? <IconCheck size={18} /> : <IconCopy size={18} />}
              </ActionIcon>
            </Tooltip>
          )}
        </CopyButton>
      </Group>

      <Button
        color="red"
        variant="filled"
        onClick={handleClose}
        fullWidth
      >
        {t("showOnce.close")}
      </Button>
    </Stack>
  );
};

export default ShowOnceModalContent;
