"use client"
import { Box, Button, Container, Flex, Group, Title, Text, useMantineTheme } from '@mantine/core';
import classes from './Hero.module.css';
import Image from 'next/image';
import Link from 'next/link';
import { useMediaQuery } from '@mantine/hooks';
import { useTranslations } from 'next-intl'


export function LandingHero() {
  const theme = useMantineTheme()
  const isMobile = useMediaQuery(`(max-width: ${theme.breakpoints.sm})`);

  const t = useTranslations("LandingHeroComponent")

  return (
    <div className={classes.root}>
      <Container size="xxl">
        <Group justify="space-between">

          <div>
            <Text className={classes.title}>DuckVOD</Text>
            <Title c={theme.colors.gray[3]} mt={5} order={3}>{t('subtitle')}</Title>
            <Flex mt={10}>
              <Button
                variant="gradient"
                gradient={{ from: 'blue', to: 'purple' }}
                component={Link}
                href="/channels"
                className={classes.button}
              >
                {t('channelsButton')}
              </Button>
              <Button
                ml={10}
                variant="default"
                component={Link}
                href="/login"
              >
                {t('loginButton')}
              </Button>
            </Flex>
          </div>

          {!isMobile && (
            <Box>
              <Flex justify={"center"} align={"center"}>
                <div className={classes.logoBackground}></div>
                <Image src="/images/ganymede_logo.png" height={100} width={100} alt="Ganymede logo" className={classes.logo} />
              </Flex>
            </Box>
          )}
        </Group>
      </Container>
    </div>
  );
}