import { ApiKey, useDeleteApiKey } from "@/app/hooks/useApiKeys";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Button, Code, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  apiKey: ApiKey;
  handleClose: () => void;
};

// DeleteApiKeyModalContent confirms revocation of an API key. The key is
// soft-deleted on the backend (revoked_at) and the verification cache is
// flushed so the key stops working immediately.
const DeleteApiKeyModalContent = ({ apiKey, handleClose }: Props) => {
  const t = useTranslations("AdminApiKeyComponents");
  const [loading, setLoading] = useState(false);

  const deleteApiKey = useDeleteApiKey();
  const axiosPrivate = useAxiosPrivate();

  const handleRevoke = async () => {
    try {
      setLoading(true);
      await deleteApiKey.mutateAsync({ axiosPrivate, id: apiKey.id });
      showNotification({ message: t("revokeNotification") });
      handleClose();
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Text>{t("revokeConfirmText")}</Text>
      <Code block>{JSON.stringify(apiKey, null, 2)}</Code>
      <Button mt={5} color="red" onClick={handleRevoke} loading={loading} fullWidth>
        {t("revokeButton")}
      </Button>
    </div>
  );
};

export default DeleteApiKeyModalContent;
