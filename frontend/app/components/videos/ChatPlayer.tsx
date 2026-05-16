import { Video, VideoType } from "@/app/hooks/useVideos";
import classes from "./ChatPlayer.module.css";
import {
  Emote,
  Badge,
  useGetEmotesForVideo,
  useGetBadgesForVideo,
  Comment,
  GanymedeFormattedMessageFragment,
  GanymedeFormattedMessageType,
  getChatForVideo,
  getSeekChatForVideo
} from "@/app/hooks/useChat";
import { RefObject, useEffect, useRef, useState, useCallback, useMemo } from "react";
import { Box, Center, Loader, Text } from "@mantine/core";
import ChatMessage from "./ChatMessage";
import { uuidv4 } from "@/app/util/util";
import VideoEventBus from "@/app/util/VideoEventBus";
import useSettingsStore from "@/app/store/useSettingsStore";
import { useTranslations } from "next-intl";
import { MediaPlayerInstance } from "@vidstack/react";

// Constants moved to top level
const CHAT_OFFSET_SIZE = 10;
const MAX_CHAT_MESSAGES = 50;
const TICK_INTERVAL = 100;
const TIME_SKIP_THRESHOLD = 2;
const IGNORED_BADGES = new Set(['no_audio', 'no_video', 'predictions']);
const SUBSCRIPTION_BADGES = new Set(['subscriber', 'sub-gifter', 'sub_gifter', 'bits']);
const SCROLL_THRESHOLD = 100; // px from bottom to trigger auto-scroll

interface ChatMaps {
  emoteMap: Map<string, Emote>;
  thirdPartyEmoteMap: Map<string, Emote>;
  generalBadgeMap: Map<string, Badge>;
  subscriptionBadgeMap: Map<string, Badge>;
  subscriptionGiftBadgeMap: Map<string, Badge>;
  bitBadgeMap: Map<string, Badge>;
}

interface Params {
  video: Video;
  playerRef: RefObject<MediaPlayerInstance | null>;
}

interface ChatError {
  message: string;
  timestamp: number;
}

interface PendingChatRange {
  start: number;
  end: number;
  generation: number;
}

