import type { StatsResponse, UserListResponse } from "@/types/users";
import { USER_PAGE_SIZE } from "@/types/users";

function getApiBaseUrl() {
  return process.env.NEXT_PUBLIC_API_BASE_URL?.replace(/\/$/, "");
}

export async function fetchUsers(cursor?: string): Promise<UserListResponse> {
  const baseUrl = getApiBaseUrl();
  if (!baseUrl) throw new Error("Set NEXT_PUBLIC_API_BASE_URL in frontend/.env.local.");

  const params = new URLSearchParams({ limit: String(USER_PAGE_SIZE) });
  if (cursor) params.set("cursor", cursor);

  const response = await fetch(`${baseUrl}/api/users?${params}`, { cache: "no-store" });
  if (!response.ok) throw new Error(`Could not load users (API ${response.status}).`);
  return (await response.json()) as UserListResponse;
}

export async function fetchUserStats(userIds: string[]): Promise<StatsResponse> {
  const baseUrl = getApiBaseUrl();
  if (!baseUrl) throw new Error("Set NEXT_PUBLIC_API_BASE_URL in frontend/.env.local.");

  const response = await fetch(`${baseUrl}/api/users/stats`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ userIds }),
  });

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as { error?: string } | null;
    throw new Error(payload?.error ?? `The API returned an error (${response.status}).`);
  }

  return (await response.json()) as StatsResponse;
}
