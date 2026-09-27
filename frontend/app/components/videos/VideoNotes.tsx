"use client"
import { RefObject, useRef, useState } from "react";
import { ActionIcon, Button, Group, Text, Textarea, Title, Tooltip, Typography } from "@mantine/core";
import { showNotification } from "@mantine/notifications";
import { IconClock, IconX } from "@tabler/icons-react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { useTranslations } from "next-intl";
import { UserRole } from "@/app/hooks/useAuthentication";
import { useAxiosPrivate } from "@/app/hooks/useAxios";
import { Video, useUpdateVideoNotes } from "@/app/hooks/useVideos";
import useAuthStore from "@/app/store/useAuthStore";
import { durationToTime } from "@/app/util/util";
import { parseTimestampHref } from "@/app/util/videoNotes";
import classes from "./VideoNotes.module.css";

const MAX_NOTES_LENGTH = 10000;

type Props = {
  video: Video;
  playerRef: RefObject<HTMLVideoElement | null>;
  docked?: boolean;
  onEdit?: () => void;
  onClose?: () => void;
};

// External images are not auto-loaded: stored notes are viewed by other users
// (or anonymously when login is not required), so auto-fetching http(s) image
// URLs would disclose viewer IP/UA/Referer to a third-party host.
// External images render as click-to-open links instead.
const isExternalImageSrc = (src?: string): boolean => {
  if (!src) return false;
  const trimmed = src.trim().toLowerCase();
  return (
    trimmed.startsWith("http://") ||
    trimmed.startsWith("https://") ||
    trimmed.startsWith("//")
  );
};

