import { useInfiniteQuery } from "@tanstack/react-query";
import { listExpenses, type ListExpensesParams } from "../api/expenses";
import type { ExpensePage } from "../types";

export const expensesQueryKey = ["expenses"] as const;

export function useExpenses(limit?: number) {
  return useInfiniteQuery<
    ExpensePage,
    Error,
    { pages: ExpensePage[]; pageParams: (string | null)[] },
    typeof expensesQueryKey,
    string | null
  >({
    queryKey: expensesQueryKey,
    initialPageParam: null,
    queryFn: ({ pageParam }) => {
      const params: ListExpensesParams = { limit };
      if (pageParam) {
        params.before = pageParam;
      }
      return listExpenses(params);
    },
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
  });
}
