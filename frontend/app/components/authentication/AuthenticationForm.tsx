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

const schema = z.object({
  username: z.string().min(3, { message: "Username should have at least 3 characters" }),
  password: z.string().min(8, { message: "Password should have at least 8 characters" })
})

export enum AuthFormType {
  Login = "login",
  Register = "register"
}

type Props = {
  type: AuthFormType
}

export function AuthenticationForm({ type }: Props) {
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
          message: "Logged in"
        })
        router.push('/')
      } catch (error) {
        console.error(`Error logging in: ${error instanceof Error ? error.message : String(error)}`)
      } finally {
        setLoading(false)
      }
    } else {
      try {
        setLoading(true)
        await authRegister(username, password)
        router.push('/login')
        showNotification({
          message: "Successfully registered, please sign in."
        })
      } catch (error) {
        console.error(`Error registering in: ${error instanceof Error ? error.message : String(error)}`)
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
        Welcome to Ganymede, {type} with
      </Text>

      {(env("NEXT_PUBLIC_SHOW_SSO_LOGIN_BUTTON") == "true") ? (
        <Box>
          <Group grow mb="md" mt="md">
            <Button fullWidth onClick={handleSSOLogin}>Single Sign-On (SSO)</Button>
          </Group>

          <Divider label="Or continue with local credentials" labelPosition="center" my="lg" />
        </Box>
      ) : (
        <Box mt="md" mb="md"></Box>
      )}

      <form onSubmit={form.onSubmit((values) => handleAuth(values.username, values.password))}>
        <Stack>
          <TextInput
            label="Username"
            placeholder="Your username"
            key={form.key('username')}
            {...form.getInputProps('username')}
            radius="md"
          />

          <PasswordInput
            label="Password"
            placeholder="Your password"
            key={form.key('password')}
            {...form.getInputProps('password')}
            radius="md"
          />
        </Stack>

        <Group justify="space-between" mt="xl">

          {(type == AuthFormType.Login) && (
            <Anchor component={Link} href="/register" type="button" c="dimmed" size="xs">
              {"Don't have an account? Register"}
            </Anchor>
          )}
          {(type == AuthFormType.Register) && (
            <Anchor component={Link} href="/login" type="button" c="dimmed" size="xs">
              {"Already have an account? Login"}
            </Anchor>
          )}
          <Button type="submit" loading={loading}>
            {upperFirst(type)}
          </Button>
        </Group>
      </form>
    </Paper>
  );
}