const VideoNotes = ({ video, playerRef, docked = false, onEdit, onClose }: Props) => {
  const t = useTranslations("VideoComponents");
  const axiosPrivate = useAxiosPrivate();
  const hasPermission = useAuthStore((state) => state.hasPermission);
  const canEdit = hasPermission(UserRole.Editor);

  const [editing, setEditing] = useState(docked);
  const [draft, setDraft] = useState(video.notes ?? "");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const updateNotesMutate = useUpdateVideoNotes();

  const handleEdit = () => {
    setDraft(video.notes ?? "");
    if (onEdit) {
      onEdit();
    } else {
      setEditing(true);
    }
  };

  const handleCancel = () => {
    setDraft(video.notes ?? "");
    if (docked) {
      onClose?.();
    } else {
      setEditing(false);
    }
  };

  const handleSave = async () => {
    try {
      await updateNotesMutate.mutateAsync({
        axiosPrivate,
        videoId: video.id,
        notes: draft,
      });
      showNotification({
        message: t("notesSavedNotification"),
      });
      if (docked) {
        onClose?.();
      } else {
        setEditing(false);
      }
    } catch (error) {
      console.error(error);
      showNotification({
        color: "red",
        message: t("notesSaveError"),
      });
    }
  };

  const handleSeek = (seconds: number) => {
    if (!playerRef.current) {
      showNotification({
        color: "red",
        message: t("notesPlayerNotReady"),
      });
      return;
    }
    playerRef.current.currentTime = Math.max(0, seconds);
    // Notes sit below the player, so bring the video back into view.
    window.scrollTo({ top: 0, behavior: "smooth" });
  };

  const handleInsertTimestamp = () => {
    const seconds = Math.floor(playerRef.current?.currentTime ?? NaN);
    if (!Number.isFinite(seconds)) {
      showNotification({
        color: "red",
        message: t("notesPlayerNotReady"),
      });
      return;
    }
    const descriptionPlaceholder = t("notesTimestampDescriptionPlaceholder");
    const snippet = `[${descriptionPlaceholder} — ${durationToTime(seconds)}](#t=${seconds})`;
    const el = textareaRef.current;
    const selectionStart = el?.selectionStart ?? draft.length;
    const selectionEnd = el?.selectionEnd ?? draft.length;
    const before = draft.slice(0, selectionStart);
    const after = draft.slice(selectionEnd);
    const spaceBefore = before && !/\s$/.test(before) ? " " : "";
    const spaceAfter = after && !/^\s/.test(after) ? " " : "";
    const next = `${before}${spaceBefore}${snippet}${spaceAfter}${after}`;
    if (next.length > MAX_NOTES_LENGTH) {
      showNotification({
        color: "red",
        message: t("notesTooLongError", { max: MAX_NOTES_LENGTH }),
      });
      return;
    }
    setDraft(next);
    // Select the placeholder so the user can immediately type a description.
    requestAnimationFrame(() => {
      if (!textareaRef.current) return;
      const descriptionStart = before.length + spaceBefore.length + 1;
      const descriptionEnd = descriptionStart + descriptionPlaceholder.length;
      textareaRef.current.focus();
      textareaRef.current.setSelectionRange(descriptionStart, descriptionEnd);
    });
  };

  const notes = video.notes ?? "";
  const isDirty = draft !== notes;

  // Hide entirely for viewers when there are no notes and they cannot edit.
  if (!notes && !canEdit && !editing) {
    return null;
  }

  return (
    <div className={docked ? classes.dockedPanel : classes.inlineNotes}>
      <Group justify="space-between" mb={5}>
        <Title order={docked ? 3 : 1}>{t("notesTitle")}</Title>
        {docked && (
          <Tooltip label={t("notesCancelButton")}>
            <ActionIcon variant="subtle" onClick={handleCancel} aria-label={t("notesCancelButton")}>
              <IconX size={18} />
            </ActionIcon>
          </Tooltip>
        )}
        {canEdit && !editing && !docked && (
          <Button variant="default" size="xs" onClick={handleEdit}>
            {notes ? t("notesEditButton") : t("notesAddButton")}
          </Button>
        )}
      </Group>

      {editing ? (
        <div>
          <Textarea
            ref={textareaRef}
            placeholder={t("notesPlaceholder")}
            value={draft}
            onChange={(event) => setDraft(event.currentTarget.value)}
            autosize
            minRows={4}
            maxRows={12}
            maxLength={MAX_NOTES_LENGTH}
            disabled={updateNotesMutate.isPending}
            autoFocus={docked}
          />
          <Group justify="space-between" mt={8}>
            <Text size="xs" color="dimmed">
              {t("notesCharCount", { count: draft.length, max: MAX_NOTES_LENGTH })}
            </Text>
            <Group gap={8}>
              <Button
                variant="default"
                size="xs"
                leftSection={<IconClock size={14} />}
                onClick={handleInsertTimestamp}
                disabled={updateNotesMutate.isPending}
              >
                {t("notesInsertTimestampButton")}
              </Button>
              <Button variant="default" size="xs" onClick={handleCancel} disabled={updateNotesMutate.isPending}>
                {t("notesCancelButton")}
              </Button>
              <Button
                size="xs"
                onClick={handleSave}
                loading={updateNotesMutate.isPending}
                disabled={!isDirty}
              >
                {t("notesSaveButton")}
              </Button>
            </Group>
          </Group>
          <Text size="xs" color="dimmed" mt={4}>
            {t("notesMarkdownHint")}
          </Text>
        </div>
      ) : notes ? (
        <Typography>
          <ReactMarkdown
            remarkPlugins={[remarkGfm, remarkBreaks]}
            components={{
              a: ({ href, children }) => {
                const seconds = parseTimestampHref(href);
                if (seconds !== null) {
                  return (
                    <button
                      type="button"
                      className={classes.timestampLink}
                      onClick={() => handleSeek(seconds)}
                      title={t("notesJumpToTimestamp")}
                    >
                      {children}
                    </button>
                  );
                }
                return (
                  <a
                    href={href}
                    target="_blank"
                    rel="noreferrer"
                    className={classes.markdownLink}
                  >
                    {children}
                  </a>
                );
              },
              img: ({ src, alt, title }) => {
                if (!src) return null;
                if (isExternalImageSrc(src)) {
                  return (
                    <a
                      href={src}
                      target="_blank"
                      rel="noreferrer"
                      className={classes.markdownLink}
                    >
                      {alt || src}
                    </a>
                  );
                }
                return (
                  <img
                    src={src}
                    alt={alt}
                    title={title}
                    loading="lazy"
                    referrerPolicy="no-referrer"
                    style={{ maxWidth: "100%" }}
                  />
                );
              },
            }}
          >
            {notes}
          </ReactMarkdown>
        </Typography>
      ) : (
        <Text size="sm" color="dimmed">
          {t("notesEmpty")}
        </Text>
      )}
    </div>
  );
};

export default VideoNotes;
