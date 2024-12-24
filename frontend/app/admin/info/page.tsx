"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useGetGanymedeInformation } from "@/app/hooks/useAdmin";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Card, Container, Flex, Title, Text, Code } from "@mantine/core";
import Link from "next/link";
import classes from "./AdminInformationPage.module.css"
import { useEffect } from "react";

const AdminInformationPage = () => {
  useEffect(() => {
    document.title = "Admin - Info";
  }, []);
  const axiosPrivate = useAxiosPrivate()

  const { data, isPending, isError } = useGetGanymedeInformation(axiosPrivate)

  if (isPending) return (
    <GanymedeLoadingText message="Loading Ganymde Information" />
  )
  if (isError) return <div>Error loading Ganymede information</div>


  return (
    <div>
      <Container mt={15}>
        <Card withBorder p="xl" radius={"sm"}>
          {/* server info */}
          <div>
            <Title>
              Server
            </Title>
            <Flex>
              <Text>Commit:</Text>
              <Code ml={5}>{data.commit_hash}</Code>
            </Flex>
            <Flex>
              <Text>Build Date:</Text>
              <Code ml={5}>{data.build_time}</Code>
            </Flex>
            <Flex>
              <Text>Uptime:</Text>
              <Code ml={5}>{data.uptime}</Code>
            </Flex>
          </div>
          {/* package info */}
          <div>
            <Title>
              Package Versions
            </Title>
            <Flex>
              <Text component={Link} href="https://github.com/lay295/TwitchDownloader" target="_blank" className={classes.link}>TwitchDownloader:</Text>
              <Code ml={5}>{data.program_versions.twitch_downloader}</Code>
            </Flex>
            <Flex>
              <Text component={Link} href="https://github.com/xenova/chat-downloader" target="_blank" className={classes.link}>Chat-Downloader:</Text>
              <Code ml={5}>{data.program_versions.chat_downloader}</Code>
            </Flex>
            <Flex>
              <Text component={Link} href="https://github.com/streamlink/streamlink" target="_blank" className={classes.link}>Streamlink:</Text>
              <Code ml={5}>{data.program_versions.streamlink}</Code>
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