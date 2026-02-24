"use client"
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText"
import { useAxiosPrivate } from "@/app/hooks/useAxios"
import { Config, ProxyListItem, ProxyType, useEditConfig, useGetConfig } from "@/app/hooks/useConfig"
import { ActionIcon, Button, Card, Checkbox, Code, Collapse, Container, MultiSelect, NumberInput, Select, Text, Textarea, TextInput, Title } from "@mantine/core"
import { useForm } from "@mantine/form"
import { useDisclosure } from "@mantine/hooks"
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
  const [storageTemplateOpened, { toggle: toggleStorageTemplate }] = useDisclosure(false);
  const [channelSelect, setChannelSelect] = useState<SelectOption[]>([]);
  const axiosPrivate = useAxiosPrivate()

  const editConfigMutate = useEditConfig()

  const { data, isPending, isError } = useGetConfig(axiosPrivate)

  // Proxy types using the enum
  const proxyTypes: { label: string; value: ProxyType }[] = Object.entries(ProxyType).map(([key, value]) => ({
    label: key,
    value: value
  }));

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
        yt_dlp_video: data?.parameters.yt_dlp_video || "",
      },
      archive: {
        save_as_hls: data?.archive.save_as_hls ?? false,
        generate_sprite_thumbnails: data?.archive.generate_sprite_thumbnails ?? true
      },
      storage_templates: {
        folder_template: data?.storage_templates.folder_template || "",
        file_template: data?.storage_templates.file_template || "",
      },
      livestream: {
        proxies: data?.livestream.proxies || [],
        proxy_enabled: data?.livestream.proxy_enabled ?? true,
        proxy_whitelist: data?.livestream.proxy_whitelist || [],
        watch_while_archiving: data?.livestream.watch_while_archiving ?? false,
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
      <Container mt={15} size="7xl">
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

            <Collapse in={storageTemplateOpened} px={25} pt={10}>

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
                    description="Do not include the file extension. The file type will be appended to the end of the file name such as -video -chat and -thumbnail."
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
                    <ul>
                      <li><Code>{"{{uuid}}"}</Code>: Unique identifier for the archive</li>
                    </ul>
                    <Text>Video</Text>
                    <ul>
                      <li><Code>{"{{id}}"}</Code>: Video ID</li>
                      <li><Code>{"{{channel}}"}</Code>: Channel name</li>
                      <li><Code>{"{{title}}"}</Code>: Video title (file safe)</li>
                      <li><Code>{"{{type}}"}</Code>: Video type (live, archive, highlight)</li>
                      <li><Code>{"{{date}}"}</Code>: Formatted date (YYYY-MM-DD)</li>
                      <li><Code>{"{{YYYY}}"}</Code>: Year</li>
                      <li><Code>{"{{MM}}"}</Code>: Month</li>
                      <li><Code>{"{{DD}}"}</Code>: Day</li>
                      <li><Code>{"{{HH}}"}</Code>: Hour</li>
                    </ul>

                  </div>
                </div>

                <div>
                  <Title mt={5} order={4}>
                    {t('archiveSettings.examplesText')}
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

            <TextInput
              label={t('videoSettings.ytdlpVideoArgsLabel')}
              description={t('videoSettings.ytdlpVideoArgsDescription')}
              key={form.key('parameters.yt_dlp_video')}
              {...form.getInputProps('parameters.yt_dlp_video')}
            />

            <Title mt={10} order={3}>{t('videoSettings.liveStreamTitle')}</Title>

            <Checkbox
              mt={10}
              label={t('videoSettings.watchWhileArchivingLabel')}
              description={t('videoSettings.watchWhileArchivingDescription')}
              key={form.key('livestream.watch_while_archiving')}
              {...form.getInputProps('livestream.watch_while_archiving', { type: "checkbox" })}
              mr={15}
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
                    <Select
                      w={350}
                      label={t('videoSettings.proxyTypeLabel')}
                      key={form.key(`livestream.proxies.${index}.proxy_type`)}
                      data={proxyTypes}
                      defaultValue={ProxyType.TwitchHLS}
                      {...form.getInputProps(`livestream.proxies.${index}.proxy_type`)}
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
                form.insertListItem('livestream.proxies', { url: '', header: '', proxy_type: ProxyType.TwitchHLS })
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