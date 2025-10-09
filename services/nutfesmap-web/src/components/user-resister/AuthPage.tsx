"use client";
import React, { useState } from "react";
import { authApi } from "./authApi";

export default function AuthPage() {
  // --- ログイン用 state ---
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");

  // --- 新規登録用 state ---
  const [signUpName, setSignUpName] = useState("");
  const [signUpEmail, setSignUpEmail] = useState("");
  const [signUpPassword, setSignUpPassword] = useState("");
  const [signUpConfirm, setSignUpConfirm] = useState("");

  // 共通 state
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  // --- ログイン処理 ---
  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);
    setLoading(true);
    try {
      const res = await authApi.login(loginEmail, loginPassword);
      console.log("ログイン成功:", res);
      setSuccess("ログインに成功しました！");
      // navigate("/dashboard") 等も可
  } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError(String(err));
      }
  } finally {
    setLoading(false);
  }

  // --- 新規登録処理 ---
  const handleSignUp = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSuccess(null);

    if (signUpPassword !== signUpConfirm) {
      setError("パスワードが一致しません");
      return;
    }

    setLoading(true);
    try {
      const res = await authApi.register(signUpName, signUpEmail, signUpPassword);
      console.log("登録成功:", res);
      setSuccess("登録が完了しました！ログインしてください。");
      // 登録完了後、上のログイン欄に自動で反映してもOK
      setLoginEmail(signUpEmail);
      setLoginPassword(signUpPassword);
  } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError(String(err));
      }
  } finally {
    setLoading(false);
  }
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-100">
      <div className="w-full max-w-lg bg-white rounded-2xl shadow-lg p-8 space-y-8">
        <h1 className="text-3xl font-bold text-center mb-2 text-gray-800">NUTFES MAP Auth</h1>

        {error && <p className="text-red-500 text-center">{error}</p>}
        {success && <p className="text-green-600 text-center">{success}</p>}

        {/* ログインフォーム */}
        <section>
          <h2 className="text-xl font-semibold text-center mb-4 text-blue-600">ログイン</h2>
          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">メールアドレス</label>
              <input
                type="email"
                value={loginEmail}
                onChange={(e) => setLoginEmail(e.target.value)}
                className="w-full mt-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">パスワード</label>
              <input
                type="password"
                value={loginPassword}
                onChange={(e) => setLoginPassword(e.target.value)}
                className="w-full mt-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="w-full bg-blue-500 text-white py-2 rounded-lg hover:bg-blue-600 disabled:opacity-50"
            >
              {loading ? "ログイン中..." : "ログイン"}
            </button>
          </form>
        </section>

        <hr className="border-gray-300" />

        {/* 新規登録フォーム */}
        <section>
          <h2 className="text-xl font-semibold text-center mb-4 text-green-600">新規登録</h2>
          <form onSubmit={handleSignUp} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">ユーザー名</label>
              <input
                type="text"
                value={signUpName}
                onChange={(e) => setSignUpName(e.target.value)}
                className="w-full mt-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">メールアドレス</label>
              <input
                type="email"
                value={signUpEmail}
                onChange={(e) => setSignUpEmail(e.target.value)}
                className="w-full mt-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">パスワード</label>
              <input
                type="password"
                value={signUpPassword}
                onChange={(e) => setSignUpPassword(e.target.value)}
                className="w-full mt-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">パスワード（確認）</label>
              <input
                type="password"
                value={signUpConfirm}
                onChange={(e) => setSignUpConfirm(e.target.value)}
                className="w-full mt-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="w-full bg-green-500 text-white py-2 rounded-lg hover:bg-green-600 disabled:opacity-50"
            >
              {loading ? "登録中..." : "登録する"}
            </button>
          </form>
        </section>
      </div>
    </div>
  );
}
      setLoading(false);
}};