import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { StorageFinding, useDeleteStorageFindings, useImportStorageFindings } from "@/app/hooks/useStorage";
import { formatBytes } from "@/app/util/util";
import { Button, Code, ScrollArea, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

export type StorageAction = "delete" | "import";

type Props = {
  action: StorageAction;
  findings: StorageFinding[];
  handleClose: () => void;
}

const StorageActionModalContent = ({ action, findings, handleClose }: Props) => {
  const t = useTranslations('AdminStorageComponents')
  const [loading, setLoading] = useState(false)
  const deleteMutate = useDeleteStorageFindings()
  const importMutate = useImportStorageFindings()
  const axiosPrivate = useAxiosPrivate()

  const totalBytes = findings.reduce((total, finding) => total + finding.size_bytes, 0);

  const handleAction = async () => {
    setLoading(true)
    try {
      const variables = { axiosPrivate, ids: findings.map((finding) => finding.id) };
      const result = action === "delete"
        ? await deleteMutate.mutateAsync(variables)
        : await importMutate.mutateAsync(variables);

      if (result.failed.length === 0) {
        showNotification({
          message: t(action === "delete" ? 'deletedNotification' : 'importedNotification', { length: result.done.length }),
        })
      } else {
        showNotification({
          title: t('partialTitle', { done: result.done.length, failed: result.failed.length }),
          // the path of a finding is absolute on the server, show what the table shows
          message: result.failed.map((failure) => {
            const finding = findings.find((entry) => entry.id === failure.id);
            return `${finding?.relative_path ?? failure.id}: ${failure.error}`;
          }).join("\n"),
          color: "red",
          autoClose: false,
        })
      }
      handleClose()
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Text>
        {t(action === "delete" ? 'deleteText' : 'importText', { length: findings.length, size: formatBytes(totalBytes, 2) })}
      </Text>
      <ScrollArea.Autosize mah={200} my={10}>
        {findings.map((finding) => (
          <Code key={finding.id} block>{finding.relative_path}</Code>
        ))}
      </ScrollArea.Autosize>
      <Button color={action === "delete" ? "red" : "violet"} onClick={handleAction} loading={loading} fullWidth>
        {t(action === "delete" ? 'deleteButton' : 'importButton')}
      </Button>
    </div>
  );
}

export default StorageActionModalContent;
