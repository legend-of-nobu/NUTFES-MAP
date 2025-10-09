"use client";
import React, { useState } from "react";
import { useAuth } from "@/components/user-resister/AuthContext";
import { registerUser } from "@/components/user-resister/Auth";

export default function AuthPage() {
  const { login, isLoading, user } = useAuth();
  const [mode, setMode] = useState<"login" | "register">("login");

  // ログインフォーム入力値
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  // 新規登録フォーム入力値
  const [name, setName] = useState("");
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");

  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      setError(null);
      await login(email, password);
    } catch (err) {
      setError("メールアドレスまたはパスワードが違います。");
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setLoading(true);
      setError(null);
      await registerUser(name, regEmail, regPassword);
      alert("登録が完了しました。ログインしてください。");
      setMode("login");
    } catch (err) {
      setError("登録に失敗しました。");
    } finally {
      setLoading(false);
    }
  };

  if (isLoading) return <p>ロード中...</p>;
  if (user) return <p>{user.name}さんでログイン中です。</p>;

  return (
    <div className="flex flex-col items-center min-h-screen bg-gray-100 p-6">
      <div className="bg-white rounded-2xl shadow-md w-full max-w-md p-8">
        <h1 className="text-2xl font-bold mb-6 text-center">
          {mode === "login" ? "ログイン" : "新規登録"}
        </h1>

        {error && <p className="text-red-500 text-sm mb-4">{error}</p>}

        {mode === "login" ? (
          <form onSubmit={handleLogin} className="space-y-4">
            <input
              type="email"
              placeholder="メールアドレス"
              className="w-full p-2 border rounded"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <input
              type="password"
              placeholder="パスワード"
              className="w-full p-2 border rounded"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <button
              type="submit"
              className="w-full bg-blue-500 text-white py-2 rounded hover:bg-blue-600"
              disabled={loading}
            >
              {loading ? "送信中..." : "ログイン"}
            </button>
          </form>
        ) : (
          <form onSubmit={handleRegister} className="space-y-4">
            <input
              type="text"
              placeholder="名前"
              className="w-full p-2 border rounded"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <input
              type="email"
              placeholder="メールアドレス"
              className="w-full p-2 border rounded"
              value={regEmail}
              onChange={(e) => setRegEmail(e.target.value)}
            />
            <input
              type="password"
              placeholder="パスワード"
              className="w-full p-2 border rounded"
              value={regPassword}
              onChange={(e) => setRegPassword(e.target.value)}
            />
            <button
              type="submit"
              className="w-full bg-green-500 text-white py-2 rounded hover:bg-green-600"
              disabled={loading}
            >
              {loading ? "送信中..." : "登録"}
            </button>
          </form>
        )}

        <button
          className="text-sm text-blue-500 mt-4 underline"
          onClick={() =>
            setMode(mode === "login" ? "register" : "login")
          }
        >
          {mode === "login"
            ? "新規登録はこちら"
            : "ログイン画面に戻る"}
        </button>
      </div>
    </div>
  );
}
