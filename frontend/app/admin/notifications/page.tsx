"use client"
import {
  ActionIcon,
  Badge,
  Box,
  Button,
  Checkbox,
  Code,
  Container,
  Drawer,
  Flex,
  Group,
  Modal,
  NativeSelect,
  Text,
  TextInput,
  Textarea,
  Title,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useForm } from "@mantine/form";
import { showNotification } from "@mantine/notifications";
import { DataTable } from "mantine-datatable";
import { useState } from "react";
import { AxiosError } from "axios";
import { useTranslations } from "next-intl";
import { IconEdit, IconPlayerPlay, IconTrash } from "@tabler/icons-react";
import GanymedeLoadingText from "@/app/components/utils/GanymedeLoadingText";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import {
  type Notification,
  type CreateNotificationInput,
  NotificationType,
  AppriseType,
  AppriseFormat,
  NotificationEventType,
  useGetNotifications,
  useCreateNotification,
  useUpdateNotification,
  useDeleteNotification,
  useTestNotification,
} from "@/app/hooks/useNotification";
import { usePageTitle } from "@/app/util/util";

// Extract error message from axios error or generic Error
const getErrorMessage = (error: unknown, fallback: string): string => {
  if (error instanceof AxiosError && error.response?.data?.message) {
    return error.response.data.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return fallback;
};

// Template variable definitions grouped by category
const CHANNEL_VARS = ["channel_id", "channel_ext_id", "channel_display_name"];
const VIDEO_VARS = ["vod_id", "vod_ext_id", "vod_platform", "vod_type", "vod_title", "vod_duration", "vod_views", "vod_resolution", "vod_streamed_at", "vod_created_at"];
const QUEUE_VARS = ["queue_id", "queue_created_at"];

// Which variables are available for each trigger type
const TRIGGER_VARIABLES: Record<string, { label: string; vars: string[] }[]> = {
  video_success: [
    { label: "Channel", vars: CHANNEL_VARS },
    { label: "Video", vars: VIDEO_VARS },
    { label: "Queue", vars: QUEUE_VARS },
  ],
  live_success: [
    { label: "Channel", vars: CHANNEL_VARS },
    { label: "Video", vars: VIDEO_VARS },
    { label: "Queue", vars: QUEUE_VARS },
  ],
  error: [
    { label: "Channel", vars: CHANNEL_VARS },
    { label: "Video", vars: VIDEO_VARS },
    { label: "Queue", vars: QUEUE_VARS },
    { label: "Error", vars: ["failed_task"] },
  ],
  is_live: [
    { label: "Channel", vars: CHANNEL_VARS },
    { label: "Live", vars: ["category"] },
  ],
};

const TemplateVariableHints = ({ triggerKey, variablesLabel }: { triggerKey: string; variablesLabel: string }) => {
  const groups = TRIGGER_VARIABLES[triggerKey];
  if (!groups) return null;
  return (
    <Box mt={4} ml={28}>
      <Text size="xs" c="dimmed">
        {variablesLabel}{" "}
        {groups.map((group, gi) => (
          <span key={group.label}>
            {gi > 0 && " "}
            {group.vars.map((v, vi) => (
              <span key={v}>
                {vi > 0 && " "}
                <Code style={{ fontSize: "10px" }}>{`{{${v}}}`}</Code>
              </span>
            ))}
          </span>
        ))}
      </Text>
    </Box>
  );
};

const AdminNotificationsPage = () => {
  const t = useTranslations("AdminNotificationsPage");
  usePageTitle(t("title"));

  const axiosPrivate = useAxiosPrivate();
  const { data: notifications, isPending, isError } = useGetNotifications(axiosPrivate);

  const createMutation = useCreateNotification();
  const updateMutation = useUpdateNotification();
  const deleteMutation = useDeleteNotification();
  const testMutation = useTestNotification();

  const [drawerOpened, { open: openDrawer, close: closeDrawer }] = useDisclosure(false);
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] = useDisclosure(false);
  const [testModalOpened, { open: openTestModal, close: closeTestModal }] = useDisclosure(false);
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null);
  const [deletingNotification, setDeletingNotification] = useState<Notification | null>(null);
  const [testingNotification, setTestingNotification] = useState<Notification | null>(null);
  const [testEventType, setTestEventType] = useState<NotificationEventType>(NotificationEventType.VideoSuccess);

  const form = useForm<CreateNotificationInput>({
    initialValues: {
      name: "",
      enabled: true,
      type: NotificationType.Webhook,
      url: "",
      trigger_video_success: false,
      trigger_live_success: false,
      trigger_error: false,
      trigger_is_live: false,
      video_success_template: "✅ Video Archived: {{vod_title}} by {{channel_display_name}}.",
      live_success_template: "✅ Live Stream Archived: {{vod_title}} by {{channel_display_name}}.",
      error_template: "⚠️ Error: Queue {{queue_id}} failed at task {{failed_task}}.",
      is_live_template: "🔴 {{channel_display_name}} is live!",
      apprise_urls: "",
      apprise_title: "",
      apprise_type: AppriseType.Info,
      apprise_tag: "",
      apprise_format: AppriseFormat.Text,
    },
    validate: (values) => {
      const errors: Record<string, string> = {};

      if (!values.name.trim()) {
        errors.name = t("validation.nameRequired");
      }

      if (!values.url.trim()) {
        errors.url = t("validation.urlRequired");
      } else {
        try {
          const url = new URL(values.url);
          if (!["http:", "https:"].includes(url.protocol)) {
            errors.url = t("validation.urlInvalidProtocol");
          }
        } catch {
          errors.url = t("validation.urlInvalid");
        }
      }

      if (!values.trigger_video_success && !values.trigger_live_success && !values.trigger_error && !values.trigger_is_live) {
        errors.trigger_video_success = t("validation.triggerRequired");
      }

      if (values.trigger_video_success && !values.video_success_template.trim()) {
        errors.video_success_template = t("validation.templateRequired");
      }
      if (values.trigger_live_success && !values.live_success_template.trim()) {
        errors.live_success_template = t("validation.templateRequired");
      }
      if (values.trigger_error && !values.error_template.trim()) {
        errors.error_template = t("validation.templateRequired");
      }
      if (values.trigger_is_live && !values.is_live_template.trim()) {
        errors.is_live_template = t("validation.templateRequired");
      }

      if (values.type === NotificationType.Apprise && !values.apprise_urls.trim() && !values.apprise_tag.trim()) {
        errors.apprise_urls = t("validation.appriseUrlsOrTagRequired");
      }

      return errors;
    },
  });

  const handleOpenCreate = () => {
    setEditingNotification(null);
    form.reset();
    openDrawer();
  };

  const handleOpenEdit = (n: Notification) => {
    setEditingNotification(n);
    form.setValues({
      name: n.name,
      enabled: n.enabled,
      type: n.type,
      url: n.url,
      trigger_video_success: n.trigger_video_success,
      trigger_live_success: n.trigger_live_success,
      trigger_error: n.trigger_error,
      trigger_is_live: n.trigger_is_live,
      video_success_template: n.video_success_template,
      live_success_template: n.live_success_template,
      error_template: n.error_template,
      is_live_template: n.is_live_template,
      apprise_urls: n.apprise_urls,
      apprise_title: n.apprise_title,
      apprise_type: n.apprise_type || AppriseType.Info,
      apprise_tag: n.apprise_tag,
      apprise_format: n.apprise_format || AppriseFormat.Text,
    });
    openDrawer();
  };

  const handleSubmit = async () => {
    const values = form.values;
    try {
      if (editingNotification) {
        await updateMutation.mutateAsync({
          axiosPrivate,
          id: editingNotification.id,
          input: values,
        });
        showNotification({
          title: t("toast.successTitle"),
          message: t("toast.notificationUpdated"),
        });
      } else {
        await createMutation.mutateAsync({
          axiosPrivate,
          input: values,
        });
        showNotification({
          title: t("toast.successTitle"),
          message: t("toast.notificationCreated"),
        });
      }
      closeDrawer();
    } catch (error) {
      showNotification({
        title: t("toast.errorTitle"),
        message: getErrorMessage(error, editingNotification ? t("toast.updateFailed") : t("toast.createFailed")),
        color: "red",
      });
    }
  };

  const handleDelete = async () => {
    if (!deletingNotification) return;
    try {
      await deleteMutation.mutateAsync({
        axiosPrivate,
        id: deletingNotification.id,
      });
      showNotification({
        title: t("toast.successTitle"),
        message: t("toast.notificationDeleted"),
      });
      closeDeleteModal();
    } catch (error) {
      showNotification({
        title: t("toast.errorTitle"),
        message: getErrorMessage(error, t("toast.deleteFailed")),
        color: "red",
      });
    }
  };

  const handleTest = async () => {
    if (!testingNotification) return;
    try {
      await testMutation.mutateAsync({
        axiosPrivate,
        id: testingNotification.id,
        eventType: testEventType,
      });
      showNotification({
        title: t("toast.successTitle"),
        message: t("toast.testSent"),
      });
      closeTestModal();
    } catch (error) {
      showNotification({
        title: t("toast.errorTitle"),
        message: getErrorMessage(error, t("toast.testFailed")),
        color: "red",
      });
    }
  };

  if (isPending) return <GanymedeLoadingText message={t("loading")} />;
  if (isError) return <div>{t("error")}</div>;

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2}>
          <Title>{t("header")}</Title>
          <Button onClick={handleOpenCreate} variant="default">
            {t("createButton")}
          </Button>
        </Group>

        <Text mt={5} mb={10}>
          {t("description")}{" "}
          <a href="https://github.com/Zibbp/ganymede/wiki/Notifications" target="_blank" rel="noopener noreferrer">
            {t("descriptionWikiLink")}
          </a>{" "}
          {t("descriptionEnd")}
        </Text>

        <Box mt={5}>
          <DataTable<Notification>
            withTableBorder
            borderRadius="sm"
            withColumnBorders
            striped
            highlightOnHover
            records={notifications ?? []}
            columns={[
              {
                accessor: "name",
                title: t("columns.name"),
              },
              {
                accessor: "type",
                title: t("columns.type"),
                render: (n) => (
                  <Badge color={n.type === NotificationType.Apprise ? "grape" : "blue"} variant="light">
                    {n.type}
                  </Badge>
                ),
              },
              {
                accessor: "enabled",
                title: t("columns.enabled"),
                render: (n) => (
                  <Badge color={n.enabled ? "green" : "gray"} variant="light">
                    {n.enabled ? t("enabledYes") : t("enabledNo")}
                  </Badge>
                ),
              },
              {
                accessor: "triggers",
                title: t("columns.triggers"),
                render: (n) => {
                  const triggers = [];
                  if (n.trigger_video_success) triggers.push(t("triggerVideoSuccess"));
                  if (n.trigger_live_success) triggers.push(t("triggerLiveSuccess"));
                  if (n.trigger_error) triggers.push(t("triggerError"));
                  if (n.trigger_is_live) triggers.push(t("triggerIsLive"));
                  return (
                    <Group gap={4}>
                      {triggers.map((tr) => (
                        <Badge key={tr} size="xs" variant="outline">
                          {tr}
                        </Badge>
                      ))}
                      {triggers.length === 0 && (
                        <Text size="xs" c="dimmed">{t("triggersNone")}</Text>
                      )}
                    </Group>
                  );
                },
              },
              {
                accessor: "actions",
                title: t("columns.actions"),
                render: (n) => (
                  <Group gap={4}>
                    <ActionIcon
                      variant="light"
                      color="blue"
                      onClick={() => handleOpenEdit(n)}
                    >
                      <IconEdit size={18} />
                    </ActionIcon>
                    <ActionIcon
                      variant="light"
                      color="violet"
                      onClick={() => {
                        setTestingNotification(n);
                        openTestModal();
                      }}
                    >
                      <IconPlayerPlay size={18} />
                    </ActionIcon>
                    <ActionIcon
                      variant="light"
                      color="red"
                      onClick={() => {
                        setDeletingNotification(n);
                        openDeleteModal();
                      }}
                    >
                      <IconTrash size={18} />
                    </ActionIcon>
                  </Group>
                ),
              },
            ]}
          />
        </Box>
      </Container>

      {/* Create / Edit Drawer */}
      <Drawer
        opened={drawerOpened}
        onClose={closeDrawer}
        position="right"
        size="lg"
        title={editingNotification ? t("drawer.titleEdit") : t("drawer.titleCreate")}
      >
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <TextInput
            label={t("drawer.nameLabel")}
            placeholder={form.values.type === NotificationType.Apprise ? t("drawer.namePlaceholderApprise") : t("drawer.namePlaceholderWebhook")}
            required
            {...form.getInputProps("name")}
          />
          <Checkbox
            mt={10}
            label={t("drawer.enabledLabel")}
            {...form.getInputProps("enabled", { type: "checkbox" })}
          />
          <NativeSelect
            mt={10}
            label={t("drawer.typeLabel")}
            required
            data={[
              { value: NotificationType.Webhook, label: "Webhook" },
              { value: NotificationType.Apprise, label: "Apprise" },
            ]}
            {...form.getInputProps("type")}
          />
          <TextInput
            mt={10}
            label={form.values.type === NotificationType.Apprise ? t("drawer.urlLabelApprise") : t("drawer.urlLabelWebhook")}
            description={
              form.values.type === NotificationType.Apprise
                ? t("drawer.urlDescriptionApprise")
                : t("drawer.urlDescriptionWebhook")
            }
            placeholder={
              form.values.type === NotificationType.Apprise
                ? t("drawer.urlPlaceholderApprise")
                : t("drawer.urlPlaceholderWebhook")
            }
            required
            {...form.getInputProps("url")}
          />

          <Text fw={700} size="sm" mt={20}>
            {t("drawer.eventTriggersLabel")} <Text component="span" c="red" size="sm">*</Text>
          </Text>
          <Text size="xs" c="dimmed">{t("drawer.eventTriggersDescription")}</Text>
          {form.errors.trigger_video_success && (
            <Text size="xs" c="red" mt={2}>{form.errors.trigger_video_success}</Text>
          )}

          <Checkbox
            mt={10}
            label={t("drawer.triggerVideoArchiveSuccess")}
            {...form.getInputProps("trigger_video_success", { type: "checkbox" })}
          />
          {form.values.trigger_video_success && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label={t("drawer.messageLabel")}
                required
                {...form.getInputProps("video_success_template")}
              />
              <TemplateVariableHints triggerKey="video_success" variablesLabel={t("drawer.variablesLabel")} />
            </>
          )}

          <Checkbox
            mt={10}
            label={t("drawer.triggerLiveArchiveSuccess")}
            {...form.getInputProps("trigger_live_success", { type: "checkbox" })}
          />
          {form.values.trigger_live_success && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label={t("drawer.messageLabel")}
                required
                {...form.getInputProps("live_success_template")}
              />
              <TemplateVariableHints triggerKey="live_success" variablesLabel={t("drawer.variablesLabel")} />
            </>
          )}

          <Checkbox
            mt={10}
            label={t("drawer.triggerError")}
            {...form.getInputProps("trigger_error", { type: "checkbox" })}
          />
          {form.values.trigger_error && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label={t("drawer.messageLabel")}
                required
                {...form.getInputProps("error_template")}
              />
              <TemplateVariableHints triggerKey="error" variablesLabel={t("drawer.variablesLabel")} />
            </>
          )}

          <Checkbox
            mt={10}
            label={t("drawer.triggerChannelIsLive")}
            {...form.getInputProps("trigger_is_live", { type: "checkbox" })}
          />
          {form.values.trigger_is_live && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label={t("drawer.messageLabel")}
                required
                {...form.getInputProps("is_live_template")}
              />
              <TemplateVariableHints triggerKey="is_live" variablesLabel={t("drawer.variablesLabel")} />
            </>
          )}

          {/* Apprise-specific fields */}
          {form.values.type === NotificationType.Apprise && (
            <>
              <Title order={4} mt={20}>{t("drawer.appriseSettingsTitle")}</Title>
              <Text size="sm" c="dimmed">
                {t("drawer.appriseSettingsDescription")}{" "}
                <a href="https://github.com/caronc/apprise-api" target="_blank" rel="noopener noreferrer">
                  {t("drawer.appriseSettingsLink")}
                </a>{" "}
                {t("drawer.appriseSettingsDescriptionEnd")}
              </Text>
              <TextInput
                mt={10}
                label={t("drawer.appriseUrlsLabel")}
                description={t("drawer.appriseUrlsDescription")}
                placeholder={t("drawer.appriseUrlsPlaceholder")}
                {...form.getInputProps("apprise_urls")}
              />
              <TextInput
                mt={10}
                label={t("drawer.appriseTitleLabel")}
                description={t("drawer.appriseTitleDescription")}
                placeholder={t("drawer.appriseTitlePlaceholder")}
                {...form.getInputProps("apprise_title")}
              />
              <NativeSelect
                mt={10}
                label={t("drawer.appriseTypeLabel")}
                data={[
                  { value: AppriseType.Info, label: "Info" },
                  { value: AppriseType.Success, label: "Success" },
                  { value: AppriseType.Warning, label: "Warning" },
                  { value: AppriseType.Failure, label: "Failure" },
                ]}
                {...form.getInputProps("apprise_type")}
              />
              <TextInput
                mt={10}
                label={t("drawer.appriseTagLabel")}
                description={t("drawer.appriseTagDescription")}
                placeholder={t("drawer.appriseTagPlaceholder")}
                {...form.getInputProps("apprise_tag")}
              />
              <NativeSelect
                mt={10}
                label={t("drawer.appriseFormatLabel")}
                data={[
                  { value: AppriseFormat.Text, label: "Text" },
                  { value: AppriseFormat.HTML, label: "HTML" },
                  { value: AppriseFormat.Markdown, label: "Markdown" },
                ]}
                {...form.getInputProps("apprise_format")}
              />
            </>
          )}

          <Button mt={20} type="submit" fullWidth loading={createMutation.isPending || updateMutation.isPending}>
            {editingNotification ? t("drawer.submitUpdate") : t("drawer.submitCreate")}
          </Button>
        </form>
      </Drawer>

      {/* Delete Modal */}
      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title={t("deleteModal.title")}>
        {deletingNotification && (
          <div>
            <Text>
              {t("deleteModal.confirmText")} <strong>{deletingNotification.name}</strong>?
            </Text>
            <Flex mt={15} gap={10} justify="flex-end">
              <Button variant="default" onClick={closeDeleteModal}>{t("deleteModal.cancelButton")}</Button>
              <Button color="red" onClick={handleDelete} loading={deleteMutation.isPending}>
                {t("deleteModal.deleteButton")}
              </Button>
            </Flex>
          </div>
        )}
      </Modal>

      {/* Test Modal */}
      <Modal opened={testModalOpened} onClose={closeTestModal} title={t("testModal.title")}>
        {testingNotification && (
          <div>
            <Text mb={10}>
              {t("testModal.description")} <strong>{testingNotification.name}</strong> {t("testModal.descriptionEnd")}
            </Text>
            <NativeSelect
              label={t("testModal.eventTypeLabel")}
              data={[
                { value: NotificationEventType.VideoSuccess, label: t("testModal.eventVideoSuccess") },
                { value: NotificationEventType.LiveSuccess, label: t("testModal.eventLiveSuccess") },
                { value: NotificationEventType.Error, label: t("testModal.eventError") },
                { value: NotificationEventType.IsLive, label: t("testModal.eventIsLive") },
              ]}
              value={testEventType}
              onChange={(e) => setTestEventType(e.currentTarget.value as NotificationEventType)}
            />
            <Flex mt={15} gap={10} justify="flex-end">
              <Button variant="default" onClick={closeTestModal}>{t("testModal.cancelButton")}</Button>
              <Button color="violet" onClick={handleTest} loading={testMutation.isPending}>
                {t("testModal.sendButton")}
              </Button>
            </Flex>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default AdminNotificationsPage;
