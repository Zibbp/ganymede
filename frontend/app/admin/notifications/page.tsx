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

const TemplateVariableHints = ({ triggerKey }: { triggerKey: string }) => {
  const groups = TRIGGER_VARIABLES[triggerKey];
  if (!groups) return null;
  return (
    <Box mt={4} ml={28}>
      <Text size="xs" c="dimmed">
        Variables:{" "}
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
  usePageTitle("Notifications");

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

      // Name required
      if (!values.name.trim()) {
        errors.name = "Name is required";
      }

      // URL required and must be a valid URL
      if (!values.url.trim()) {
        errors.url = "URL is required";
      } else {
        try {
          const url = new URL(values.url);
          if (!["http:", "https:"].includes(url.protocol)) {
            errors.url = "URL must start with http:// or https://";
          }
        } catch {
          errors.url = "Must be a valid URL";
        }
      }

      // At least one trigger must be enabled
      if (!values.trigger_video_success && !values.trigger_live_success && !values.trigger_error && !values.trigger_is_live) {
        errors.trigger_video_success = "At least one trigger must be enabled";
      }

      // Templates required for enabled triggers
      if (values.trigger_video_success && !values.video_success_template.trim()) {
        errors.video_success_template = "Template is required when trigger is enabled";
      }
      if (values.trigger_live_success && !values.live_success_template.trim()) {
        errors.live_success_template = "Template is required when trigger is enabled";
      }
      if (values.trigger_error && !values.error_template.trim()) {
        errors.error_template = "Template is required when trigger is enabled";
      }
      if (values.trigger_is_live && !values.is_live_template.trim()) {
        errors.is_live_template = "Template is required when trigger is enabled";
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
          title: "Success",
          message: "Notification updated",
        });
      } else {
        await createMutation.mutateAsync({
          axiosPrivate,
          input: values,
        });
        showNotification({
          title: "Success",
          message: "Notification created",
        });
      }
      closeDrawer();
    } catch (error) {
      showNotification({
        title: "Error",
        message: `Failed to ${editingNotification ? "update" : "create"} notification`,
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
        title: "Success",
        message: "Notification deleted",
      });
      closeDeleteModal();
    } catch (error) {
      showNotification({
        title: "Error",
        message: "Failed to delete notification",
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
        title: "Success",
        message: "Test notification sent",
      });
      closeTestModal();
    } catch (error) {
      showNotification({
        title: "Error",
        message: "Failed to send test notification",
        color: "red",
      });
    }
  };

  if (isPending) return <GanymedeLoadingText message="Loading notifications..." />;
  if (isError) return <div>Error loading notifications</div>;

  return (
    <div>
      <Container size="7xl">
        <Group justify="space-between" mt={2}>
          <Title>Notifications</Title>
          <Button onClick={handleOpenCreate} variant="default">
            Create Notification
          </Button>
        </Group>

        <Text mt={5} mb={10}>
          Configure notification destinations. Each notification can subscribe to one or more event types
          and be sent via webhook or Apprise. Visit the{" "}
          <a href="https://github.com/Zibbp/ganymede/wiki/Notifications" target="_blank">
            wiki
          </a>{" "}
          for more information.
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
                title: "Name",
              },
              {
                accessor: "type",
                title: "Type",
                render: (n) => (
                  <Badge color={n.type === NotificationType.Apprise ? "grape" : "blue"} variant="light">
                    {n.type}
                  </Badge>
                ),
              },
              {
                accessor: "enabled",
                title: "Enabled",
                render: (n) => (
                  <Badge color={n.enabled ? "green" : "gray"} variant="light">
                    {n.enabled ? "Yes" : "No"}
                  </Badge>
                ),
              },
              {
                accessor: "triggers",
                title: "Triggers",
                render: (n) => {
                  const triggers = [];
                  if (n.trigger_video_success) triggers.push("Video Success");
                  if (n.trigger_live_success) triggers.push("Live Success");
                  if (n.trigger_error) triggers.push("Error");
                  if (n.trigger_is_live) triggers.push("Is Live");
                  return (
                    <Group gap={4}>
                      {triggers.map((t) => (
                        <Badge key={t} size="xs" variant="outline">
                          {t}
                        </Badge>
                      ))}
                      {triggers.length === 0 && (
                        <Text size="xs" c="dimmed">None</Text>
                      )}
                    </Group>
                  );
                },
              },
              {
                accessor: "actions",
                title: "Actions",
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
        title={editingNotification ? "Edit Notification" : "Create Notification"}
      >
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <TextInput
            label="Name"
            placeholder={form.values.type === NotificationType.Apprise ? "My Apprise Notification" : "My Discord Webhook"}
            required
            {...form.getInputProps("name")}
          />
          <Checkbox
            mt={10}
            label="Enabled"
            {...form.getInputProps("enabled", { type: "checkbox" })}
          />
          <NativeSelect
            mt={10}
            label="Type"
            required
            data={[
              { value: NotificationType.Webhook, label: "Webhook" },
              { value: NotificationType.Apprise, label: "Apprise" },
            ]}
            {...form.getInputProps("type")}
          />
          <TextInput
            mt={10}
            label={form.values.type === NotificationType.Apprise ? "Apprise API URL" : "Webhook URL"}
            description={
              form.values.type === NotificationType.Apprise
                ? "The URL of your Apprise API instance (e.g. http://apprise:8000/notify/ for stateful or http://apprise:8000/notify for stateless)"
                : "The webhook URL to send notifications to (e.g. Discord, Slack)"
            }
            placeholder={
              form.values.type === NotificationType.Apprise
                ? "http://apprise:8000/notify"
                : "https://discord.com/api/webhooks/..."
            }
            required
            {...form.getInputProps("url")}
          />

          <Text fw={700} size="sm" mt={20}>
            Event Triggers <Text component="span" c="red" size="sm">*</Text>
          </Text>
          <Text size="xs" c="dimmed">Select at least one event. A message template is required for each enabled trigger.</Text>
          {form.errors.trigger_video_success && (
            <Text size="xs" c="red" mt={2}>{form.errors.trigger_video_success}</Text>
          )}

          <Checkbox
            mt={10}
            label="Video Archive Success"
            {...form.getInputProps("trigger_video_success", { type: "checkbox" })}
          />
          {form.values.trigger_video_success && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label="Message"
                required
                {...form.getInputProps("video_success_template")}
              />
              <TemplateVariableHints triggerKey="video_success" />
            </>
          )}

          <Checkbox
            mt={10}
            label="Live Archive Success"
            {...form.getInputProps("trigger_live_success", { type: "checkbox" })}
          />
          {form.values.trigger_live_success && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label="Message"
                required
                {...form.getInputProps("live_success_template")}
              />
              <TemplateVariableHints triggerKey="live_success" />
            </>
          )}

          <Checkbox
            mt={10}
            label="Error"
            {...form.getInputProps("trigger_error", { type: "checkbox" })}
          />
          {form.values.trigger_error && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label="Message"
                required
                {...form.getInputProps("error_template")}
              />
              <TemplateVariableHints triggerKey="error" />
            </>
          )}

          <Checkbox
            mt={10}
            label="Channel Is Live"
            {...form.getInputProps("trigger_is_live", { type: "checkbox" })}
          />
          {form.values.trigger_is_live && (
            <>
              <Textarea
                mt={5}
                ml={28}
                label="Message"
                required
                {...form.getInputProps("is_live_template")}
              />
              <TemplateVariableHints triggerKey="is_live" />
            </>
          )}

          {/* Apprise-specific fields */}
          {form.values.type === NotificationType.Apprise && (
            <>
              <Title order={4} mt={20}>Apprise Settings</Title>
              <Text size="sm" c="dimmed">
                These fields are only used when the type is Apprise. Visit the{" "}
                <a href="https://github.com/caronc/apprise-api" target="_blank">
                  Apprise API documentation
                </a>{" "}
                for more information.
              </Text>
              <TextInput
                mt={10}
                label="Apprise URLs (stateless mode)"
                description="Comma-separated Apprise notification URLs for stateless mode"
                placeholder="discord://webhook_id/webhook_token"
                {...form.getInputProps("apprise_urls")}
              />
              <TextInput
                mt={10}
                label="Apprise Title Template"
                description="Supports the same template variables as body templates"
                placeholder="{{channel_display_name}} - Notification"
                {...form.getInputProps("apprise_title")}
              />
              <NativeSelect
                mt={10}
                label="Apprise Notification Type"
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
                label="Apprise Tag (stateful mode)"
                description="Tag for Apprise stateful configurations"
                placeholder="all"
                {...form.getInputProps("apprise_tag")}
              />
              <NativeSelect
                mt={10}
                label="Apprise Format"
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
            {editingNotification ? "Update" : "Create"}
          </Button>
        </form>
      </Drawer>

      {/* Delete Modal */}
      <Modal opened={deleteModalOpened} onClose={closeDeleteModal} title="Delete Notification">
        {deletingNotification && (
          <div>
            <Text>
              Are you sure you want to delete the notification <strong>{deletingNotification.name}</strong>?
            </Text>
            <Flex mt={15} gap={10} justify="flex-end">
              <Button variant="default" onClick={closeDeleteModal}>Cancel</Button>
              <Button color="red" onClick={handleDelete} loading={deleteMutation.isPending}>
                Delete
              </Button>
            </Flex>
          </div>
        )}
      </Modal>

      {/* Test Modal */}
      <Modal opened={testModalOpened} onClose={closeTestModal} title="Test Notification">
        {testingNotification && (
          <div>
            <Text mb={10}>
              Send a test notification to <strong>{testingNotification.name}</strong> with dummy data.
            </Text>
            <NativeSelect
              label="Event Type"
              data={[
                { value: NotificationEventType.VideoSuccess, label: "Video Archive Success" },
                { value: NotificationEventType.LiveSuccess, label: "Live Archive Success" },
                { value: NotificationEventType.Error, label: "Error" },
                { value: NotificationEventType.IsLive, label: "Channel Is Live" },
              ]}
              value={testEventType}
              onChange={(e) => setTestEventType(e.currentTarget.value as NotificationEventType)}
            />
            <Flex mt={15} gap={10} justify="flex-end">
              <Button variant="default" onClick={closeTestModal}>Cancel</Button>
              <Button color="violet" onClick={handleTest} loading={testMutation.isPending}>
                Send Test
              </Button>
            </Flex>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default AdminNotificationsPage;
