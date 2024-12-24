/* eslint-disable @next/next/no-img-element */
import { Comment, GanymedeFormattedBadge, GanymedeFormattedMessageFragment, GanymedeFormattedMessageType } from "@/app/hooks/useChat";
import classes from "./ChatMessage.module.css"
import { Text, Tooltip } from "@mantine/core"

interface Params {
  comment: Comment;
}

const ChatMessage = ({ comment }: Params) => {
  return (
    <div key={comment._id} className={classes.chatMessage}>
      {/* badges */}
      <span>
        {comment.ganymede_formatted_badges &&
          comment.ganymede_formatted_badges.map(
            (badge: GanymedeFormattedBadge) => (
              <Tooltip key={badge._id} label={badge.title} position="top">
                <img
                  className={classes.badge}
                  src={badge.url}
                  height="18"
                  alt={badge.title}
                />
              </Tooltip>
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
        (fragment: GanymedeFormattedMessageFragment) => {
          switch (fragment.type) {
            case GanymedeFormattedMessageType.Text:
              return (
                <Text className={classes.message} span>
                  {fragment.text}
                </Text>
              )
            case GanymedeFormattedMessageType.Emote:
              const emoteName = fragment.emote?.name || fragment.text;
              // some emotes include a height, use the provided height or hardcode a standard height if not included
              if (fragment.emote?.height != 0 && fragment.emote?.width != 0) {
                return (
                  <Tooltip label={emoteName} position="top">
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
                  <Tooltip label={emoteName} position="top">
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
      )}

    </div>
  );
}

export default ChatMessage;