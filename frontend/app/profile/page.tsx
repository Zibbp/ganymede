"use client"
import { Box, Button, Card, Center, Checkbox, Container, Divider, Drawer, Text, Title } from "@mantine/core";
import useAuthStore from "../store/useAuthStore";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import useSettingsStore from "../store/useSettingsStore";
import { useDisclosure } from "@mantine/hooks";
import AuthChangePassword from "../components/auth/ChangePassword";
import { useTranslations } from "next-intl";
import { usePageTitle } from "../util/util";

const ProfilePage = () => {
  const t = useTranslations("ProfilePage");
  usePageTitle(t('title'));
  const { user, isLoading, isLoggedIn } = useAuthStore()
  const router = useRouter();

  const [passwordDrawerOpened, { open: openPasswordDrawer, close: closePasswordDrawer }] = useDisclosure(false);

  const {
    chatPlaybackSmoothScroll,
    setChatPlaybackSmoothScroll,
    showChatHistogram,
    setShowChatHistogram,
    showProcessingVideosInRecentlyArchived,
    setShowProcessingVideosInRecentlyArchived
  } = useSettingsStore();

  const toggleSmoothScroll = () => {
    setChatPlaybackSmoothScroll(!chatPlaybackSmoothScroll);
  };

  const toggleChatHistogram = () => {
    setShowChatHistogram(!showChatHistogram);
  }

  const toggleProcessingVideosInRecentlyArchived = () => {
    setShowProcessingVideosInRecentlyArchived(!showProcessingVideosInRecentlyArchived);
  }

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
              <Text mr={5}>{t('role')}:</Text><Text>{user?.role}</Text>
            </Center>

            <Divider my={20} />

            <Title>{t('settings.title')}</Title>
            <Text size="sm">{t('settings.description')}</Text>

            <Box mt={10}>
              <Checkbox
                label={t('settings.smoothScroll')}
                description={t('settings.smoothScrollDescription')}
                checked={chatPlaybackSmoothScroll}
                onChange={toggleSmoothScroll}
                my={5}
              />
              <Checkbox
                label={t('settings.chatHistogram')}
                description={t('settings.chatHistogramDescription')}
                checked={showChatHistogram}
                onChange={toggleChatHistogram}
                my={5}
              />
              <Checkbox
                label={t('settings.showProcessingVideos')}
                description={t('settings.showProcessingVideosDescription')}
                checked={showProcessingVideosInRecentlyArchived}
                onChange={toggleProcessingVideosInRecentlyArchived}
                my={5}
              />
            </Box>



            <Divider my={20} />

            {!user?.oauth && (
              <Button onClick={openPasswordDrawer} fullWidth>
                {t('changePassword.title')}
              </Button>
            )}

          </>
        </Card>
      </Container>

      <Drawer opened={passwordDrawerOpened} onClose={closePasswordDrawer} position="right" title={t('changePassword.title')}>
        <AuthChangePassword handleClose={closePasswordDrawer} />
      </Drawer>

    </div >
  );
}

export default ProfilePage;