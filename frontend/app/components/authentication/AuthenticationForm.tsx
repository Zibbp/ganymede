'use client'
import { upperFirst } from '@mantine/hooks';
import { useForm, zodResolver } from '@mantine/form';
import {
  TextInput,
  PasswordInput,
  Text,
  Paper,
  Group,
  Button,
  Divider,
  Anchor,
  Stack,
  Box,
} from '@mantine/core';
import { z } from 'zod';
import { authLogin, authRegister } from '@/app/hooks/useAuthentication';
import { useRouter } from 'next/navigation';
import useAuthStore from '@/app/store/useAuthStore';
import { env } from 'next-runtime-env';
import { showNotification } from '@mantine/notifications';
import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';

export enum AuthFormType {
  Login = "login",
  Register = "register"
}

type Props = {
  type: AuthFormType
}

export function AuthenticationForm({ type }: Props) {
  const t = useTranslations('AuthComponents')

  const schema = z.object({
    username: z.string().min(3, { message: t('usernameTooShort') }),
    password: z.string().min(8, { message: t('passwordTooShort') })
  })


  const [loading, setLoading] = useState(false)
  const router = useRouter()
  const { fetchUser } = useAuthStore();
  const form = useForm({
    mode: "uncontrolled",
    initialValues: {
      username: '',
      password: '',
    },

    validate: zodResolver(schema),
  });

  // If force SSO is enabled redirect immediately
  useEffect(() => {
    if (env('NEXT_PUBLIC_FORCE_SSO_AUTH') == "true") {
      handleSSOLogin()
    }
  }, [])

  const handleAuth = async (username: string, password: string) => {
    if (type == AuthFormType.Login) {
      try {
        setLoading(true)
        await authLogin(username, password)
        await fetchUser()
        showNotification({
          message: t('loggedInNotification')
        })
        router.push('/')
      } catch (error) {
        console.error(`${t('errorLoggingInNotification')}: ${error instanceof Error ? error.message : String(error)}`)
      } finally {
        setLoading(false)
      }
    } else {
      try {
        setLoading(true)
        await authRegister(username, password)
        router.push('/login')
        showNotification({
          message: t('registerSuccessNotification')
        })
      } catch (error) {
        console.error(`${t('errorRegisteringNotification')}: ${error instanceof Error ? error.message : String(error)}`)
      } finally {
        setLoading(false)
      }
    }
  }

  const handleSSOLogin = async () => {
    window.location.href = `${(env('NEXT_PUBLIC_API_URL') ?? '')}/api/v1/auth/oauth/login`
  }

  return (
    <Paper radius="md" p="xl" withBorder>
      <Text size="lg" fw={500}>
        {t('welcomeText')}, {t(type)} {t('with')}
      </Text>

      {(env("NEXT_PUBLIC_SHOW_SSO_LOGIN_BUTTON") == "true") ? (
        <Box>
          <Group grow mb="md" mt="md">
            <Button fullWidth onClick={handleSSOLogin}>{t('ssoButton')}</Button>
          </Group>

          <Divider label={t('localCredentialsLabel')} labelPosition="center" my="lg" />
        </Box>
      ) : (
        <Box mt="md" mb="md"></Box>
      )}

      <form onSubmit={form.onSubmit((values) => handleAuth(values.username, values.password))}>
        <Stack>
          <TextInput
            label={t('usernameLabel')}
            placeholder={t('usernameDescription')}
            key={form.key('username')}
            {...form.getInputProps('username')}
            radius="md"
          />

          <PasswordInput
            label={t('passwordLabel')}
            placeholder={t('passwordDescription')}
            key={form.key('password')}
            {...form.getInputProps('password')}
            radius="md"
          />
        </Stack>

        <Group justify="space-between" mt="xl">

          {(type == AuthFormType.Login) && (
            <Anchor component={Link} href="/register" type="button" c="dimmed" size="xs">
              {t('registerLinkText')}
            </Anchor>
          )}
          {(type == AuthFormType.Register) && (
            <Anchor component={Link} href="/login" type="button" c="dimmed" size="xs">
              {t('loginLinkText')}
            </Anchor>
          )}
          <Button type="submit" loading={loading}>
            {upperFirst(t(type))}
          </Button>
        </Group>
      </form>
    </Paper>
  );
}