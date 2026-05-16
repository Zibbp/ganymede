"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useGetGanymedeInformation } from "@/app/hooks/useAdmin";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Card, Container, Flex, Title, Text, Code } from "@mantine/core";
import Link from "next/link";
import classes from "./AdminInformationPage.module.css"
import { useEffect } from "react";
import { useTranslations } from "next-intl";
import { usePageTitle } from "@/app/util/util";

const AdminInformationPage = () => {
  const t = useTranslations("AdminInformationPage");
  usePageTitle(t('title'));
  const axiosPrivate = useAxiosPrivate()

  const { data, isPending, isError } = useGetGanymedeInformation(axiosPrivate)

  if (isPending) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isError) return <div>{t('error')}</div>


  return (
    <div>
      <Container mt={15}>
        <Card withBorder p="xl" radius={"sm"}>
          {/* server info */}
          <div>
            <Title>
              {t('server.title')}
            </Title>
            <Flex>
              <Text>{t('server.commit')}:</Text>
              <Code ml={5}>{data.commit_hash}</Code>
            </Flex>
            <Flex>
              <Text>{t('server.tag')}:</Text>
              <Code ml={5}>{data.tag}</Code>
            </Flex>
            <Flex>
              <Text>{t('server.buildDate')}:</Text>
              <Code ml={5}>{data.build_time}</Code>
            </Flex>
            <Flex>
              <Text>{t('server.uptime')}:</Text>
              <Code ml={5}>{data.uptime}</Code>
            </Flex>
          </div>
          {/* package info */}
          <div>
            <Title>
              {t('packageVersions')}
            </Title>
            <Flex>
              <Text component={Link} href="https://github.com/lay295/TwitchDownloader" target="_blank" className={classes.link}>TwitchDownloader:</Text>
              <Code ml={5}>{data.program_versions.twitch_downloader}</Code>
            </Flex>
            <Flex>
              <Text component={Link} href="https://github.com/yt-dlp/yt-dlp" target="_blank" className={classes.link}>yt-dlp:</Text>
              <Code ml={5}>{data.program_versions.yt_dlp}</Code>
            </Flex>
            <div>
              <Text component={Link} href="https://www.ffmpeg.org/" target="_blank" className={classes.link}>FFmpeg:</Text>
              <Code ml={5} block>{data.program_versions.ffmpeg}</Code>
            </div>
          </div>
        </Card>
      </Container>
    </div>
  );
}

export default AdminInformationPage;