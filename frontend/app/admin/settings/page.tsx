"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText"
import { useAxiosPrivate } from "@/app/hooks/useAxios"
import { Config, NotificationType, ProxyListItem, useEditConfig, useGetConfig, useTestNotification } from "@/app/hooks/useConfig"
import { ActionIcon, Button, Card, Checkbox, Code, Collapse, Container, Flex, MultiSelect, NumberInput, Text, Textarea, TextInput, Title } from "@mantine/core"
import { useForm } from "@mantine/form"
import { useDisclosure } from "@mantine/hooks"
import Link from "next/link"
import { useEffect, useState } from "react"
import classes from "./AdminSettingsPage.module.css"
import { IconPlus, IconTrash } from "@tabler/icons-react"
import { Channel, useFetchChannels } from "@/app/hooks/useChannels"
import { showNotification } from "@mantine/notifications"

interface SelectOption {
  label: string;
  value: string;
}

const AdminSettingsPage = () => {
  useEffect(() => {
    document.title = "Admin - Settings";
  }, []);
  const [notificationsOpened, { toggle: toggleNotifications }] = useDisclosure(false);
  const [storageTemplateOpened, { toggle: toggleStorageTemplate }] = useDisclosure(false);
  const [channelSelect, setChannelSelect] = useState<SelectOption[]>([]);
  const axiosPrivate = useAxiosPrivate()

  const testNotificationMutate = useTestNotification()

  const editConfigMutate = useEditConfig()

  const { data, isPending, isError } = useGetConfig(axiosPrivate)

  const form = useForm({
    mode: "controlled",
    initialValues: {
      live_check_interval_seconds: data?.live_check_interval_seconds || 300,
      video_check_interval_minutes: data?.video_check_interval_minutes || 180,
      registration_enabled: data?.registration_enabled ?? true,
      parameters: {
        twitch_token: data?.parameters.twitch_token || "",
        video_convert: data?.parameters.video_convert || "",
        chat_render: data?.parameters.chat_render || "",
        streamlink_live: data?.parameters.streamlink_live || "",
      },
      archive: {
        save_as_hls: data?.archive.save_as_hls ?? false,
        generate_sprite_thumbnails: data?.archive.generate_sprite_thumbnails ?? true
      },
      notifications: {
        video_success_webhook_url: data?.notifications.video_success_webhook_url || "",
        video_success_template: data?.notifications.video_success_template || "",
        video_success_enabled: data?.notifications.video_success_enabled ?? false,
        live_success_webhook_url: data?.notifications.live_success_webhook_url || "",
        live_success_template: data?.notifications.live_success_template || "",
        live_success_enabled: data?.notifications.live_success_enabled ?? false,
        error_webhook_url: data?.notifications.error_webhook_url || "",
        error_template: data?.notifications.error_template || "",
        error_enabled: data?.notifications.error_enabled ?? false,
        is_live_webhook_url: data?.notifications.is_live_webhook_url || "",
        is_live_template: data?.notifications.is_live_template || "",
        is_live_enabled: data?.notifications.is_live_enabled ?? false,
      },
      storage_templates: {
        folder_template: data?.storage_templates.folder_template || "",
        file_template: data?.storage_templates.file_template || "",
      },
      livestream: {
        proxies: data?.livestream.proxies || [],
        proxy_enabled: data?.livestream.proxy_enabled ?? true,
        proxy_whitelist: data?.livestream.proxy_whitelist || [],
      }
    }
  })

  useEffect(() => {
    if (!data || !form) return

    form.setValues(data)
    form.resetDirty(data)

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data])

  const folderExample1 = "{{id}}-{{uuid}}";
  const folderExample2 = "{{date}}-{{title}}-{{type}}-{{id}}-{{uuid}}";
  const fileExample1 = "{{id}}";

  const { data: channels } = useFetchChannels();

  useEffect(() => {
    if (!channels) return;

    const transformedChannels: SelectOption[] = channels.map((channel: Channel) => ({
      label: channel.name,
      value: channel.name,
    }));

    setChannelSelect(transformedChannels);
  }, [channels]);

  const handleSubmitForm = async () => {
    try {
      const formValues = form.getValues()

      const submitConfig: Config = { ...formValues }

      await editConfigMutate.mutateAsync({
        axiosPrivate,
        config: submitConfig
      });

      showNotification({
        message: "Settings Saved",
        color: "green"
      });

    } catch (error) {
      console.error(error);
    }
  }

  if (isPending) return (
    <GanymedeLoadingText message="Loading settings" />
  )
  if (isError) return <div>Error loading settings</div>

  return (
    <div>
      <Container mt={15}>
        <Card withBorder p="xl" radius={"sm"}>
          <form onSubmit={form.onSubmit(() => {
            handleSubmitForm()
          })}>
            <Title>Settings</Title>
            <Text>Visit the <a className={classes.link} href="https://github.com/Zibbp/ganymede/wiki/Application-Settings" target="_blank">wiki</a> for documentation about each setting.</Text>

            <Title order={3}>Application</Title>
            <Checkbox
              mt={10}
              label="Registration Enabled"
              key={form.key('registration_enabled')}
              {...form.getInputProps('registration_enabled', { type: "checkbox" })}
            />

            <Button
              mt={15}
              onClick={toggleNotifications}
              fullWidth
              radius="md"
              size="md"
              variant="outline"
              color="orange"
            >
              Notification Settings
            </Button>

            <Collapse in={notificationsOpened}>
              <Text>Must be a webhook URL or an Apprise HTTP URL, visit the <a href="https://github.com/Zibbp/ganymede/wiki/Notifications" target="_blank">wiki</a> for more information.</Text>

              {/* video archive success */}
              <Title order={3}>Video Archive Success Notification</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label="Enabled"
                  key={form.key('notifications.video_success_enabled')}
                  {...form.getInputProps('notifications.video_success_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.VideoSuccess })}>
                  Test
                </Button>
              </Flex>
              <TextInput
                label="Webhook URL"
                placeholder="https://webhook.curl"
                key={form.key('notifications.video_success_webhook_url')}
                {...form.getInputProps('notifications.video_success_webhook_url')}
              />
              <Textarea
                label="Template"
                placeholder=""
                key={form.key('notifications.video_success_template')}
                {...form.getInputProps('notifications.video_success_template')}
              />

              <Text>Available variables to use in the template:</Text>
              <div>
                <Text>Channel</Text>
                <Code>
                  {"{{channel_id}} {{channel_ext_id}} {{channel_display_name}}"}
                </Code>
                <Text>Video</Text>
                <Code>
                  {
                    "{{vod_id}} {{vod_ext_id}} {{vod_platform}} {{vod_type}} {{vod_title}} {{vod_duration}} {{vod_views}} {{vod_resolution}} {{vod_streamed_at}} {{vod_created_at}}"
                  }
                </Code>
                <Text>Queue</Text>
                <Code>{"{{queue_id}} {{queue_created_at}}"}</Code>
              </div>

              {/* live archive success */}
              <Title order={3}>Live Archive Success Notification</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label="Enabled"
                  key={form.key('notifications.live_success_enabled')}
                  {...form.getInputProps('notifications.live_success_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.LiveSuccess })}>
                  Test
                </Button>
              </Flex>
              <TextInput
                label="Webhook URL"
                placeholder="https://webhook.curl"
                key={form.key('notifications.live_success_webhook_url')}
                {...form.getInputProps('notifications.live_success_webhook_url')}
              />
              <Textarea
                label="Template"
                placeholder=""
                key={form.key('notifications.live_success_template')}
                {...form.getInputProps('notifications.live_success_template')}
              />

              <Text>Available variables to use in the template:</Text>
              <div>
                <Text>Channel</Text>
                <Code>
                  {"{{channel_id}} {{channel_ext_id}} {{channel_display_name}}"}
                </Code>
                <Text>Video</Text>
                <Code>
                  {
                    "{{vod_id}} {{vod_ext_id}} {{vod_platform}} {{vod_type}} {{vod_title}} {{vod_duration}} {{vod_views}} {{vod_resolution}} {{vod_streamed_at}} {{vod_created_at}}"
                  }
                </Code>
                <Text>Queue</Text>
                <Code>{"{{queue_id}} {{queue_created_at}}"}</Code>
              </div>

              {/* is live */}
              <Title order={3}>Channel Is Live Notification</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label="Enabled"
                  key={form.key('notifications.is_live_enabled')}
                  {...form.getInputProps('notifications.is_live_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.IsLive })}>
                  Test
                </Button>
              </Flex>
              <TextInput
                label="Webhook URL"
                placeholder="https://webhook.curl"
                key={form.key('notifications.is_live_webhook_url')}
                {...form.getInputProps('notifications.is_live_webhook_url')}
              />
              <Textarea
                label="Template"
                placeholder=""
                key={form.key('notifications.is_live_template')}
                {...form.getInputProps('notifications.is_live_template')}
              />

              <Text>Available variables to use in the template:</Text>
              <div>
                <Text>Channel</Text>
                <Code>
                  {"{{channel_id}} {{channel_ext_id}} {{channel_display_name}}"}
                </Code>
                <Text>Video</Text>
                <Code>
                  {
                    "{{vod_id}} {{vod_ext_id}} {{vod_platform}} {{vod_type}} {{vod_title}} {{vod_duration}} {{vod_views}} {{vod_resolution}} {{vod_streamed_at}} {{vod_created_at}}"
                  }
                </Code>
                <Text>Queue</Text>
                <Code>{"{{queue_id}} {{queue_created_at}}"}</Code>
              </div>

              {/* error */}
              <Title order={3}>Error Notification</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label="Enabled"
                  key={form.key('notifications.error_enabled')}
                  {...form.getInputProps('notifications.error_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.Error })}>
                  Test
                </Button>
              </Flex>
              <TextInput
                label="Webhook URL"
                placeholder="https://webhook.curl"
                key={form.key('notifications.error_webhook_url')}
                {...form.getInputProps('notifications.error_webhook_url')}
              />
              <Textarea
                label="Template"
                placeholder=""
                key={form.key('notifications.error_template')}
                {...form.getInputProps('notifications.error_template')}
              />

              <Text>Available variables to use in the template:</Text>
              <div>
                <Text>Task</Text>
                <Code>{"{{failed_task}}"}</Code>
                <Text>Channel</Text>
                <Code>
                  {"{{channel_id}} {{channel_ext_id}} {{channel_display_name}}"}
                </Code>
                <Text>Video</Text>
                <Code>
                  {
                    "{{vod_id}} {{vod_ext_id}} {{vod_platform}} {{vod_type}} {{vod_title}} {{vod_duration}} {{vod_views}} {{vod_resolution}} {{vod_streamed_at}} {{vod_created_at}}"
                  }
                </Code>
                <Text>Queue</Text>
                <Code>{"{{queue_id}} {{queue_created_at}}"}</Code>
              </div>

            </Collapse>

            <Title mt={10} order={3}>Archive</Title>

            <NumberInput
              label="Live Stream Check Interval Seconds"
              description="How often watched channels are checked for live streams in seconds. REQUIRES RESTART!"
              placeholder="300"
              key={form.key('live_check_interval_seconds')}
              {...form.getInputProps('live_check_interval_seconds')}
              min={5}
            />

            <NumberInput
              mt={10}
              label="Video Check Interval Minutes"
              description="How often watched channels are checked for videos in minutes. REQUIRES RESTART!"
              placeholder="180"
              key={form.key('video_check_interval_minutes')}
              {...form.getInputProps('video_check_interval_minutes')}
              min={1}
            />


            <Checkbox
              mt={15}
              label="Convert MP4 to HLS"
              key={form.key('archive.save_as_hls')}
              {...form.getInputProps('archive.save_as_hls', { type: "checkbox" })}
              mr={15}
            />

            <Checkbox
              mt={15}
              label="Generate Sprite Thumbnails"
              description="Preview thumbnail when scrubbing a video's timeline. These are generated after the video is archived."
              key={form.key('archive.generate_sprite_thumbnails')}
              {...form.getInputProps('archive.generate_sprite_thumbnails', { type: "checkbox" })}
              mr={15}
            />

            <Button
              mt={15}
              onClick={toggleStorageTemplate}
              fullWidth
              radius="md"
              size="md"
              variant="outline"
              color="orange"
            >
              Storage Template Settings
            </Button>

            <Collapse in={storageTemplateOpened}>

              <div>
                <Text mb={10}>
                  Customize how folders and files are named. This only applies to new
                  files. To apply to existing files execute the migration task on the{" "}
                  <Link className={classes.link} href="/admin/tasks">
                    tasks
                  </Link>{" "}
                  page.
                </Text>
                <div>
                  <Title order={4}>Folder Template</Title>

                  <Textarea
                    description="{{uuid}} is required to be present for the folder template."
                    key={form.key('storage_templates.folder_template')}
                    {...form.getInputProps('storage_templates.folder_template')}
                    required
                  />
                </div>

                <div>
                  <Title mt={5} order={4}>
                    File Template
                  </Title>

                  <Textarea
                    description="Do not include the file extension. The file type will be appened to the end of the file name such as -video -chat and -thumbnail."
                    key={form.key('storage_templates.file_template')}
                    {...form.getInputProps('storage_templates.file_template')}
                    required
                  />
                </div>

                <div>
                  <Title mt={5} order={4}>
                    Available Variables
                  </Title>

                  <div>
                    <Text>Ganymede</Text>
                    <Code>{"{{uuid}}"}</Code>
                    <Text>Twitch Video</Text>
                    <Code>{"{{id}} {{channel}} {{title}} {{date}} {{type}}"}</Code>
                    <Text ml={20} mt={5} size="sm">
                      ID: Twitch video ID <br /> Date: Date streamed or uploaded <br />{" "}
                      Type: Twitch video type (live, archive, highlight)
                    </Text>
                  </div>
                </div>

                <div>
                  <Title mt={5} order={4}>
                    Examples
                  </Title>

                  <Text>Folder</Text>
                  <Code block>{folderExample1}</Code>
                  <Code block>{folderExample2}</Code>
                  <Text>File</Text>
                  <Code block>{fileExample1}</Code>
                </div>


              </div>

            </Collapse>

            <Title mt={10} order={3}>Video</Title>

            <TextInput
              label="Twitch Token"
              description="Supply your Twitch token for downloading ad-free livestreams and subscriber-only videos."
              key={form.key('parameters.twitch_token')}
              {...form.getInputProps('parameters.twitch_token')}
            />

            <TextInput
              label="Video Convert FFmpeg Arguments"
              description="Post-download video processing FFmpeg arguments."
              key={form.key('parameters.video_convert')}
              {...form.getInputProps('parameters.video_convert')}
            />

            <Title mt={10} order={3}>Live Stream</Title>

            <TextInput
              label="Streamlink Parameters"
              description="For live streams. Must be comma separated."
              key={form.key('parameters.streamlink_live')}
              {...form.getInputProps('parameters.streamlink_live')}
            />

            <Title mt={5} order={5}>Proxy</Title>
            <Text>Archive live streams through a proxy to prevent ads. Your Twitch token <b>is not sent</b> to the proxy.</Text>

            <Checkbox
              mt={10}
              label="Enable proxy"
              key={form.key('livestream.proxy_enabled')}
              {...form.getInputProps('livestream.proxy_enabled', { type: "checkbox" })}
              mr={15}
            />

            <div>
              {form.values.livestream.proxies && form.values.livestream.proxies.map((proxy: ProxyListItem, index) => (
                <div key={index}>
                  <div key={index} className={classes.proxyList}>
                    <TextInput
                      className={classes.proxyInput}
                      placeholder="https://proxy.url"
                      label="Proxy URL"
                      key={form.key(`livestream.proxies.${index}.url`)}
                      {...form.getInputProps(`livestream.proxies.${index}.url`)}
                    />
                    <TextInput
                      label="Header"
                      className={classes.proxyInput}
                      key={form.key(`livestream.proxies.${index}.header`)}
                      {...form.getInputProps(`livestream.proxies.${index}.header`)}
                    />
                    <ActionIcon
                      color="red"
                      size="lg"
                      mt={20}
                      onClick={() => form.removeListItem('livestream.proxies', index)}
                    >
                      <IconTrash size="1.125rem" />
                    </ActionIcon>
                  </div>
                </div>
              ))}
            </div>
            <Button
              onClick={() =>
                form.insertListItem('livestream.proxies', { url: '', header: '' })
              }
              mt={10}
              leftSection={<IconPlus size="1rem" />}
            >
              Add
            </Button>

            <MultiSelect
              label="Whitelist Channels"
              description="Select channels that are excluded from using the proxy if enabled. Instead your Twitch token will be used. Select channels that you are subscribed to."
              data={channelSelect}
              key={form.key('livestream.proxy_whitelist')}
              {...form.getInputProps('livestream.proxy_whitelist')}
              searchable
            />

            <Title mt={10} order={3}>Chat</Title>

            <TextInput
              label="Chat Render Arguments"
              description="TwitchDownloaderCLI chat render arguments."
              key={form.key('parameters.chat_render')}
              {...form.getInputProps('parameters.chat_render')}
            />

            <Button
              mt={15}
              type="submit"
              fullWidth
              loading={editConfigMutate.isPending}
            >
              Save Settings
            </Button>

          </form>

        </Card>

      </Container>
    </div>
  );
}

export default AdminSettingsPage;