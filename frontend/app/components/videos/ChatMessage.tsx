/* eslint-disable @next/next/no-img-element */
import { Comment, GanymedeFormattedBadge, GanymedeFormattedMessageFragment, GanymedeFormattedMessageType } from "@/app/hooks/useChat";
import { durationToTime } from "@/app/util/util";
import classes from "./ChatMessage.module.css"
import { Text, Tooltip } from "@mantine/core"
import { useTranslations } from "next-intl";

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

  return (
    <div key={comment._id} className={`${classes.chatMessage} ${!showTimestamp ? classes.chatMessageNoTimestamp : ""}`}>
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
          :{" "}
        </Text>
        {/* message */}
        {comment.ganymede_formatted_message && comment.ganymede_formatted_message.map(
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
        )}
      </span>

    </div>
  );
}

export default ChatMessage;
