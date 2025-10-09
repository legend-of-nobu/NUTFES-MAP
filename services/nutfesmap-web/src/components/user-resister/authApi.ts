// src/api/authApi.ts
import { fetchWithCsrf } from "./fetchWithcsrf";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "https://example.com";

export const authApi = {
  // --- ログイン ---
  login: async (email: string, password: string) => {
    const res = await fetchWithCsrf(`${API_BASE}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include", // Cookie送受信用
      body: JSON.stringify({ email, password }),
    });

    if (!res.ok) {
      const msg = await res.text();
      throw new Error(msg || "ログインに失敗しました");
    }

    return await res.json(); // { accessToken: "...", ... }
  },

  // --- 新規登録 ---
  register: async (username: string, email: string, password: string) => {
    const res = await fetchWithCsrf(`${API_BASE}/users/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, email, password }),
    });

    if (!res.ok) {
      const msg = await res.text();
      throw new Error(msg || "登録に失敗しました");
    }

    try {return await res.json(); // { id, name, email, ... }
  } catch {
    return null;
  }
  },

  // --- ログアウト ---
  logout: async () => {
    const res = await fetchWithCsrf(`${API_BASE}/auth/logout`, {
      method: "POST",
      credentials: "include",
    });

    if (!res.ok) {
      const msg = await res.text();
      throw new Error(msg || "ログアウトに失敗しました");
    }
  },

  // --- 自分の情報取得 ---
  fetchMe: async () => {
    const res = await fetch(`${API_BASE}/users/me`, {
      credentials: "include",
    });

    if (!res.ok) {
      const msg = await res.text();
      throw new Error(msg || "ユーザー情報の取得に失敗しました");
    }

    return await res.json(); // { id, name, email, ... }
  }
  
};
