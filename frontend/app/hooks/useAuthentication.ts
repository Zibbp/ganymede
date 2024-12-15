import useAxios, { ApiResponse } from "./useAxios";
import { notifications } from "@mantine/notifications";
import { AxiosError } from "axios";
import classes from "@/app/components/notifications/Notifications.module.css";

export interface User {
  id: string;
  username: string;
  oauth: boolean;
  role: UserRole;
  updated_at: Date;
  created_at: Date;
}

export enum UserRole {
  Admin = "admin",
  Editor = "editor",
  Archiver = "archiver",
  User = "user",
}

type MeResponse = ApiResponse<User>;

// authLogin performs a HTTP request to the login route
const authLogin = async (username: string, password: string) => {
  try {
    await useAxios.post(
      "/api/v1/auth/login",
      {
        username,
        password,
      },
      {
        withCredentials: true,
      }
    );
  } catch (error) {
    if (error instanceof AxiosError) {
      console.error("error logging in", error);

      notifications.show({
        color: "red",
        title: error.response?.statusText || "Error",
        message:
          error.response?.data?.message || "An unexpected error occurred",
        classNames: classes,
      });

      throw new Error("Error logging in");
    }
  }
};

// authRegister performs a HTTP request to the register route
const authRegister = async (username: string, password: string) => {
  try {
    await useAxios.post(
      "/api/v1/auth/register",
      {
        username,
        password,
      },
      {
        withCredentials: true,
      }
    );
  } catch (error) {
    if (error instanceof AxiosError) {
      console.error("error registering in", error);

      notifications.show({
        color: "red",
        title: error.response?.statusText || "Error",
        message:
          error.response?.data?.message || "An unexpected error occurred",
        classNames: classes,
      });

      throw new Error("Error registering in");
    }
  }
};

// authLogout performs a HTTP request to the logout route
const authLogout = async () => {
  try {
    await useAxios.post(
      "/api/v1/auth/logout",
      {},
      {
        withCredentials: true,
      }
    );
  } catch (error) {
    if (error instanceof AxiosError) {
      console.error("error logging out", error);

      notifications.show({
        color: "red",
        title: error.response?.statusText || "Error",
        message:
          error.response?.data?.message || "An unexpected error occurred",
        classNames: classes,
      });

      throw new Error("Error logging out");
    }
  }
};

// authChangePassword performs a HTTP request to the change password route
const authChangePassword = async (
  oldPassword: string,
  newPassword: string,
  confirmNewPassword: string
) => {
  try {
    await useAxios.post(
      "/api/v1/auth/change-password",
      {
        old_password: oldPassword,
        new_password: newPassword,
        confirm_new_password: confirmNewPassword,
      },
      {
        withCredentials: true,
      }
    );
  } catch (error) {
    if (error instanceof AxiosError) {
      console.error("error changin password", error);

      notifications.show({
        color: "red",
        title: error.response?.statusText || "Error",
        message:
          error.response?.data?.message || "An unexpected error occurred",
        classNames: classes,
      });
    }
  }
};

// getUserInfo performs a HTTP request getting the logged in user's information
const getUserInfo = async (): Promise<MeResponse> => {
  try {
    const response = await useAxios.get("/api/v1/auth/me", {
      withCredentials: true,
    });
    return response.data;
  } catch (error) {
    // Handle all types of errors
    if (error instanceof AxiosError) {
      throw new Error(`Error getting user information: ${error.message}`, {
        cause: error,
      });
    }
    // Handle other types of errors
    throw new Error(
      `Unknown error occurred: ${
        error instanceof Error ? error.message : String(error)
      }`
    );
  }
};

export { authLogin, getUserInfo, authLogout, authRegister, authChangePassword };
