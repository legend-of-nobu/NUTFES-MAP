// src/contexts/AuthContext.tsx
"use client";
import React, { createContext, useContext, useState, useEffect } from "react";
import { login as loginApi, logout as logoutApi, fetchMe } from "@/components/user-resister/Auth";
import { User } from "@/types/User";

type AuthContextType = {
  user: User | null;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: React.ReactNode }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // 起動時にログイン状態を確認
  useEffect(() => {
    const init = async () => {
      try {
        const me = await fetchMe(); // /users/me
        setUser(me);
      } catch {
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };
    init();
  }, []);

  // ログイン処理
  const login = async (email: string, password: string) => {
    await loginApi(email, password); // /auth/login
    const me = await fetchMe();
    setUser(me);
  };

  // ログアウト処理
  const logout = async () => {
    await logoutApi(); // /auth/logout
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

// フックで使えるようにする
export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
};
