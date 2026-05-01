import { useQuery } from "@tanstack/react-query";
import { listCategories } from "../api/categories";
import type { Category } from "../types";

export const categoriesQueryKey = ["categories"] as const;

export function useCategories() {
  return useQuery<Category[]>({
    queryKey: categoriesQueryKey,
    queryFn: () => listCategories(),
    staleTime: 24 * 60 * 60 * 1000,
  });
}
