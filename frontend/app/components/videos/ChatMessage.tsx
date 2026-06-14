/* eslint-disable @next/next/no-img-element */
import { Comment, GanymedeChatMessageKind, GanymedeFormattedBadge, GanymedeFormattedMessageFragment, GanymedeFormattedMessageType } from "@/app/hooks/useChat";
import { durationToTime } from "@/app/util/util";
import classes from "./ChatMessage.module.css"
import { Text, Tooltip } from "@mantine/core"
import { useTranslations } from "next-intl";
import { IconBolt, IconMessageCircle, IconStar } from "@tabler/icons-react";

interface Params {
  comment: Comment;
  showTimestamp: boolean;
  timestampSeconds: number | null;
  onTimestampClick: () => void;
}

const ChatMessage = ({ comment, showTimestamp, timestampSeconds, onTimestampClick }: Params) => {
  const t = useTranslations("VideoComponents");
  const hasTimestamp = timestampSeconds !== null;
  const timestampLabel = hasTimestamp ? durationToTime(Math.floor(timestampSeconds)) : "";
  const timestampActionLabel = hasTimestamp
    ? t("chatJumpToTimestamp", { timestamp: timestampLabel })
    : "";
  const messageKind = comment.ganymede_chat_message_kind ?? GanymedeChatMessageKind.Normal;
  const isEvent = messageKind === GanymedeChatMessageKind.UserNotice;
  const isHighlighted = messageKind === GanymedeChatMessageKind.Highlighted;
  const isAction = messageKind === GanymedeChatMessageKind.Action;
  const bitsSpent = comment.message.bits_spent ?? 0;
  const eventSystemMessage = isEvent ? comment.message.user_notice_params?.system_msg?.trim() : "";
  const eventUserMessageFromParams = isEvent ? comment.message.user_notice_params?.params?.["user-message"]?.trim() : "";
  const eventUserMessageFromBody = isEvent && eventSystemMessage && comment.message.body.startsWith(eventSystemMessage)
    ? comment.message.body.slice(eventSystemMessage.length).trim()
    : "";
  const eventUserMessage = eventUserMessageFromParams || eventUserMessageFromBody;
  const rowClassName = [
    classes.chatMessage,
    !showTimestamp ? classes.chatMessageNoTimestamp : "",
    isEvent ? classes.eventMessage : "",
    isHighlighted ? classes.highlightedMessage : "",
    isAction ? classes.actionMessage : "",
  ].filter(Boolean).join(" ");

  const renderFormattedMessage = () => (
    comment.ganymede_formatted_message && comment.ganymede_formatted_message.map(
      (fragment: GanymedeFormattedMessageFragment, index: number) => {
        switch (fragment.type) {
          case GanymedeFormattedMessageType.Text:
            return (
              <Text key={`${comment._id}-text-${index}`} className={classes.message} span>
                {fragment.text}
              </Text>
            )
          case GanymedeFormattedMessageType.Emote: {
            const emoteName = fragment.emote?.name || fragment.text;
            // some emotes include a height, use the provided height or hardcode a standard height if not included
            if ((fragment.emote?.height ?? 0) !== 0 && (fragment.emote?.width ?? 0) !== 0) {
              return (
                <Tooltip key={`${comment._id}-emote-${index}`} label={emoteName} position="top">
                  <img
                    src={fragment.url}
                    className={classes.emoteImage}
                    height={fragment.emote?.height}
                    alt={emoteName}
                  />
                </Tooltip>
              );
            } else {
              return (
                <Tooltip key={`${comment._id}-emote-${index}`} label={emoteName} position="top">
                  <img
                    src={fragment.url}
                    className={classes.emoteImage}
                    alt={emoteName}
                    height={28}
                  />
                </Tooltip>
              );
            }
          }

        }
      }
    )
  );

  return (
    <div key={comment._id} className={rowClassName}>
      {showTimestamp && (
        hasTimestamp ? (
          <button
            type="button"
            className={classes.timestamp}
            onClick={onTimestampClick}
            aria-label={timestampActionLabel}
            title={timestampActionLabel}
          >
            {timestampLabel}
          </button>
        ) : (
          <span className={classes.timestampPlaceholder} aria-hidden="true" />
        )
      )}
      <span className={classes.content}>
        {comment.message.reply && (
          <span className={classes.replyPreview}>
            <IconMessageCircle size={13} stroke={1.8} aria-hidden="true" />
            <Text className={classes.replyAuthor} span>
              {comment.message.reply.parent_display_name || comment.message.reply.parent_user_login}
            </Text>
            <Text className={classes.replyBody} span>
              {comment.message.reply.parent_msg_body}
            </Text>
          </span>
        )}
        {isEvent && (
          <span className={classes.eventLabel}>
            <IconStar size={13} stroke={1.9} aria-hidden="true" />
            {comment.ganymede_event_label || "Event"}
          </span>
        )}
        {/* badges */}
        <span>
          {comment.ganymede_formatted_badges &&
            comment.ganymede_formatted_badges.map(
              (badge: GanymedeFormattedBadge) => (
                badge.url && (
                  <Tooltip key={badge._id} label={badge.title} position="top">
                    <img
                      className={classes.badge}
                      src={badge.url}
                      height="18"
                      alt={badge.title}
                    />
                  </Tooltip>
                )
              )
            )}
        </span>
        {bitsSpent > 0 && (
          <span className={classes.bitsLabel}>
            <IconBolt size={12} stroke={2} aria-hidden="true" />
            {bitsSpent.toLocaleString()} bits
          </span>
        )}
        {/* username */}
        <Text
          fw={700}
          lh={1}
          size="sm"
          style={{ color: comment.message.user_color }}
          span
        >
          {comment.commenter.display_name}
        </Text>
        <Text className={classes.message} span>
          {isAction ? " " : ": "}
        </Text>
        {/* message */}
        {isEvent && eventSystemMessage ? (
          <span className={classes.eventBody}>
            <Text className={classes.eventSystemMessage} span>
              {eventSystemMessage}
            </Text>
            {eventUserMessage && (
              <Text className={classes.eventUserMessage} span>
                {eventUserMessage}
              </Text>
            )}
          </span>
        ) : (
          renderFormattedMessage()
        )}
      </span>

    </div>
  );
}

export default ChatMessage;
