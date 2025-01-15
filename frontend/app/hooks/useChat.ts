import { useQuery } from "@tanstack/react-query";
import useAxios from "./useAxios";

export interface Emote {
  id: string;
  name: string;
  url: string;
  type: string;
  source: string;
  width: number;
  height: number;
}

export interface Badge {
  version: string;
  name: string;
  image_url: string;
  image_url_1x: string;
  image_url_2x: string;
  image_url_4x: string;
  description: string;
  title: string;
  click_action: string;
  click_url: string;
}

export interface Comment {
  _id: string;
  created_at: string;
  updated_at: string;
  channel_id: string;
  content_type: string;
  content_id: string;
  content_offset_seconds: number;
  commenter: Commenter;
  source: string;
  state: string;
  message: Message;
  more_replies: boolean;
  // format complex object into simpler ones for rendering
  ganymede_formatted_badges: GanymedeFormattedBadge[];
  ganymede_formatted_message: GanymedeFormattedMessageFragment[];
}

export interface GanymedeFormattedBadge {
  _id: string;
  version: string;
  title: string;
  url: string;
}

export interface GanymedeFormattedMessageFragment {
  type: GanymedeFormattedMessageType;
  text?: string;
  url?: string;
  emote?: GanymedeFormattedEmote;
}

export enum GanymedeFormattedMessageType {
  Text = "text",
  Emote = "emote",
}

export interface GanymedeFormattedEmote {
  id: string;
  name: string;
  url: string;
  type: string;
  width: number;
  height: number;
}

export interface Commenter {
  display_name: string;
  _id: string;
  name: string;
  type: string;
  bio: string;
  created_at: string;
  updated_at: string;
  logo: string;
}

export interface Message {
  body: string;
  bits_spent: string;
  fragments: Fragment[];
  is_action: boolean;
  user_badges: UserBadge[];
  user_color: string;
  user_notice_params: UserNoticeParams;
  Emoticons: EmoticonElement[];
}

export interface Fragment {
  text: string;
  emoticon: FragmentEmoticon;
}

export interface FragmentEmoticon {
  emoticon_id: string;
  emoticon_set_id: string;
}

export interface UserBadge {
  _id: string;
  version: string;
}

export interface UserNoticeParams {
  msg_id: string;
}

export interface EmoticonElement {
  _id: string;
  begin: number;
  end: number;
}

const getEmotesForVideo = async (videoId: string): Promise<Array<Emote>> => {
  const response = await useAxios.get(`/api/v1/vod/${videoId}/chat/emotes`);
  return response.data.data;
};

const useGetEmotesForVideo = (videoId: string) => {
  return useQuery<Array<Emote>>({
    queryKey: ["video", "emotes", videoId],
    queryFn: () => getEmotesForVideo(videoId),
    refetchInterval: false,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchIntervalInBackground: false,
  });
};

const getChatForVideo = async (
  videoId: string,
  start: number,
  end: number
): Promise<Array<Comment>> => {
  const response = await useAxios.get(`/api/v1/vod/${videoId}/chat`, {
    params: {
      start,
      end,
    },
  });
  return response.data.data;
};

const useGetChatForVideo = (videoId: string, start: number, end: number) => {
  return useQuery<Array<Comment>>({
    queryKey: ["video", "chat", videoId, start, end],
    queryFn: () => getChatForVideo(videoId, start, end),
    refetchInterval: false,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchIntervalInBackground: false,
  });
};

const getSeekChatForVideo = async (
  videoId: string,
  start: number,
  count: number
): Promise<Array<Comment>> => {
  const response = await useAxios.get(`/api/v1/vod/${videoId}/chat/seek`, {
    params: {
      start,
      count,
    },
  });
  return response.data.data;
};

const useGetSeekChatForVideo = (
  videoId: string,
  start: number,
  count: number
) => {
  return useQuery<Array<Comment>>({
    queryKey: ["video", "chat", "seek", videoId, start, count],
    queryFn: () => getSeekChatForVideo(videoId, start, count),
    refetchInterval: false,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchIntervalInBackground: false,
  });
};

const getBadgesForVideo = async (videoId: string): Promise<Array<Badge>> => {
  const response = await useAxios.get(`/api/v1/vod/${videoId}/chat/badges`);
  return response.data.data;
};

const useGetBadgesForVideo = (videoId: string) => {
  return useQuery<Array<Badge>>({
    queryKey: ["video", "badges", videoId],
    queryFn: () => getBadgesForVideo(videoId),
    refetchInterval: false,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchIntervalInBackground: false,
  });
};

export {
  useGetEmotesForVideo,
  useGetBadgesForVideo,
  useGetChatForVideo,
  useGetSeekChatForVideo,
  getChatForVideo,
  getSeekChatForVideo,
};
