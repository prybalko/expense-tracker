import { request } from "./client";
import type { User } from "../types";

export function login(username: string, password: string): Promise<User> {
  return request<User>("/api/auth/login", {
    method: "POST",
    body: { username, password },
  });
}

export function logout(): Promise<{ ok: boolean }> {
  return request<{ ok: boolean }>("/api/auth/logout", { method: "POST" });
}

export function getMe(): Promise<User> {
  return request<User>("/api/auth/me");
}
