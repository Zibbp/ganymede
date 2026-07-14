import { Badge, Comment, Emote, useGetChatForChatterInVideo } from "@/app/hooks/useChat";
import {
  CloseButton,
  FloatingWindow,
  Group,
  ScrollArea,
  Text,
} from "@mantine/core";
import GanymedeLoadingText from "../utils/GanymedeLoadingText";
import { useTranslations } from "next-intl";
import ChatMessage from "./ChatMessage";
import { UseFloatingWindowOptions } from "@mantine/hooks"
import { RefObject, useEffect, useLayoutEffect, useMemo, useRef } from "react"
import { processComment } from "@/app/util/chat"

interface ChatMaps {
  emoteMap: Map<string, Emote>;
  thirdPartyEmoteMap: Map<string, Emote>;
  generalBadgeMap: Map<string, Badge>;
  subscriptionBadgeMap: Map<string, Badge>;
}

interface Params {
  videoId: string;
  chatterId: string;
  chatterName: string;
  initialScrollMessageId?: string;
  timestampSeconds: ((comment: Comment) => number | null) | null;
  onTimestampClick: (timestamp: number) => void;
  onClose: () => void;
  initialPosition: UseFloatingWindowOptions['initialPosition'];
  chatMapsRef: RefObject<ChatMaps>;
}

const ChatChatterMessages = ({
  videoId,
  chatterId,
  chatterName,
  initialScrollMessageId,
  timestampSeconds,
  onTimestampClick,
  onClose,
  initialPosition,
  chatMapsRef,
}: Params) => {
  const {
    data: comments,
    isLoading,
    isError,
  } = useGetChatForChatterInVideo(videoId, chatterId);
  const t = useTranslations("VideoComponents");
  const messagesContainerRef = useRef<HTMLDivElement>(null);

  const processedComments = useMemo(() => {
    if (!comments) return null;
    return comments.map((comment) => processComment(
      comment,
      chatMapsRef.current.subscriptionBadgeMap,
      chatMapsRef.current.generalBadgeMap,
      chatMapsRef.current.emoteMap,
      chatMapsRef.current.thirdPartyEmoteMap,
      (error) => {
        console.error(error);
      }
    ));
  }, [comments, chatMapsRef.current.emoteMap, chatMapsRef.current.thirdPartyEmoteMap, chatMapsRef.current.generalBadgeMap, chatMapsRef.current.subscriptionBadgeMap]);

  useEffect(() => {
    const handle = requestAnimationFrame(() => {
      if (!messagesContainerRef.current || !initialScrollMessageId || !processedComments?.length) return;
      const messageElement = messagesContainerRef.current.querySelector(`[data-message-id="${CSS.escape(initialScrollMessageId)}"]`);
      if (messageElement && 'scrollIntoView' in messageElement) {
        messageElement.scrollIntoView({ behavior: "smooth" });
      }
    })
    return () => cancelAnimationFrame(handle);
  }, [isLoading, processedComments, initialScrollMessageId]);

  return (
    <FloatingWindow
      w={340}
      withBorder
      dragHandleSelector=".drag-handle"
      initialPosition={initialPosition}
      constrainToViewport={true}
    >
      <Group
        justify="space-between"
        px="md"
        py="sm"
        className="drag-handle"
        style={{ cursor: "move" }}
      >
        <Text>{t("chatterMessages", { name: chatterName })}</Text>
        <CloseButton onClick={onClose} />
      </Group>
      <ScrollArea.Autosize mah={300} px="md" pb="sm" ref={messagesContainerRef}>
        {isLoading && <GanymedeLoadingText message={t("loadingChatterMessages")} />}
        {!isLoading && isError && <Text color="red">{t("chatError")}</Text>}
        {!isLoading &&
          !isError &&
          (!processedComments || processedComments.length === 0 ? (
            <Text>{t("noChat")}</Text>
          ) : (
            processedComments.map((comment) => (
              <ChatMessage
                key={comment._id}
                highlightAnimation={comment._id === initialScrollMessageId}
                comment={comment}
                showTimestamp={true}
                timestampSeconds={timestampSeconds?.(comment) ?? null}
                onTimestampClick={() => {
                  onTimestampClick(comment.content_offset_seconds);
                }}
              />
            ))
          ))}
      </ScrollArea.Autosize>
    </FloatingWindow>
  );
};

export default ChatChatterMessages;
