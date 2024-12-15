"use client"
import { Box, Button, Card, Center, Checkbox, Container, Divider, Drawer, Text, Title } from "@mantine/core";
import useAuthStore from "../store/useAuthStore";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import useSettingsStore from "../store/useSettingsStore";
import { useDisclosure } from "@mantine/hooks";
import AuthChangePassword from "../components/auth/ChangePassword";

const ProfilePage = () => {
  const { user, isLoading, isLoggedIn } = useAuthStore()
  const router = useRouter();

  const [passwordDrawerOpened, { open: openPasswordDrawer, close: closePasswordDrawer }] = useDisclosure(false);

  const {
    chatPlaybackSmoothScroll,
    setChatPlaybackSmoothScroll,
  } = useSettingsStore();

  // Handler to toggle the value
  const toggleSmoothScroll = () => {
    setChatPlaybackSmoothScroll(!chatPlaybackSmoothScroll);
  };

  useEffect(() => {
    if (!isLoading && !isLoggedIn) {
      router.push("/login");
    }
  }, [isLoggedIn, isLoading, router]);

  return (
    <div>
      <Container mt={15}>
        <Card withBorder p="xl" radius={"sm"}>
          <>
            <Center>
              <Title>{user?.username}</Title>
            </Center>
            <Center>
              <Text mr={5}>Role:</Text><Text>{user?.role}</Text>
            </Center>

            <Divider my={20} />

            <Title>Settings</Title>
            <Text size="sm">Settings are persisted in local browser storage.</Text>

            <Box mt={10}>
              <Checkbox
                label="Smooth chat scrolling"
                description="Enable smooth scrolling for new chat messages. May look bad if there is a large volume of messages per second."
                checked={chatPlaybackSmoothScroll}
                onChange={toggleSmoothScroll}
              />
            </Box>



            <Divider my={20} />

            {!user?.oauth && (
              <Button onClick={openPasswordDrawer} fullWidth>
                Change Password
              </Button>
            )}

          </>
        </Card>
      </Container>

      <Drawer opened={passwordDrawerOpened} onClose={closePasswordDrawer} position="right" title="Change Password">
        <AuthChangePassword handleClose={closePasswordDrawer} />
      </Drawer>

    </div >
  );
}

export default ProfilePage;