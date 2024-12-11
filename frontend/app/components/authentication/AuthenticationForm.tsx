'use client'
import { useToggle, upperFirst } from '@mantine/hooks';
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
import { authLogin } from '@/app/hooks/useAuthentication';
import { useRouter } from 'next/navigation';
import useAuthStore from '@/app/store/useAuthStore';
import { env } from 'next-runtime-env';

const schema = z.object({
  username: z.string().min(3, { message: "Username should have at least 3 characters" }),
  password: z.string().min(8, { message: "Password should have at least 8 characters" })
})



export function AuthenticationForm() {
  const router = useRouter()
  const { fetchUser } = useAuthStore();
  const [type, toggle] = useToggle(['login', 'register']);
  const form = useForm({
    mode: "uncontrolled",
    initialValues: {
      username: '',
      password: '',
    },

    validate: zodResolver(schema),
  });

  const handleLogin = async (username: string, password: string) => {
    try {
      await authLogin(username, password)
      await fetchUser()
      router.push('/')
    } catch (error) {
      console.error(`Error logging in: ${error instanceof Error ? error.message : String(error)}`)
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

      <form onSubmit={form.onSubmit((values) => handleLogin(values.username, values.password))}>
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
          <Anchor component="button" type="button" c="dimmed" onClick={() => toggle()} size="xs">
            {type === 'register'
              ? 'Already have an account? Login'
              : "Don't have an account? Register"}
          </Anchor>
          <Button type="submit" radius="xl">
            {upperFirst(type)}
          </Button>
        </Group>
      </form>
    </Paper>
  );
}