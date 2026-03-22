"use client";

import Link from "next/link";
import { useEffect, useState, useTransition } from "react";
import { useRouter } from "next/navigation";
import { SellerAuthShell } from "@/components/seller-auth-shell";
import { authStorage } from "@/lib/api/client";
import { sellerAPI } from "@/lib/api/seller";

export default function SellerLoginPage() {
  const router = useRouter();
  const [isNavigating, startTransition] = useTransition();
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("123456");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [hasToken, setHasToken] = useState(false);

  useEffect(() => {
    setHasToken(Boolean(authStorage.getToken()));
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const params = new URLSearchParams(window.location.search);
    if (params.get("registered") === "1") {
      setSuccess("注册成功，seller_token 已保存。Week 6 会接上控制台页，当前可以继续联调后端接口。");
    }
  }, []);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    setSuccess("");

    try {
      const data = await sellerAPI.login(phone.trim(), code.trim());
      authStorage.setToken(data.token);
      setHasToken(true);
      setSuccess("登录成功，正在进入卖家控制台。");
      startTransition(() => {
        router.push("/seller/dashboard");
      });
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : "登录失败";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  function clearToken() {
    authStorage.clearToken();
    setHasToken(false);
    setSuccess("本地 seller_token 已清除。");
  }

  return (
    <SellerAuthShell
      kicker="Seller Login"
      title="卖家登录"
      subtitle="用手机号和 MVP 验证码进入卖家入口。当前阶段先把认证体验跑通，控制台视图会在 Week 6 接上。"
    >
      <form className="auth-form" onSubmit={handleSubmit}>
        <label className="auth-field">
          <span className="auth-label">手机号</span>
          <input
            className="auth-input"
            type="tel"
            value={phone}
            onChange={(event) => setPhone(event.target.value)}
            placeholder="13800138000"
            autoComplete="tel"
            required
          />
        </label>

        <label className="auth-field">
          <span className="auth-label">验证码</span>
          <input
            className="auth-input"
            type="text"
            value={code}
            onChange={(event) => setCode(event.target.value)}
            placeholder="123456"
            inputMode="numeric"
            required
          />
          <span className="auth-hint">MVP 阶段固定验证码：123456</span>
        </label>

        {error ? <div className="auth-message auth-message--error">{error}</div> : null}
        {success ? <div className="auth-message auth-message--success">{success}</div> : null}
        {hasToken ? (
          <div className="auth-message auth-message--success">
            浏览器里已经存在 seller token。你可以直接继续联调，或者先把它清掉再重新登录。
          </div>
        ) : null}

        <div className="auth-actions">
          <button className="auth-button" type="submit" disabled={loading || isNavigating}>
            {loading || isNavigating ? "登录中..." : "登录"}
          </button>
          {hasToken ? (
            <button className="auth-button auth-button--ghost" type="button" onClick={clearToken}>
              清除本地 token
            </button>
          ) : null}
        </div>
      </form>

      <p className="auth-footer">
        还没有卖家账号？<Link href="/seller/register">去注册</Link>
      </p>
    </SellerAuthShell>
  );
}
