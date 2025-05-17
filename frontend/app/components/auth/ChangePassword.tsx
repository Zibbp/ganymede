import { authChangePassword } from "@/app/hooks/useAuthentication";
import { Button, PasswordInput } from "@mantine/core";
import { useForm, zodResolver } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { useTranslations } from "next-intl";
import { useState } from "react";
import { z } from "zod";

type Props = {
  handleClose: () => void;
}

const AuthChangePassword = ({ handleClose }: Props) => {
  const t = useTranslations('AuthComponents')
  const [loading, setLoading] = useState(false)

  const schema = z.object({
    password: z.string().min(8, { message: t('validation.password') }),
    new_password: z.string().min(8, { message: t('validation.password') }),
    confirm_new_password: z.string().min(8, { message: t('validation.password') })
  })

  const form = useForm({
    mode: "uncontrolled",
    initialValues: {
      password: '',
      new_password: '',
      confirm_new_password: '',
    },

    validate: zodResolver(schema),
  });

  const handleSubmit = async (password: string, newPassword: string, confirmNewPassword: string) => {
    if (newPassword != confirmNewPassword) {
      showNotification({
        message: t('passwordMustMatch'),
        color: "red"
      })
      return
    }
    try {
      setLoading(true)
      await authChangePassword(password, newPassword, confirmNewPassword)
      showNotification({
        message: t('passwordChangeSuccess')
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
      <form onSubmit={form.onSubmit((values) => handleSubmit(values.password, values.new_password, values.confirm_new_password))}>
        <PasswordInput
          label={t('currentPasswordLabel')}
          key={form.key('password')}
          {...form.getInputProps('password')}
          radius="md"
        />
        <PasswordInput
          label={t('newPasswordLabel')}
          key={form.key('new_password')}
          {...form.getInputProps('new_password')}
          radius="md"
        />
        <PasswordInput
          label={t('confirmPasswordLabel')}
          key={form.key('confirm_new_password')}
          {...form.getInputProps('confirm_new_password')}
          radius="md"
        />

        <Button mt={10} type="submit" loading={loading} fullWidth>
          {t('changePasswordButton')}
        </Button>
      </form>
    </div>
  );
}

export default AuthChangePassword;