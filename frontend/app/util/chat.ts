import { Badge, Comment, Emote, GanymedeChatMessageKind, GanymedeFormattedBadge, GanymedeFormattedMessageFragment, GanymedeFormattedMessageType } from "../hooks/useChat"

const IGNORED_BADGES: ReadonlySet<string> = new Set(['no_audio', 'no_video', 'predictions']);
const SUBSCRIPTION_BADGES: ReadonlySet<string> = new Set(['subscriber', 'sub-gifter', 'sub_gifter', 'bits']);
const EVENT_LABELS: Record<string, string> = {
  sub: "chatEventSub",
  resub: "chatEventResub",
  subgift: "chatEventGiftSub",
  anonsubgift: "chatEventGiftSub",
  submysterygift: "chatEventGiftBomb",
  raid: "chatEventRaid",
  unraid: "chatEventRaid",
  ritual: "chatEventRitual",
  announcement: "chatEventAnnouncement",
};
const FIRST_MESSAGE_LABEL = "chatEventFirstMessage";

interface ChatProcessingMaps {
  subscriptionBadgeMap: Map<string, Badge>;
  generalBadgeMap: Map<string, Badge>;
  emoteMap: Map<string, Emote>;
  thirdPartyEmoteMap: Map<string, Emote>;
}

const getBadgesToFormattedComment = (
  comment: Comment,
  subscriptionBadgeMap: Map<string, Badge>,
  generalBadgeMap: Map<string, Badge>
): GanymedeFormattedBadge[] => {
  if (!comment.message.user_badges) {
    return [];
  }

  return comment.message.user_badges
    .filter(badge => !IGNORED_BADGES.has(badge._id))
    .map(badge => {
      const isSubscriptionBadge = SUBSCRIPTION_BADGES.has(badge._id);
      const badgeMap = isSubscriptionBadge
        ? subscriptionBadgeMap.get(badge.version)
        : generalBadgeMap.get(badge._id);

      return {
        _id: badge._id,
        id: badge._id,
        version: badge.version,
        title: badgeMap?.title || '',
        url: badgeMap?.image_url_1x || '',
      };
    });
}

const getEmotesToFormattedComment = (
  comment: Comment,
  emoteMap: Map<string, Emote>,
  thirdPartyEmoteMap: Map<string, Emote>,
  onError: (error: Error) => void,
): GanymedeFormattedMessageFragment[] => {
  if (!comment.message.fragments?.length) return [];

  try {
    return comment.message.fragments.flatMap(fragment => {
      if (fragment.emoticon) {
        const emote = emoteMap.get(fragment.emoticon.emoticon_id);
        if (!emote) {
          throw new Error(`Emote not found for ID: ${fragment.emoticon.emoticon_id}`);
        }
        return [{
          type: GanymedeFormattedMessageType.Emote,
          text: fragment.text,
          url: emote.type === "embed"
            ? `data:image/png;base64,${emote.url}`
            : emote.url,
          emote,
        }];
      }

      return fragment.text.split(" ").map(word => {
        const thirdPartyEmote = thirdPartyEmoteMap.get(word);
        return thirdPartyEmote
          ? {
            type: GanymedeFormattedMessageType.Emote,
            text: word,
            url: thirdPartyEmote.type === "embed"
              ? `data:image/png;base64,${thirdPartyEmote.url}`
              : thirdPartyEmote.url,
            emote: thirdPartyEmote,
          }
          : {
            type: GanymedeFormattedMessageType.Text,
            text: ` ${word} `,
          };
      });
    });
  } catch (error) {
    onError(error as Error);
    return [{ type: GanymedeFormattedMessageType.Text, text: comment.message.body }];
  }
}

const classifyComment = (comment: Comment): { ganymede_chat_message_kind: GanymedeChatMessageKind, ganymede_event_label?: string } => {
  const msgID = comment.message.user_notice_params?.msg_id;
  const noticeID = typeof msgID === "string" ? msgID : "";
  const noticeParams = comment.message.user_notice_params?.params ?? {};
  const isFirstMessage = comment.message.is_first_message
    || comment.message.user_badges?.some(badge => badge._id === "first-msg")
    || (
      noticeID === "ritual"
      && noticeParams["msg-param-ritual-name"] === "new_chatter"
    );

  if (noticeID && noticeID !== "highlighted-message") {
    return {
      ganymede_chat_message_kind: GanymedeChatMessageKind.UserNotice,
      ganymede_event_label: isFirstMessage
        ? FIRST_MESSAGE_LABEL
        : EVENT_LABELS[noticeID] ?? "chatEventGeneric",
    };
  }

  if (noticeID === "highlighted-message") {
    return {
      ganymede_chat_message_kind: GanymedeChatMessageKind.Highlighted,
    };
  }

  if (isFirstMessage) {
    return {
      ganymede_chat_message_kind: GanymedeChatMessageKind.FirstMessage,
      ganymede_event_label: FIRST_MESSAGE_LABEL,
    };
  }

  if (comment.message.is_action) {
    return {
      ganymede_chat_message_kind: GanymedeChatMessageKind.Action,
    };
  }

  if ((comment.message.bits_spent ?? 0) > 0) {
    return {
      ganymede_chat_message_kind: GanymedeChatMessageKind.Bits,
    };
  }

  return {
    ganymede_chat_message_kind: GanymedeChatMessageKind.Normal,
  };
};

const processComment = (
  comment: Comment,
  maps: ChatProcessingMaps,
  onError: (error: Error) => void,
): Comment => {
  return {
    ...comment,
    ganymede_formatted_badges: getBadgesToFormattedComment(
      comment,
      maps.subscriptionBadgeMap,
      maps.generalBadgeMap,
    ),
    ganymede_formatted_message: getEmotesToFormattedComment(
      comment,
      maps.emoteMap,
      maps.thirdPartyEmoteMap,
      onError,
    ),
    ...classifyComment(comment),
  }
}

export {
  IGNORED_BADGES,
  processComment,
}

export type { ChatProcessingMaps }
