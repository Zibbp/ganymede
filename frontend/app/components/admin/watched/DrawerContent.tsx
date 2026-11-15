import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Channel, useFetchChannels } from "@/app/hooks/useChannels";
import { WatchedChannel, WatchedChannelTitleRegex, useBulkCreateWatchedChannels, useBulkEditWatchedChannels, useEditWatchedChannel, useGetWatchedChannesl } from "@/app/hooks/useWatchedChannels";
import { ActionIcon, Button, NumberInput, TextInput, Tooltip, Text, Divider, Checkbox, Select, Title, Box, Group, Grid, MultiSelect, Collapse, Alert } from "@mantine/core";
import { useForm } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { IconPlus, IconTrash } from "@tabler/icons-react";
import { useEffect, useRef, useState } from "react";
import classes from "./Watched.module.css"
import { getTwitchCategories } from "@/app/hooks/useCategory";
import { useDisclosure } from "@mantine/hooks";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { metadata } from "@/app/layout";
import { VideoQuality } from "@/app/hooks/useArchive";

type Props = {
  watchedChannel: WatchedChannel | null
  // Set for bulk edit; the form settings are applied to every channel.
  watchedChannels?: WatchedChannel[]
  mode: WatchedChannelEditMode
  handleClose: () => void;
}

interface SelectOption {
  label: string;
  value: string;
}

export enum WatchedChannelEditMode {
  Create = "create",
  Edit = "edit",
  BulkEdit = "bulk_edit",
}

// Quality options using the enum
const qualityOptions: SelectOption[] = Object.entries(VideoQuality).map(([key, value]) => ({
  label: key.replace('quality', ''),
  value: value
}));

// Twitch clips do not publish audio-only or 160p renditions.
const clipQualityOptions: SelectOption[] = qualityOptions.filter(
  (option) => option.value !== VideoQuality.audio && option.value !== VideoQuality.quality160p
);

