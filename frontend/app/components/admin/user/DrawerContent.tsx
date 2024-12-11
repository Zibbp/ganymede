import { User, UserRole } from "@/app/hooks/useAuthentication";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { useEditUser } from "@/app/hooks/useUsers";
import { Button, TextInput, Select } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useState } from "react";
import { z } from "zod";

type Props = {
  user: User
  handleClose: () => void;
}


const schema = z.object({
  id: z.string().min(2, { message: "ID should have at least 2 characters" }).max(50),
  username: z.string().min(3, { message: "Username should have at least 3 characters" }),
  role: z.nativeEnum(UserRole)
})

const AdminUserDrawerContent = ({ user, handleClose }: Props) => {
  const axiosPrivate = useAxiosPrivate()
  const useEditUserMutate = useEditUser()
  const [loading, setLoading] = useState(false)

  const form = useForm({
    mode: "controlled",
    initialValues: {
      id: user.id,
      username: user.username,
      role: user.role,
      oauth: user.oauth,
      created_at: user.created_at
    },

    validate: zodResolver(schema),
  })

  const handleSubmitForm = async () => {
    const formValues = form.getValues()
    console.debug(`User form submit - ${formValues}`)

    // @ts-expect-error uncessary
    const submitUser: User = { ...formValues }

    // edit user
    try {
      setLoading(true)

      await useEditUserMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        user: submitUser
      })

      showNotification({
        message: "User edited"
      })

      handleClose()
    } catch (error) {
      setLoading(false)
      console.error(error)
    }
  }

  const selectorUserRole = Object.values(UserRole).map((type) => ({
    value: type,
    label: type.charAt(0).toUpperCase() + type.slice(1),
  }));

  return (
    <div>
      <form onSubmit={form.onSubmit(() => {
        handleSubmitForm()
      })}>
        <TextInput
          disabled={true}
          label="ID"
          placeholder="Auto generated"
          key={form.key('id')}
          {...form.getInputProps('id')}
        />

        <TextInput
          withAsterisk
          label="Username"
          placeholder="ganymede"
          key={form.key('username')}
          {...form.getInputProps('username')}
        />

        <Select
          label="Role"
          data={selectorUserRole}
          key={form.key('role')}
          {...form.getInputProps('role')}
        />

        <TextInput
          disabled
          label="Created At"
          placeholder="ganymede"
          key={form.key('created_at')}
          {...form.getInputProps('created_at')}
        />
        <Button mt={10} type="submit" loading={loading} fullWidth>Edit User</Button>
      </form>

    </div>
  );
}

export default AdminUserDrawerContent;