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
import { useTranslations } from "next-intl"
import { usePageTitle } from "@/app/util/util"

interface SelectOption {
  label: string;
  value: string;
}

const AdminSettingsPage = () => {
  const t = useTranslations('AdminSettingsPage');
  usePageTitle(t('title'))
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
        message: t('saveSuccess'),
        color: "green"
      });

    } catch (error) {
      console.error(error);
    }
  }

  if (isPending) return (
    <GanymedeLoadingText message={t('loading')} />
  )
  if (isError) return <div>{t('error')}</div>

  return (
    <div>
      <Container mt={15}>
        <Card withBorder p="xl" radius={"sm"}>
          <form onSubmit={form.onSubmit(() => {
            handleSubmitForm()
          })}>
            <Title>{t('header')}</Title>
            <Text>{t('headerDescription.part1')} <a className={classes.link} href="https://github.com/Zibbp/ganymede/wiki/Application-Settings" target="_blank">{t('headerDescription.part2')}</a> {t('headerDescription.part3')}</Text>

            <Title order={3}>{t('applicationSettings.header')}</Title>
            <Checkbox
              mt={10}
              label={t('applicationSettings.registrationEnabledLabel')}
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
              {t('applicationSettings.notificationSettingsButton')}
            </Button>

            <Collapse in={notificationsOpened}>
              <Text>Must be a webhook URL or an Apprise HTTP URL, visit the <a href="https://github.com/Zibbp/ganymede/wiki/Notifications" target="_blank">wiki</a> for more information.</Text>

              {/* video archive success */}
              <Title order={3}>{t('applicationSettings.videoArchiveSuccessNotification')}</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label={t('applicationSettings.enabledLabel')}
                  key={form.key('notifications.video_success_enabled')}
                  {...form.getInputProps('notifications.video_success_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.VideoSuccess })}>
                  {t('applicationSettings.testButton')}
                </Button>
              </Flex>
              <TextInput
                label={t('applicationSettings.webhookUrlLabel')}
                placeholder="https://webhook.curl"
                key={form.key('notifications.video_success_webhook_url')}
                {...form.getInputProps('notifications.video_success_webhook_url')}
              />
              <Textarea
                label={t('applicationSettings.templateLabel')}
                placeholder=""
                key={form.key('notifications.video_success_template')}
                {...form.getInputProps('notifications.video_success_template')}
              />

              <Text>{t('applicationSettings.availableVariables')}:</Text>
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
              <Title order={3}>{t('applicationSettings.liveArchiveSuccessNotification')}</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label={t('applicationSettings.enabledLabel')}
                  key={form.key('notifications.live_success_enabled')}
                  {...form.getInputProps('notifications.live_success_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.LiveSuccess })}>
                  {t('applicationSettings.testButton')}
                </Button>
              </Flex>
              <TextInput
                label={t('applicationSettings.webhookUrlLabel')}
                placeholder="https://webhook.curl"
                key={form.key('notifications.live_success_webhook_url')}
                {...form.getInputProps('notifications.live_success_webhook_url')}
              />
              <Textarea
                label={t('applicationSettings.templateLabel')}
                placeholder=""
                key={form.key('notifications.live_success_template')}
                {...form.getInputProps('notifications.live_success_template')}
              />

              <Text>{t('applicationSettings.availableVariables')}:</Text>
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
              <Title order={3}>{t('applicationSettings.channelIsLiveNotification')}</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label={t('applicationSettings.enabledLabel')}
                  key={form.key('notifications.is_live_enabled')}
                  {...form.getInputProps('notifications.is_live_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.IsLive })}>
                  {t('applicationSettings.testButton')}
                </Button>
              </Flex>
              <TextInput
                label={t('applicationSettings.webhookUrlLabel')}
                placeholder="https://webhook.curl"
                key={form.key('notifications.is_live_webhook_url')}
                {...form.getInputProps('notifications.is_live_webhook_url')}
              />
              <Textarea
                label={t('applicationSettings.templateLabel')}
                placeholder=""
                key={form.key('notifications.is_live_template')}
                {...form.getInputProps('notifications.is_live_template')}
              />

              <Text>{t('applicationSettings.availableVariables')}:</Text>
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
                <Text>Live Stream</Text>
                <Code>
                  {
                    "{{category}}"
                  }
                </Code>
                <Text>Queue</Text>
                <Code>{"{{queue_id}} {{queue_created_at}}"}</Code>
              </div>

              {/* error */}
              <Title order={3}>{t('applicationSettings.errorNotification')}</Title>
              <Flex>
                <Checkbox
                  mt={10}
                  label={t('applicationSettings.enabledLabel')}
                  key={form.key('notifications.error_enabled')}
                  {...form.getInputProps('notifications.error_enabled', { type: "checkbox" })}
                  mr={15}
                />
                <Button variant="outline" color="violet"
                  onClick={() => testNotificationMutate.mutate({ axiosPrivate, type: NotificationType.Error })}>
                  {t('applicationSettings.testButton')}
                </Button>
              </Flex>
              <TextInput
                label={t('applicationSettings.webhookUrlLabel')}
                placeholder="https://webhook.curl"
                key={form.key('notifications.error_webhook_url')}
                {...form.getInputProps('notifications.error_webhook_url')}
              />
              <Textarea
                label={t('applicationSettings.templateLabel')}
                placeholder=""
                key={form.key('notifications.error_template')}
                {...form.getInputProps('notifications.error_template')}
              />

              <Text>{t('applicationSettings.availableVariables')}:</Text>
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

            <Title mt={10} order={3}>{t('archiveSettings.header')}</Title>

            <NumberInput
              label={t('archiveSettings.livestreamCheckIntervalLabel')}
              description={t('archiveSettings.livestreamCheckIntervalDescription')}
              placeholder="300"
              key={form.key('live_check_interval_seconds')}
              {...form.getInputProps('live_check_interval_seconds')}
              min={5}
            />

            <NumberInput
              mt={10}
              label={t('archiveSettings.videoCheckIntervalLabel')}
              description={t('archiveSettings.videoCheckIntervalDescription')}
              placeholder="180"
              key={form.key('video_check_interval_minutes')}
              {...form.getInputProps('video_check_interval_minutes')}
              min={1}
            />


            <Checkbox
              mt={15}
              label={t('archiveSettings.mp4ToHLSConversionLabel')}
              key={form.key('archive.save_as_hls')}
              {...form.getInputProps('archive.save_as_hls', { type: "checkbox" })}
              mr={15}
            />

            <Checkbox
              mt={15}
              label={t('archiveSettings.generateSpriteThumbnailsLabel')}
              description={t('archiveSettings.generateSpriteThumbnailsDescription')}
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
              {t('archiveSettings.storageTemplateSettings')}
            </Button>

            <Collapse in={storageTemplateOpened}>

              <div>
                <Text mb={10}>
                  {t('archiveSettings.storageTemplateSettingsDescription')}
                </Text>
                <div>
                  <Title order={4}>{t('archiveSettings.folderTemplateText')}</Title>

                  <Textarea
                    description="{{uuid}} is required to be present for the folder template."
                    key={form.key('storage_templates.folder_template')}
                    {...form.getInputProps('storage_templates.folder_template')}
                    required
                  />
                </div>

                <div>
                  <Title mt={5} order={4}>
                    {t('archiveSettings.fileTemplateText')}
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
                    {t('archiveSettings.examples')}
                  </Title>

                  <Text>Folder</Text>
                  <Code block>{folderExample1}</Code>
                  <Code block>{folderExample2}</Code>
                  <Text>File</Text>
                  <Code block>{fileExample1}</Code>
                </div>


              </div>

            </Collapse>

            <Title mt={10} order={3}>{t('videoSettings.header')}</Title>

            <TextInput
              label={t('videoSettings.twitchTokenLabel')}
              description={t('videoSettings.twitchTokenDescription')}
              key={form.key('parameters.twitch_token')}
              {...form.getInputProps('parameters.twitch_token')}
            />

            <TextInput
              label={t('videoSettings.convertFFmpegArgsLabel')}
              description={t('videoSettings.convertFFmpegArgsDescription')}
              key={form.key('parameters.video_convert')}
              {...form.getInputProps('parameters.video_convert')}
            />

            <Title mt={10} order={3}>{t('videoSettings.liveStreamTitle')}</Title>

            <TextInput
              label={t('videoSettings.streamlinkArgsLabel')}
              description={t('videoSettings.streamlinkArgsDescription')}
              key={form.key('parameters.streamlink_live')}
              {...form.getInputProps('parameters.streamlink_live')}
            />

            <Title mt={5} order={5}>{t('videoSettings.proxySettings')}</Title>
            <Text>{t('videoSettings.proxySettingsDescription')}</Text>

            <Checkbox
              mt={10}
              label={t('videoSettings.proxyEnableLabel')}
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
                      label={t('videoSettings.proxyURLLabel')}
                      key={form.key(`livestream.proxies.${index}.url`)}
                      {...form.getInputProps(`livestream.proxies.${index}.url`)}
                    />
                    <TextInput
                      label={t('videoSettings.proxyHeaderLabel')}
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
              {t('videoSettings.proxyAddButton')}
            </Button>

            <MultiSelect
              label={t('videoSettings.whitelistChannelsLabel')}
              description={t('videoSettings.whitelistChannelsDescription')}
              data={channelSelect}
              key={form.key('livestream.proxy_whitelist')}
              {...form.getInputProps('livestream.proxy_whitelist')}
              searchable
            />

            <Title mt={10} order={3}>{t('chatSettings.header')}</Title>

            <TextInput
              label={t('chatSettings.chatRenderArgsLabel')}
              description={t('chatSettings.chatRenderArgsDescription')}
              key={form.key('parameters.chat_render')}
              {...form.getInputProps('parameters.chat_render')}
            />

            <Button
              mt={15}
              type="submit"
              fullWidth
              loading={editConfigMutate.isPending}
            >
              {t('submit')}
            </Button>

          </form>

        </Card>

      </Container>
    </div>
  );
}

export default AdminSettingsPage;