import axios from "axios";
import { env } from "next-runtime-env";
import { useRouter } from "next/navigation";
import useAuthStore from "../store/useAuthStore";
import { showNotification } from "@mantine/notifications";

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  message: string;
}

const useAxios = axios.create({
  baseURL: env("NEXT_PUBLIC_API_URL"),
  headers: {
    "Content-Type": "application/json",
  },
});

export const useAxiosPrivate = () => {
  const router = useRouter();
  const { logout } = useAuthStore();

  const axiosPrivate = axios.create({
    baseURL: env("NEXT_PUBLIC_API_URL"),
    headers: {
      "Content-Type": "application/json",
    },
    withCredentials: true, // Send cookies with requests
  });

  axiosPrivate.interceptors.response.use(
    function (response) {
      return response;
    },
    async (err) => {
      const prevRequest = err?.config;
      console.log(err);
      if (err.response.status === 401 && !prevRequest?.sent) {
        console.debug("401 detected in private request");
        // clear auth store
        logout();

        router.push("/login");
        prevRequest.sent = true;
        // try {
        //   if (useUserStore.getState().oauth == true) {
        //     await refreshOAuthAccessToken();
        //   } else {
        //     await refreshAccessToken();
        //   }

        //   console.debug("New access token received - retrying request");
        //   prevRequest.headers["Content-Type"] = "application/json";
        //   delete prevRequest.headers;
        //   return axiosInstance.request(prevRequest);
        // } catch (err) {
        //   console.error(
        //     "Failed to refresh access token - redirecting to login"
        //   );
        // }
      }
      showNotification({
        color: "red",
        title: "Request error",
        message: err.response.data.message,
      });
      return Promise.reject(err);
    }
  );

  return axiosPrivate;
};

export default useAxios;
