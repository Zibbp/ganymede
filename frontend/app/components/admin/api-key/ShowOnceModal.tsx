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
import { useState } from "react";

type Props = {
  // secret is the full Bearer token. It is shown to the admin exactly
  // once at creation time; subsequent listings only return the prefix.
  secret: string;
  handleClose: () => void;
};

const ShowOnceModalContent = ({ secret, handleClose }: Props) => {
  const t = useTranslations("AdminApiKeyComponents");
  // Acknowledge gates the close button so an admin can't dismiss the
  // modal accidentally before saving the secret somewhere.
  const [acknowledged, setAcknowledged] = useState(false);

  return (
    <Stack gap="md">
      <Alert color="red" icon={<IconAlertTriangle size={18} />}>
        {t("showOnce.warning")}
      </Alert>

      <Text size="sm">{t("showOnce.instructions")}</Text>

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
                onClick={() => {
                  copy();
                  // Copying counts as acknowledgement — the admin clearly
                  // intends to use the secret elsewhere.
                  setAcknowledged(true);
                }}
              >
                {copied ? <IconCheck size={18} /> : <IconCopy size={18} />}
              </ActionIcon>
            </Tooltip>
          )}
        </CopyButton>
      </Group>

      <Button
        color="red"
        variant={acknowledged ? "filled" : "light"}
        disabled={!acknowledged}
        onClick={handleClose}
        fullWidth
      >
        {acknowledged ? t("showOnce.closeAcknowledged") : t("showOnce.closeBlocked")}
      </Button>
    </Stack>
  );
};

export default ShowOnceModalContent;