const ChatPlayer = ({ video, playerRef }: Params) => {
  const t = useTranslations('VideoComponents')
  const [isReady, setIsReady] = useState(false);
  const [messages, setMessages] = useState<Comment[]>([]);
  const [error, setError] = useState<ChatError | null>(null);
  const [shouldAutoScroll, setShouldAutoScroll] = useState(true);

  // Refs for internal state management
  const internalMessagesRef = useRef<Comment[]>([]);
  const lastTimeRef = useRef(0);
  const lastCheckTimeRef = useRef(0);
  const pendingRangeRef = useRef<PendingChatRange | null>(null);
  const isSeekLoadingRef = useRef(false);
  const requestGenerationRef = useRef(0);
  const retryCountRef = useRef(0);
  const queuedIdsRef = useRef<Set<string>>(new Set());
  const processedIdsRef = useRef<Set<string>>(new Set());
  const processedIdsOrderRef = useRef<string[]>([]);
  const chatMapsRef = useRef<ChatMaps>({
    emoteMap: new Map(),
    thirdPartyEmoteMap: new Map(),
    generalBadgeMap: new Map(),
    subscriptionBadgeMap: new Map(),
    subscriptionGiftBadgeMap: new Map(),
    bitBadgeMap: new Map()
  });
  const chatContainerRef = useRef<HTMLDivElement>(null);

  const { chatPlaybackSmoothScroll, showChatTimestamps } = useSettingsStore()
  const clipVodOffset = useMemo<number | null>(() => {
    const offset = video.clip_vod_offset;
    return video.type === VideoType.Clip && typeof offset === "number" && Number.isFinite(offset)
      ? offset
      : null;
  }, [video.clip_vod_offset, video.type]);

  // Custom hooks with error handling
  const { data: chatEmotes, error: emotesError } = useGetEmotesForVideo(video.id);
  const { data: chatBadges, error: badgesError } = useGetBadgesForVideo(video.id);

  const scrollToBottom = useCallback((smooth = chatPlaybackSmoothScroll) => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTo({
        top: chatContainerRef.current.scrollHeight,
        behavior: smooth ? 'smooth' : 'auto'
      });
    }
  }, [chatPlaybackSmoothScroll]);

  const checkShouldScroll = useCallback(() => {
    if (!chatContainerRef.current) return true;

    const { scrollHeight, scrollTop, clientHeight } = chatContainerRef.current;
    const distanceFromBottom = scrollHeight - (scrollTop + clientHeight);

    return distanceFromBottom <= SCROLL_THRESHOLD;
  }, []);

  // Scroll event handler
  const handleScroll = useCallback(() => {
    if (!chatContainerRef.current) return;

    const shouldScroll = checkShouldScroll();
    if (shouldScroll !== shouldAutoScroll) {
      setShouldAutoScroll(shouldScroll);
    }
  }, [checkShouldScroll, shouldAutoScroll]);

  // Modified message setter with scroll behavior
  const setMessagesWithScroll = useCallback((newMessages: Comment[] | ((prev: Comment[]) => Comment[])) => {
    setMessages(prev => {
      const nextMessages = typeof newMessages === 'function' ? newMessages(prev) : newMessages;

      // Schedule scroll after render if auto-scroll is enabled
      if (shouldAutoScroll) {
        requestAnimationFrame(() => scrollToBottom());
      }

      return nextMessages;
    });
  }, [shouldAutoScroll, scrollToBottom]);

  // Error handling utility
  const handleError = useCallback((error: Error, context: string) => {
    console.error(`Error in ${context}:`, error);
    setError({ message: `${context}: ${error.message}`, timestamp: Date.now() });

    // Implement exponential backoff for retries
    const maxRetries = 3;
    if (retryCountRef.current < maxRetries) {
      const backoffTime = Math.pow(2, retryCountRef.current) * 1000;
      setTimeout(() => {
        retryCountRef.current++;
        // Reset error if successful
        setError(null);
      }, backoffTime);
    }
  }, []);

  // Memoized system message creator
  const createSystemMessage = useMemo(() => (message: string): Comment => ({
    _id: uuidv4(),
    content_offset_seconds: 0,
    // @ts-expect-error additional fields unnecessary
    commenter: {
      display_name: "Ganymede",
    },
    // @ts-expect-error additional fields unnecessary
    message: {
      body: message,
      user_color: "#a65ee8",
    },
    ganymede_formatted_message: [{
      type: GanymedeFormattedMessageType.Text,
      text: message
    }]
  }), []);

  const addCustomComment = useCallback((message: string) => {
    const comment = createSystemMessage(message);
    setMessagesWithScroll(prev => [...prev, comment]);
  }, [createSystemMessage, setMessagesWithScroll]);

  const seekToComment = useCallback((seconds: number) => {
    if (!playerRef.current) return;
    if (!Number.isFinite(seconds)) return;

    const playerTime = clipVodOffset !== null
      ? seconds - clipVodOffset
      : seconds;

    if (!Number.isFinite(playerTime)) return;

    playerRef.current.currentTime = Math.max(0, playerTime);
  }, [clipVodOffset, playerRef]);

  const getCommentTimestampSeconds = useCallback((comment: Comment): number | null => {
    if (!Number.isFinite(comment.content_offset_seconds)) return null;

    if (clipVodOffset !== null) {
      const timestampSeconds = comment.content_offset_seconds - clipVodOffset;
      if (!Number.isFinite(timestampSeconds)) return null;

      return Math.max(0, timestampSeconds);
    }

    return comment.content_offset_seconds;
  }, [clipVodOffset]);

  // Optimized badge processing
  const addBadgesToFormattedComment = useCallback((comment: Comment) => {
    if (!comment.message.user_badges) {
      comment.ganymede_formatted_badges = [];
      return comment;
    }

    comment.ganymede_formatted_badges = comment.message.user_badges
      .filter(badge => !IGNORED_BADGES.has(badge._id))
      .map(badge => {
        const isSubscriptionBadge = SUBSCRIPTION_BADGES.has(badge._id);
        const badgeMap = isSubscriptionBadge
          ? chatMapsRef.current.subscriptionBadgeMap.get(badge.version)
          : chatMapsRef.current.generalBadgeMap.get(badge._id);

        return {
          _id: badge._id,
          id: badge._id,
          version: badge.version,
          title: badgeMap?.title || '',
          url: badgeMap?.image_url_1x || '',
        };
      });

    return comment;
  }, []);

  // Optimized emote processing with error handling
  const addEmotesToFormattedComment = useCallback((comment: Comment): GanymedeFormattedMessageFragment[] => {
    if (!comment.message.fragments?.length) return [];

    try {
      return comment.message.fragments.flatMap(fragment => {
        if (fragment.emoticon) {
          const emote = chatMapsRef.current.emoteMap.get(fragment.emoticon.emoticon_id);
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
          const thirdPartyEmote = chatMapsRef.current.thirdPartyEmoteMap.get(word);
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
      handleError(error as Error, "Emote processing");
      return [{ type: GanymedeFormattedMessageType.Text, text: comment.message.body }];
    }
  }, [handleError]);

  const enqueueComments = useCallback((comments?: Comment[]) => {
    if (!comments?.length) return;

    const nextMessages = [...internalMessagesRef.current];
    let addedMessage = false;

    comments.forEach((comment) => {
      if (!comment._id) return;
      if (queuedIdsRef.current.has(comment._id) || processedIdsRef.current.has(comment._id)) return;

      queuedIdsRef.current.add(comment._id);
      nextMessages.push(comment);
      addedMessage = true;
    });

    if (!addedMessage) return;

    nextMessages.sort((a, b) => a.content_offset_seconds - b.content_offset_seconds);
    internalMessagesRef.current = nextMessages;
  }, []);

  // Optimized chat fetching with stale response protection.
  const getChat = useCallback(async (start: number, end: number, generation = requestGenerationRef.current) => {
    if (pendingRangeRef.current) return false;

    pendingRangeRef.current = { start, end, generation };

    try {
      const data = await getChatForVideo(video.id, start, end);
      if (generation !== requestGenerationRef.current) return false;

      enqueueComments(data);
      lastCheckTimeRef.current = Math.max(lastCheckTimeRef.current, end);
      return true;
    } catch (error) {
      handleError(error as Error, "Chat fetching");
      return false;
    } finally {
      const pendingRange = pendingRangeRef.current;
      if (
        pendingRange?.generation === generation &&
        pendingRange.start === start &&
        pendingRange.end === end
      ) {
        pendingRangeRef.current = null;
      }
    }
  }, [enqueueComments, video.id, handleError]);

  const getSeekChat = useCallback(async (start: number, count: number, generation = requestGenerationRef.current) => {
    if (isSeekLoadingRef.current) return false;

    isSeekLoadingRef.current = true;

    try {
      const data = await getSeekChatForVideo(video.id, start, count);
      if (generation !== requestGenerationRef.current) return false;

      enqueueComments(data);
      return true;
    } catch (error) {
      handleError(error as Error, "Seek chat fetching");
      return false;
    } finally {
      if (generation === requestGenerationRef.current) {
        isSeekLoadingRef.current = false;
      }
    }
  }, [enqueueComments, video.id, handleError]);

  const clearChat = useCallback(() => {
    requestGenerationRef.current += 1;
    pendingRangeRef.current = null;
    isSeekLoadingRef.current = false;
    internalMessagesRef.current = [];
    queuedIdsRef.current.clear();
    processedIdsRef.current.clear();
    processedIdsOrderRef.current = [];
    setMessagesWithScroll([]);
    addCustomComment(t('chatTimeSkipDetected'));
    return requestGenerationRef.current;
  }, [addCustomComment, setMessagesWithScroll, t]);

  // Function to add an ID to the processed set
  const addProcessedId = useCallback((id: string) => {
    if (processedIdsRef.current.has(id)) return;

    processedIdsRef.current.add(id);
    processedIdsOrderRef.current.push(id);

    // Remove oldest IDs if size exceeds MAX_CHAT_MESSAGES * 2
    while (processedIdsOrderRef.current.length > MAX_CHAT_MESSAGES * 2) {
      const oldestId = processedIdsOrderRef.current.shift();
      if (oldestId) {
        processedIdsRef.current.delete(oldestId);
      }
    }
  }, []);

  // chatTick handles processing of chat messages
  const chatTick = useCallback(async (time: number) => {
    try {
      // Collect new messages to add in one batch
      const newMessagesToAdd: Array<Comment> = [];

      // Process messages from the internal ref
      while (internalMessagesRef.current.length > 0) {
        const comment = internalMessagesRef.current[0];

        // Stop if the message is in the future
        if (comment.content_offset_seconds > time) break;

        // Remove the message from the queue
        internalMessagesRef.current.shift();
        queuedIdsRef.current.delete(comment._id);

        // Skip duplicates
        if (processedIdsRef.current.has(comment._id)) continue;

        // Process the message (e.g. add badges and emotes)
        const processedComment = addBadgesToFormattedComment(comment);
        processedComment.ganymede_formatted_message = addEmotesToFormattedComment(processedComment);

        // Add to batch and mark as processed
        newMessagesToAdd.push(processedComment);
        addProcessedId(comment._id);
      }

      // Update state once with all new messages
      if (newMessagesToAdd.length > 0) {
        setMessagesWithScroll((prev) => {
          const updatedMessages = [...prev, ...newMessagesToAdd];
          return updatedMessages.slice(-MAX_CHAT_MESSAGES);
        });
      }

    } catch (error) {
      handleError(error as Error, "Chat processing");
    }
  }, [addBadgesToFormattedComment, addEmotesToFormattedComment, addProcessedId, handleError, setMessagesWithScroll]);

  // Initialize chat data
  useEffect(() => {
    if (!chatEmotes?.length || !chatBadges?.length) return;
    if (emotesError || badgesError) {
      const errorMessage = (emotesError?.message || badgesError?.message || "Unknown error");
      handleError(new Error(errorMessage), "Chat initialization");
      return;
    }

    try {
      // Process emotes
      chatEmotes.forEach((emote: Emote) => {
        if (!emote.name || emote.type === "twitch") {
          chatMapsRef.current.emoteMap.set(emote.id, emote);
        } else {
          chatMapsRef.current.emoteMap.set(emote.name, emote);
          chatMapsRef.current.thirdPartyEmoteMap.set(emote.name, emote);
        }
      });

      // Process badges
      chatBadges.forEach((badge: Badge) => {
        switch (badge.name) {
          case "subscriber":
            chatMapsRef.current.subscriptionBadgeMap.set(badge.version, badge);
            break;
          case "sub-gifter":
            chatMapsRef.current.subscriptionGiftBadgeMap.set(badge.version, badge);
            break;
          case "bits":
            chatMapsRef.current.bitBadgeMap.set(badge.version, badge);
            break;
          default:
            if (!IGNORED_BADGES.has(badge.name)) {
              chatMapsRef.current.generalBadgeMap.set(badge.name, badge);
            }
        }
      });

      setIsReady(true);
      addCustomComment(t('chatPlayerReady'));
      addCustomComment(
        t.markup('chatPlayerReadyStats', {
          lengthBadges: chatMapsRef.current.generalBadgeMap.size.toLocaleString(),
          lengthSubBadges: chatMapsRef.current.subscriptionBadgeMap.size.toLocaleString(),
          lengthEmotes: chatMapsRef.current.emoteMap.size.toLocaleString(),
        })
      );
    } catch (error) {
      handleError(error as Error, "Chat initialization");
    }
  }, [chatEmotes, chatBadges, addCustomComment, handleError, emotesError, badgesError]);

  // Chat update interval
  useEffect(() => {
    if (!isReady) return;

    const interval = setInterval(() => {
      const { time, isPaused } = VideoEventBus.getData();
      if (isPaused) return;

      if (Math.abs(time - lastTimeRef.current) > TIME_SKIP_THRESHOLD) {
        console.log(`Player time skip detected - ${lastTimeRef.current} -> ${time}`);
        const generation = clearChat();
        lastCheckTimeRef.current = time;
        getSeekChat(time, 50, generation);
        getChat(time, time + CHAT_OFFSET_SIZE, generation);
      }

      lastTimeRef.current = time;

      if (time <= lastCheckTimeRef.current) return;
      if (pendingRangeRef.current) return;

      const startTime = lastCheckTimeRef.current || time;
      const endTime = startTime + CHAT_OFFSET_SIZE;

      getChat(startTime, endTime);
    }, TICK_INTERVAL);

    return () => clearInterval(interval);
  }, [isReady, clearChat, getChat, getSeekChat]);

  // Chat processing interval
  useEffect(() => {
    if (!isReady) return;

    const interval = setInterval(() => {
      const { time } = VideoEventBus.getData();
      chatTick(time);
    }, TICK_INTERVAL);

    return () => clearInterval(interval);
  }, [isReady, chatTick]);

  // Add scroll event listener
  useEffect(() => {
    const container = chatContainerRef.current;
    if (!container) return;

    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [handleScroll]);

  // Initial scroll on mount
  useEffect(() => {
    scrollToBottom(false);
  }, [scrollToBottom]);

  if (!isReady) {
    return (
      <div className={classes.chatPlayerContainer}>
        <Center>
          <div style={{ marginTop: "100%" }}>
            <Center>
              <Loader size="xl" />
            </Center>
            <Text mt={5}>{t('loadingChat')}</Text>
            {error && (
              <Text size="sm">
                {t('chatError')}: {error.message}
              </Text>
            )}
          </div>
        </Center>

      </div>
    );
  }

  return (
    <div
      ref={chatContainerRef}
      className={`${classes.chatPlayerContainer} `}
    >
      {error && (
        <Box className="p-2 mb-2 bg-red-100 text-red-800 rounded">
          <Text size="sm">{error.message}</Text>
        </Box>
      )}
      {messages.map((comment) => (
        <ChatMessage
          key={comment._id}
          comment={comment}
          showTimestamp={showChatTimestamps}
          timestampSeconds={showChatTimestamps ? getCommentTimestampSeconds(comment) : null}
          onTimestampClick={() => {
            if (!Number.isFinite(comment.content_offset_seconds)) return;
            seekToComment(comment.content_offset_seconds);
          }}
        />
      ))}
    </div>
  );
};

export default ChatPlayer;
