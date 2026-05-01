import { useQuery } from "@tanstack/react-query";
import { getInsights } from "../api/insights";
import type { Insights, InsightsQuery } from "../types";

export function insightsQueryKey(query: InsightsQuery) {
  return ["insights", query.view, query.year ?? null, query.month ?? null] as const;
}

export function useInsights(query: InsightsQuery) {
  return useQuery<Insights>({
    queryKey: insightsQueryKey(query),
    queryFn: () => getInsights(query),
  });
}
