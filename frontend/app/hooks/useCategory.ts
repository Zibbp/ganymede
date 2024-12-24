import { useQuery } from "@tanstack/react-query";
import useAxios from "./useAxios";

export interface Category {
  id: string;
  name: string;
}

const getTwitchCategories = async (): Promise<Array<Category>> => {
  const response = await useAxios.get(`/api/v1/category`);
  return response.data.data;
};

const useGetTwitchCategories = () => {
  return useQuery({
    queryKey: ["twitch_categories"],
    queryFn: () => getTwitchCategories(),
  });
};

export { useGetTwitchCategories, getTwitchCategories };
