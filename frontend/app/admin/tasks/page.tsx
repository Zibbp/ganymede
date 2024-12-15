"use client"
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Card, Container, Title, Text, Tooltip, ActionIcon, Group, Box } from "@mantine/core";
import classes from "./AdminTasksPage.module.css"
import { useEffect, useState } from "react";
import { IconPlayerPlay } from "@tabler/icons-react";
import { Task, useStartTask } from "@/app/hooks/useTasks";
import { showNotification } from "@mantine/notifications";

const AdminTasksPage = () => {
  useEffect(() => {
    document.title = "Admin - Tasks";
  }, []);
  const axiosPrivate = useAxiosPrivate()
  const [loading, setLoading] = useState(false)

  const startTaskMutate = useStartTask()

  const startTask = async (task: Task) => {
    try {
      setLoading(true)

      await startTaskMutate.mutateAsync({
        axiosPrivate: axiosPrivate,
        task: task
      })

      showNotification({
        message: "Task started, see container logs for more information."
      })
    } catch (error) {
      console.error(error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Container mt={15}>
        <Card withBorder p="xl" radius={"sm"}>

          <Title>Tasks</Title>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Check watched channels for live streams to archive</Text>
              <Text size="xs">Occurs at interval set in the config.</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.CheckLive)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Check watched channels for videos to archive</Text>
              <Text size="xs">Occurs at interval set in the config.</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.CheckVod)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Check watched channels for clips to archive</Text>
              <Text size="xs">Occurs daily at 00:00.</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.CheckClips)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Storage Template Migration</Text>
              <Text size="xs">Apply storage template to existing files. Read the <a className={classes.link} target="_blank" href="https://github.com/Zibbp/ganymede/wiki/Storage-Templates-and-Migration">docs</a> before starting.</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.StorageMigration)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Prune Videos</Text>
              <Text size="xs">Prune videos from channels that have retention settings configured.</Text>
              <Text size="xs">Occurs daily at 00:00.</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.PruneVideo)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Get JSON Web Key Sets (JWKS) From SSO Provider</Text>
              <Text size="xs">Occurs daily at 00:00.</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.GetJWKS)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Save Chapters for Twitch Videos</Text>
              <Text size="xs">Save chapters for already archived Twitch videos (automatically does this for new archives).</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.SaveChapters)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>

          <Group justify="space-between" py={5}>
            <Box>
              <Text fw={"bold"}>Update Live Stream Archives with Video IDs</Text>
              <Text size="xs">Attempt to update live stream archives with their corresponding video ID (automatically does this after live archive finishes).</Text>
            </Box>
            <Tooltip label="Start Task">
              <ActionIcon
                onClick={() => startTask(Task.UpdateStreamVodIds)}
                loading={loading}
                color="green"
                variant="filled"
                size="lg"
              >
                <IconPlayerPlay size={24} />
              </ActionIcon>
            </Tooltip>
          </Group>


        </Card>
      </Container>
    </div>
  );
}

export default AdminTasksPage;