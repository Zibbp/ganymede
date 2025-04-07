import { User } from "@/app/hooks/useAuthentication";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useDeleteUser } from "@/app/hooks/useUsers";
import { Button, Code, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";

type Props = {
  user: User
  handleClose: () => void;
}


const DeleteUserModalContent = ({ user, handleClose }: Props) => {
  const t = useTranslations('AdminUserComponents')
  const [loading, setLoading] = useState(false)

  const deleteUserMutate = useDeleteUser()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteQueue = async () => {
    try {
      setLoading(true)

      await deleteUserMutate.mutateAsync({ axiosPrivate: axiosPrivate, userId: user.id })

      showNotification({
        message: t('deleteNotification')
      })

      handleClose()
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Text>{t('deleteConfirmText')}</Text>

      <Code block>{JSON.stringify(user, null, 2)}</Code>

      <Button mt={5} color="red" onClick={handleDeleteQueue} loading={loading} fullWidth>{t('deleteButton')}</Button>
    </div>
  );
}

export default DeleteUserModalContent;