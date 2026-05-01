import { request } from "./client";
import type { Insights, InsightsQuery } from "../types";

export function getInsights(query: InsightsQuery): Promise<Insights> {
  return request<Insights>("/api/insights", {
    query: {
      view: query.view,
      year: query.year,
      month: query.month,
    },
  });
}