const AdminWatchedChannelDrawerContent = ({ watchedChannel: singleWatchedChannel, watchedChannels = [], mode, handleClose }: Props) => {
  const t = useTranslations('AdminWatchedComponents')
  const axiosPrivate = useAxiosPrivate()
  const isBulkEdit = mode === WatchedChannelEditMode.BulkEdit
  // Bulk edit seeds the form from the first selected channel.
  const watchedChannel = isBulkEdit ? (watchedChannels[0] ?? null) : singleWatchedChannel
  const [liveTitleRegexes, setLiveTitleRegexes] = useState<WatchedChannelTitleRegex[]>(
    watchedChannel?.edges.title_regex || []
  );

  const [channelSelect, setChannelSelect] = useState<SelectOption[]>([]);

  const [clipMoreInfoOpened, { toggle: clipMoreInfoToggle }] = useDisclosure(false);

  // Initialize edit watched channel mutation
  const editWatchedChannelMutation = useEditWatchedChannel();
  const bulkEditWatchedChannelsMutation = useBulkEditWatchedChannels();
  const { data: existingWatchedChannels } = useGetWatchedChannesl(axiosPrivate);

  const form = useForm({
    mode: "controlled",
    initialValues: {
      id: watchedChannel?.id || "",
      watch_live: watchedChannel?.watch_live ?? false,
      watch_vod: watchedChannel?.watch_vod ?? false,
      download_archives: watchedChannel?.download_archives ?? true,
      download_highlights: watchedChannel?.download_highlights ?? true,
      download_uploads: watchedChannel?.download_uploads ?? true,
      resolution: watchedChannel?.resolution || "best",
      vod_resolution: watchedChannel?.vod_resolution || watchedChannel?.resolution || "best",
      clip_resolution: watchedChannel?.clip_resolution || watchedChannel?.vod_resolution || watchedChannel?.resolution || "best",
      archive_chat: watchedChannel?.archive_chat ?? true,
      channel_id: watchedChannel?.edges.channel.id || "",
      channel_ids: [] as string[],
      render_chat: watchedChannel?.render_chat ?? true,
      download_sub_only: watchedChannel?.download_sub_only ?? false,
      video_age: watchedChannel?.video_age || 0,
      apply_categories_to_live: watchedChannel?.apply_categories_to_live ?? false,
      strict_categories_live: watchedChannel?.strict_categories_live ?? false,
      blacklist_categories: watchedChannel?.blacklist_categories ?? false,
      watch_clips: watchedChannel?.watch_clips ?? false,
      clips_limit: watchedChannel?.clips_limit || 5,
      clips_interval_days: watchedChannel?.clips_interval_days || 7,
      clips_ignore_last_checked: watchedChannel?.clips_ignore_last_checked ?? false,
      update_metadata_minutes: watchedChannel?.update_metadata_minutes || 15,
      live_title_regexes: [],
      categories: [] as string[],
    },
  })

  // Bulk edit seeds categories once so a refreshed seed record can't discard unsaved edits.
  const bulkCategoriesSeeded = useRef(false)

  useEffect(() => {
    if (!watchedChannel || !watchedChannel.edges.categories) return

    if (isBulkEdit) {
      if (bulkCategoriesSeeded.current) return
      bulkCategoriesSeeded.current = true
    }

    const categories = watchedChannel.edges.categories.map((category) => category.name);
    form.setFieldValue('categories', categories);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [watchedChannel])

  const [twitchCategoriesLoading, setTwitchCategoriesLoading] = useState(false);
  const [formattedTwitchCategories, setFormattedTwitchCategories] = useState<SelectOption[]>([]);

  const handleGetTwitchCategories = async () => {
    try {
      setTwitchCategoriesLoading(true)

      const categories = await getTwitchCategories()
      if (!categories) return

      const tmpArr = categories.map((category) => ({
        label: category.name,
        value: category.name,
      })).filter((item, index, self) =>
        index === self.findIndex((t) => t.label === item.label)
      );
      // Add "no category" option
      tmpArr.unshift({
        label: "No Category",
        value: "",
      });

      setFormattedTwitchCategories(tmpArr);
    } catch (error) {
      console.error(error)
    } finally {
      setTwitchCategoriesLoading(false)
    }
  }

  const createWatchedChannelMutation = useBulkCreateWatchedChannels();

  const handleSubmitForm = async () => {
    const formValues = form.getValues()

    try {
      if (mode === WatchedChannelEditMode.Create) {
        if (formValues.channel_ids.length === 0) {
          showNotification({ message: t('selectChannelsNotification'), color: "red" });
          return;
        }

        const newWatchedChannel: WatchedChannel = {
          id: "", // Will be generated by backend
          watch_live: formValues.watch_live,
          watch_vod: formValues.watch_vod,
          download_archives: formValues.download_archives,
          download_highlights: formValues.download_highlights,
          download_uploads: formValues.download_uploads,
          resolution: formValues.resolution,
          vod_resolution: formValues.vod_resolution,
          clip_resolution: formValues.clip_resolution,
          archive_chat: formValues.archive_chat,
          render_chat: formValues.render_chat,
          download_sub_only: formValues.download_sub_only,
          video_age: formValues.video_age,
          apply_categories_to_live: formValues.apply_categories_to_live,
          strict_categories_live: formValues.strict_categories_live,
          blacklist_categories: formValues.blacklist_categories,
          watch_clips: formValues.watch_clips,
          clips_limit: formValues.clips_limit,
          clips_interval_days: formValues.clips_interval_days,
          clips_ignore_last_checked: formValues.clips_ignore_last_checked,
          update_metadata_minutes: formValues.update_metadata_minutes,
          is_live: false, // Default value
          edges: {
            channel: {} as Channel,
            categories: [],
            title_regex: liveTitleRegexes
          },
          last_live: "",
          updated_at: "",
          created_at: ""
        };

        const result = await createWatchedChannelMutation.mutateAsync({
          axiosPrivate,
          channelIds: formValues.channel_ids,
          watchedChannel: newWatchedChannel,
          categories: formValues.categories
        });

        if (result.failed > 0) {
          showNotification({
            message: t('bulkCreatePartialNotification', { succeeded: result.succeeded, failed: result.failed }),
            color: result.succeeded > 0 ? "yellow" : "red"
          });
          if (result.succeeded === 0) return;
        } else {
          showNotification({
            message: t('createNotification', { count: result.succeeded }),
            color: "green"
          });
        }

        handleClose();
      } else if (isBulkEdit) {
        if (watchedChannels.length === 0) return;

        const result = await bulkEditWatchedChannelsMutation.mutateAsync({
          axiosPrivate,
          watchedChannels,
          settings: {
            watch_live: formValues.watch_live,
            watch_vod: formValues.watch_vod,
            download_archives: formValues.download_archives,
            download_highlights: formValues.download_highlights,
            download_uploads: formValues.download_uploads,
            resolution: formValues.resolution,
            vod_resolution: formValues.vod_resolution,
            clip_resolution: formValues.clip_resolution,
            archive_chat: formValues.archive_chat,
            render_chat: formValues.render_chat,
            download_sub_only: formValues.download_sub_only,
            video_age: formValues.video_age,
            apply_categories_to_live: formValues.apply_categories_to_live,
            strict_categories_live: formValues.strict_categories_live,
            blacklist_categories: formValues.blacklist_categories,
            watch_clips: formValues.watch_clips,
            clips_limit: formValues.clips_limit,
            clips_interval_days: formValues.clips_interval_days,
            clips_ignore_last_checked: formValues.clips_ignore_last_checked,
            update_metadata_minutes: formValues.update_metadata_minutes,
          },
          titleRegexes: liveTitleRegexes.map((r) => ({ ...r, id: "" })),
          categories: formValues.categories
        });

        if (result.failed > 0) {
          showNotification({
            message: t('bulkEditPartialNotification', { succeeded: result.succeeded, failed: result.failed }),
            color: result.succeeded > 0 ? "yellow" : "red"
          });
          if (result.succeeded === 0) return;
        } else {
          showNotification({
            message: t('bulkEditNotification', { count: result.succeeded }),
            color: "green"
          });
        }

        handleClose();
      } else if (mode === WatchedChannelEditMode.Edit && watchedChannel) {
        const updatedWatchedChannel: WatchedChannel = {
          ...watchedChannel,
          watch_live: formValues.watch_live,
          watch_vod: formValues.watch_vod,
          download_archives: formValues.download_archives,
          download_highlights: formValues.download_highlights,
          download_uploads: formValues.download_uploads,
          resolution: formValues.resolution,
          vod_resolution: formValues.vod_resolution,
          clip_resolution: formValues.clip_resolution,
          archive_chat: formValues.archive_chat,
          render_chat: formValues.render_chat,
          download_sub_only: formValues.download_sub_only,
          video_age: formValues.video_age,
          apply_categories_to_live: formValues.apply_categories_to_live,
          strict_categories_live: formValues.strict_categories_live,
          blacklist_categories: formValues.blacklist_categories,
          watch_clips: formValues.watch_clips,
          clips_limit: formValues.clips_limit,
          clips_interval_days: formValues.clips_interval_days,
          clips_ignore_last_checked: formValues.clips_ignore_last_checked,
          update_metadata_minutes: formValues.update_metadata_minutes,
          edges: {
            ...watchedChannel.edges,
            title_regex: liveTitleRegexes
          }
        };

        await editWatchedChannelMutation.mutateAsync({
          axiosPrivate,
          watchedChannel: updatedWatchedChannel,
          categories: formValues.categories
        });

        showNotification({
          message: t('editNotification'),
          color: "green"
        });

        handleClose();
      }
    } catch (error) {
      console.error(error);
      showNotification({
        message: mode === WatchedChannelEditMode.Create
          ? t('failedCreateNotification')
          : t('failedEditNotification'),
        color: "red"
      });
    }
  }


  const { data: channels } = useFetchChannels();

  useEffect(() => {
    if (!channels) return;

    // Channels that are already watched can't be added again.
    const watchedIds = new Set((existingWatchedChannels ?? []).map((wc) => wc.edges.channel.id));

    const transformedChannels: SelectOption[] = channels
      .filter((channel: Channel) => mode !== WatchedChannelEditMode.Create || !watchedIds.has(channel.id))
      .map((channel: Channel) => ({
        label: channel.name,
        value: channel.id,
      }));

    setChannelSelect(transformedChannels);
  }, [channels, existingWatchedChannels, mode]);

  return (
    <div>
      <Text>{t.rich('headerText', {
        wiki: (chunks) => <a href="https://github.com/Zibbp/ganymede/wiki/Watched-Channels" target="_blank" className={classes.link}>{chunks}</a>,
      })}</Text>

      <form onSubmit={form.onSubmit(() => {
        handleSubmitForm()
      })}>
        {isBulkEdit ? (
          <>
            <Alert mb={10} color="yellow" title={t('bulkEditAlertTitle', { count: watchedChannels.length })}>
              {t('bulkEditAlertText')}
            </Alert>
            <TextInput
              disabled={true}
              label={t('channelsLabel')}
              value={watchedChannels.map((wc) => wc.edges.channel.display_name).join(", ")}
            />
          </>
        ) : (
          <>
            <TextInput
              disabled={true}
              label={t('idLabel')}
              placeholder="Auto generated"
              key={form.key('id')}
              {...form.getInputProps('id')}
            />

            {mode === WatchedChannelEditMode.Create ? (
              <MultiSelect
                label={t('channelsLabel')}
                data={channelSelect}
                key={form.key('channel_ids')}
                {...form.getInputProps('channel_ids')}
                searchable
                clearable
              />
            ) : (
              <Select
                disabled={true}
                label={t('channelLabel')}
                data={channelSelect}
                key={form.key('channel_id')}
                {...form.getInputProps('channel_id')}
                searchable
              />
            )}
          </>
        )}

        <Checkbox
          mt={10}
          label={t('archiveChatLabel')}
          key={form.key('archive_chat')}
          {...form.getInputProps('archive_chat', { type: "checkbox" })}
        />

        <Checkbox
          mt={5}
          label={t('renderChatLabel')}
          key={form.key('render_chat')}
          {...form.getInputProps('render_chat', { type: "checkbox" })}
        />

        <Divider my="sm" size="md" />

        <div>
          <Title order={3}>{t('liveStreamsText')}</Title>
          <Text>{t('liveStreamsDescription')}</Text>

          <Checkbox
            mt={5}
            label={t('watchLiveLabel')}
            key={form.key('watch_live')}
            {...form.getInputProps('watch_live', { type: "checkbox" })}
          />

          <Select
            mt={5}
            label={t('liveResolutionLabel')}
            data={qualityOptions}
            key={form.key('resolution')}
            {...form.getInputProps('resolution')}
            searchable
          />

          {form.values.watch_live && (
            <NumberInput
              mt={5}
              label={t('updateMetadataLabel')}
              description={t('updateMetadataDescription', { minutes: form.values.update_metadata_minutes })}
              key={form.key('update_metadata_minutes')}
              {...form.getInputProps('update_metadata_minutes')}
              min={0}
            />
          )}

        </div>

        <Divider my="sm" size="md" />

        <div>
          <Title order={3}>{t('videosText')}</Title>
          <Text>{t('videosDescription')}</Text>
          <Text size="xs" fs="italic">
            {t('videosOccurrence')}
          </Text>

          <Checkbox
            mt={5}
            label={t('watchVideosLabel')}
            key={form.key('watch_vod')}
            {...form.getInputProps('watch_vod', { type: "checkbox" })}
          />

          <Select
            mt={5}
            label={t('vodResolutionLabel')}
            data={qualityOptions}
            key={form.key('vod_resolution')}
            {...form.getInputProps('vod_resolution')}
            searchable
          />

          <Box ml={30}>
            <Checkbox
              mt={5}
              label={t('downloadArchivesLabel')}
              description={t('downloadArchivesDescription')}
              key={form.key('download_archives')}
              {...form.getInputProps('download_archives', { type: "checkbox" })}
            />
            <Checkbox
              mt={5}
              label={t('downloadHighlightsLabel')}
              description={t('downloadHighlightsDescription')}
              key={form.key('download_highlights')}
              {...form.getInputProps('download_highlights', { type: "checkbox" })}
            />
            <Checkbox
              mt={5}
              label={t('downloadUploadsLabel')}
              description={t('downloadUploadsDescription')}
              key={form.key('download_uploads')}
              {...form.getInputProps('download_uploads', { type: "checkbox" })}
            />
            <Checkbox
              mt={5}
              label={t('downloadSubOnlyLabel')}
              description={t('downloadSubOnlyDescription')}
              key={form.key('download_sub_only')}
              {...form.getInputProps('download_sub_only', { type: "checkbox" })}
            />
          </Box>
        </div>

        <Divider my="sm" size="md" />

        <div>
          <Title order={3}>{t('channelClipsText')}</Title>
          <Text>{t('channelClipsDescription')}</Text>
          <Text size="xs">
            {t('channelClipsDescription1')}
          </Text>

          <Link href="#">
            <Text c="blue" mt={5} size="xs" onClick={clipMoreInfoToggle}>{t('channelClipsMoreInformationButton')}</Text>
          </Link>
          <Collapse expanded={clipMoreInfoOpened}>
            <Text size="xs" mt={5}>
              {t('channelClipsMoreInformationText')}
            </Text>
          </Collapse>


          <Checkbox
            my={5}
            label={t('watchClipsLabel')}
            key={form.key('watch_clips')}
            {...form.getInputProps('watch_clips', { type: "checkbox" })}
          />

          <Select
            mt={5}
            label={t('clipResolutionLabel')}
            description={t('clipResolutionDescription')}
            data={clipQualityOptions}
            key={form.key('clip_resolution')}
            {...form.getInputProps('clip_resolution')}
            searchable
          />


          <NumberInput
            label={t('numberOfClipsLabel')}
            description={t('numberOfClipsDescription')}
            key={form.key('clips_limit')}
            {...form.getInputProps('clips_limit')}
            min={1}
          />

          <NumberInput
            label={t('intervalDaysLabel')}
            description={t('intervalDaysDescription')}
            key={form.key('clips_interval_days')}
            {...form.getInputProps('clips_interval_days')}
            min={1}
          />

          <Checkbox
            mt={10}
            label={t('ignoreLastCheckedDateLabel')}
            description={t('ignoreLastCheckedDateDescription')}
            key={form.key('clips_ignore_last_checked')}
            {...form.getInputProps('clips_ignore_last_checked', { type: "checkbox" })}
          />

        </div>

        <Divider my="sm" size="md" />

        <div>
          <Title order={3}>{t('advancedText')}</Title>

          <NumberInput
            label={t('maxVideoAgeLabel')}
            description={t('maxVideoAgeDescription')}
            key={form.key('video_age')}
            {...form.getInputProps('video_age')}
          />

          <Group mt={5}>
            <Title order={5}>{t('titleRegexText')}</Title>
            <Tooltip label={t('titleRegexAddTooltip')}>
              <ActionIcon size="sm" variant="filled" color="green" aria-label="Add Title Regex" onClick={() => {
                const newRegex: WatchedChannelTitleRegex = {
                  apply_to_videos: false,
                  negative: false,
                  regex: "",
                  id: ""
                }
                setLiveTitleRegexes(liveTitleRegexes => [...(liveTitleRegexes ?? []), newRegex])
              }}>
                <IconPlus style={{ width: '70%', height: '70%' }} stroke={1.5} />
              </ActionIcon>
            </Tooltip>
          </Group>
          <div>
            <Text size="sm">{t.rich('titleRegexDescription', { wiki: (chunks) => <a href="https://github.com/Zibbp/ganymede/wiki/Watched-Channel-Title-Regex" target="_blank" className={classes.link}>{chunks}</a> })}</Text>
          </div>

          <div>
            {liveTitleRegexes && liveTitleRegexes.map((regex: WatchedChannelTitleRegex, index) => (
              <div key={index}>
                <Grid grow>
                  <Grid.Col span={10}>
                    <TextInput
                      label={t('titleRegexLabel')}
                      placeholder="(?i:rerun)"
                      value={regex.regex}  // Use the specific regex from the current item
                      onChange={(e) => {
                        const updatedRegexes = [...liveTitleRegexes];
                        updatedRegexes[index] = {
                          ...updatedRegexes[index],
                          regex: e.currentTarget.value,
                        };
                        setLiveTitleRegexes(updatedRegexes);
                      }}
                    />

                    <Group mt={7}>
                      <Checkbox
                        defaultChecked
                        label={t('negativeLabel')}
                        description={t('negativeDescription')}
                        checked={regex.negative}
                        onChange={(e) => {
                          const updatedRegexes = [...liveTitleRegexes];
                          updatedRegexes[index].negative = e.currentTarget.checked;
                          setLiveTitleRegexes(updatedRegexes)
                        }}
                      />
                      <Checkbox
                        defaultChecked
                        label={t('applyToVideosLabel')}
                        description={t('applyToVideosDescription')}
                        checked={regex.apply_to_videos}
                        onChange={(e) => {
                          const updatedRegexes = [...liveTitleRegexes];
                          updatedRegexes[index].apply_to_videos = e.currentTarget.checked;
                          setLiveTitleRegexes(updatedRegexes)
                        }}
                      />
                    </Group>
                  </Grid.Col>
                  <Grid.Col span={1} mt={25}>
                    <Group>
                      <ActionIcon size="lg" variant="filled" color="red" aria-label="Settings" h={80} onClick={() => {
                        const updatedRegexs = [...liveTitleRegexes]
                        updatedRegexs.splice(index, 1)
                        setLiveTitleRegexes(updatedRegexs)
                      }}>
                        <IconTrash style={{ width: '70%', height: '70%' }} stroke={1.5} />
                      </ActionIcon>
                    </Group>
                  </Grid.Col>
                </Grid>
              </div>
            ))}
          </div>
        </div>

        <Divider my="sm" size="md" />

        <div>
          <Title order={3}>{t('categoriesText')}</Title>
          <Text size="sm">{t('categoriesDescription')}</Text>
        </div>

        <Checkbox
          mt={5}
          label={t('categoriesApplyToLivestreamsLabel')}
          description={t('categoriesApplyToLivestreamsDescription')}
          key={form.key('apply_categories_to_live')}
          {...form.getInputProps('apply_categories_to_live', { type: "checkbox" })}
        />

        <Checkbox
          mt={5}
          label={t('strictCategoriesLiveLabel')}
          description={t('strictCategoriesLiveDescription')}
          key={form.key('strict_categories_live')}
          {...form.getInputProps('strict_categories_live', { type: "checkbox" })}
          disabled={!form.values.apply_categories_to_live || form.values.blacklist_categories}
        />

        <Checkbox
          mt={5}
          label={t('blacklistCategoriesLiveLabel')}
          description={t('blacklistCategoriesLiveDescription')}
          key={form.key('blacklist_categories')}
          {...form.getInputProps('blacklist_categories', { type: "checkbox" })}
          disabled={form.values.strict_categories_live}
        />

        <Box mt={10}>
          {formattedTwitchCategories.length == 0 ? (
            <Button variant="filled" color="violet" onClick={() => handleGetTwitchCategories()}
              loading={twitchCategoriesLoading}
            >{t('categoriesLoadButton')}</Button>
          ) : (
            <MultiSelect
              searchable
              limit={20}
              data={formattedTwitchCategories}
              comboboxProps={{ position: 'top', middlewares: { flip: false, shift: false } }}
              placeholder={t('categoriesSearchPlaceholder')}
              clearable
              key={form.key('categories')}
              {...form.getInputProps('categories')}
            />
          )}
        </Box>

        <Button
          mt={10}
          type="submit"
          fullWidth
          loading={editWatchedChannelMutation.isPending || bulkEditWatchedChannelsMutation.isPending || createWatchedChannelMutation.isPending}
        >
          {mode === WatchedChannelEditMode.Create
            ? t('submitButton')
            : isBulkEdit
              ? t('bulkEditButton', { count: watchedChannels.length })
              : t('editButton')}
        </Button>
      </form>

    </div>
  );
}

export default AdminWatchedChannelDrawerContent;
