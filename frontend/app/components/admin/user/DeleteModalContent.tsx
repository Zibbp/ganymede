import { User } from "@/app/hooks/useAuthentication";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useDeleteUser } from "@/app/hooks/useUsers";
import { Button, Code, Text } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";

type Props = {
  user: User
  handleClose: () => void;
}


const DeleteUserModalContent = ({ user, handleClose }: Props) => {
  const [loading, setLoading] = useState(false)

  const deleteUserMutate = useDeleteUser()
  const axiosPrivate = useAxiosPrivate()

  const handleDeleteQueue = async () => {
    try {
      setLoading(true)

      await deleteUserMutate.mutateAsync({ axiosPrivate: axiosPrivate, userId: user.id })

      showNotification({
        message: "User deleted"
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
      <Text>Are you sure you want to delete this user?</Text>

      <Code block>{JSON.stringify(user, null, 2)}</Code>

      <Button mt={5} color="red" onClick={handleDeleteQueue} loading={loading} fullWidth>Delete User</Button>
    </div>
  );
}

export default DeleteUserModalContent;