import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiResponse } from "./useAxios";
import { AxiosInstance } from "axios";
import { User } from "./useAuthentication";
import { NullResponse } from "./usePlayback";

const getUsers = async (axiosPrivate: AxiosInstance): Promise<Array<User>> => {
  const response = await axiosPrivate.get<ApiResponse<Array<User>>>(
    "/api/v1/user"
  );
  return response.data.data;
};

const getUserById = async (
  axiosPrivate: AxiosInstance,
  id: string
): Promise<User> => {
  const response = await axiosPrivate.get<ApiResponse<User>>(
    `/api/v1/user/${id}`
  );
  return response.data.data;
};

const useGetUsers = (axiosPrivate: AxiosInstance) => {
  return useQuery({
    queryKey: ["users"],
    queryFn: () => getUsers(axiosPrivate),
  });
};

const useGetUserById = (axiosPrivate: AxiosInstance, id: string) => {
  return useQuery({
    queryKey: ["user", id],
    queryFn: () => getUserById(axiosPrivate, id),
  });
};

const editUser = async (
  axiosPrivate: AxiosInstance,
  user: User
): Promise<User> => {
  const response = await axiosPrivate.put(`/api/v1/user/${user.id}`, {
    username: user.username,
    role: user.role,
  });
  return response.data.data;
};

interface EditUserVariables {
  axiosPrivate: AxiosInstance;
  user: User;
}

const useEditUser = () => {
  const queryClient = useQueryClient();
  return useMutation<User, Error, EditUserVariables>({
    mutationFn: ({ axiosPrivate, user }) => editUser(axiosPrivate, user),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
    },
  });
};

const deleteUser = async (axiosPrivate: AxiosInstance, userId: string) => {
  const response = await axiosPrivate.delete(`/api/v1/user/${userId}`);
  return response.data;
};

interface DeleteUserVariables {
  axiosPrivate: AxiosInstance;
  userId: string;
}

const useDeleteUser = () => {
  const queryClient = useQueryClient();
  return useMutation<NullResponse, Error, DeleteUserVariables>({
    mutationFn: ({ axiosPrivate, userId }) => deleteUser(axiosPrivate, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
    },
  });
};

export { useGetUserById, useGetUsers, useEditUser, useDeleteUser };
