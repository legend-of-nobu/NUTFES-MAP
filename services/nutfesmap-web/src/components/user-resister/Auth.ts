// src/api/auth.ts
import { User } from "@/types/User";

const BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL || "https://example.com";

/**
 * 共通fetch関数
 * - baseURL付き
 * - Cookie送受信対応
 * - JSON自動処理
 */
async function fetchAPI<T = unknown>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    credentials: "include", // Cookie送受信（refresh token用）
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
    ...options,
  });

  if (!res.ok) {
    // APIのエラーを例外として投げる
    const errorText = await res.text();
    throw new Error(
      `API Error (${res.status}): ${errorText || res.statusText}`
    );
  }

  // 204 No Content 対応
  if (res.status === 204) return null as unknown as T;

  return (await res.json()) as T;
}

//
// ───────────── Auth 系 ─────────────
//

// ログイン
export async function login(email: string, password: string) {
  return await fetchAPI("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

// ログアウト
export async function logout() {
  return await fetchAPI("/auth/logout", { method: "POST" });
}

// アクセストークン再発行
export async function refreshToken() {
  return await fetchAPI("/auth/refresh", { method: "POST" });
}

// ログイン中のユーザー取得
export async function fetchMe(): Promise<User> {
  return await fetchAPI("/users/me", { method: "GET" });
}

//
// ───────────── Users 系 ─────────────
//

// 新規登録
export async function registerUser(username: string, email: string, password: string) {
  const res = await fetch("http://localhost:8080/users/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, email, password }),
  });

  if (!res.ok) {
    const error = await res.text();
    throw new Error(`登録失敗: ${error}`);
  }

  return await res.json();
}

