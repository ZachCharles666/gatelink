"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { SellerAuthShell } from "@/components/seller-auth-shell";
import { authStorage } from "@/lib/api/client";
import { sellerAPI } from "@/lib/api/seller";

export default function SellerRegisterPage() {
  const router = useRouter();
  const [isNavigating, startTransition] = useTransition();
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("123456");
  const [displayName, setDisplayName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    setSuccess("");

    try {
      const data = await sellerAPI.register(phone.trim(), code.trim(), displayName.trim());
      authStorage.setToken(data.token);
      setSuccess("注册成功，正在进入卖家控制台。");
      startTransition(() => {
        router.push("/seller/dashboard?welcome=1");
      });
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : "注册失败";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <SellerAuthShell
      kicker="Seller Register"
      title="卖家注册"
      subtitle="先创建卖家身份，把 token 落进浏览器。本周只启动入口页，账户管理和收益面板会在下一阶段继续补。"
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

        <label className="auth-field">
          <span className="auth-label">显示名称</span>
          <input
            className="auth-input"
            type="text"
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
            placeholder="例如：北美 API 卖家"
          />
        </label>

        {error ? <div className="auth-message auth-message--error">{error}</div> : null}
        {success ? <div className="auth-message auth-message--success">{success}</div> : null}

        <div className="auth-actions">
          <button className="auth-button" type="submit" disabled={loading || isNavigating}>
            {loading || isNavigating ? "注册中..." : "注册"}
          </button>
        </div>
      </form>

      <p className="auth-footer">
        已经有卖家账号？<Link href="/seller/login">回到登录</Link>
      </p>
    </SellerAuthShell>
  );
}
