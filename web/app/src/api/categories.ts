import { request } from "./client";
import type { Category } from "../types";

export function listCategories(): Promise<Category[]> {
  return request<Category[]>("/api/categories");
}
