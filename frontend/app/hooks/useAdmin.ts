import { AxiosInstance } from "axios";
import { ApiResponse } from "./useAxios";
import { useQuery } from "@tanstack/react-query";

export interface GanymedeInformation {
  commit_hash: string;
  build_time: string;
  uptime: string;
  program_versions: GanymedeProgramVersions;
}

export interface GanymedeProgramVersions {
  ffmpeg: string;
  twitch_downloader: string;
  chat_downloader: string;
  streamlink: string;
}

const getGanymedeInformation = async (
  axiosPrivate: AxiosInstance
): Promise<GanymedeInformation> => {
  const response = await axiosPrivate.get<ApiResponse<GanymedeInformation>>(
    "/api/v1/admin/info"
  );
  return response.data.data;
};

const useGetGanymedeInformation = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["ganymede-information"],
    queryFn: () => getGanymedeInformation(axiosPrivate),
  });
};

export { useGetGanymedeInformation };
