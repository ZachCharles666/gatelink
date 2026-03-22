"use client";

import Link from "next/link";
import { useState } from "react";
import { BuyerAuthShell } from "@/components/buyer-auth-shell";
import { buyerAPI, buyerStorage } from "@/lib/api/buyer";

function parseIdentity(identity: string) {
  const trimmed = identity.trim();
  if (!trimmed) {
    return { email: "", phone: "" };
  }
  if (trimmed.includes("@")) {
    return { email: trimmed, phone: "" };
  }
  return { email: "", phone: trimmed };
}

export default function BuyerRegisterPage() {
  const [identity, setIdentity] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [apiKey, setAPIKey] = useState("");

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      const { email, phone } = parseIdentity(identity);
      const data = await buyerAPI.register({
        email: email || undefined,
        phone: phone || undefined,
        password: password.trim(),
      });
      buyerStorage.setToken(data.token);
      if (data.api_key) {
        buyerStorage.setAPIKey(data.api_key);
        setAPIKey(data.api_key);
      }
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : "注册失败";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  if (apiKey) {
    return (
      <BuyerAuthShell
        kicker="Buyer Key Ready"
        title="注册成功"
        subtitle="这把 buyer api_key 只在这里完整展示一次。后续会保存在浏览器里，但建议你现在就手动保存。"
      >
        <div className="auth-form">
          <div className="auth-message auth-message--success">
            buyer_token 和 buyer_api_key 都已经写入浏览器。本周先把认证入口接通，买家控制台会在 Week 7 继续补完。
          </div>
          <div className="auth-field">
            <span className="auth-label">请立即保存这把 API Key</span>
            <code className="auth-input" style={{ fontFamily: "\"SFMono-Regular\", Menlo, monospace", wordBreak: "break-all" }}>
              {apiKey}
            </code>
            <span className="auth-hint">它将用于调用 `/v1/chat/completions` 等代理接口，页面不会再次完整展示。</span>
          </div>
          <p className="auth-footer">
            已经保存好了？<Link href="/buyer/login">回到登录</Link>
          </p>
        </div>
      </BuyerAuthShell>
    );
  }

  return (
    <BuyerAuthShell
      kicker="Buyer Register"
      title="买家注册"
      subtitle="按真实后端实现，注册至少需要邮箱或手机号中的一个，再配上密码。成功后会返回 buyer api_key 并只在这里完整展示一次。"
    >
      <form className="auth-form" onSubmit={handleSubmit}>
        <label className="auth-field">
          <span className="auth-label">邮箱或手机号</span>
          <input
            className="auth-input"
            type="text"
            value={identity}
            onChange={(event) => setIdentity(event.target.value)}
            placeholder="buyer@example.com 或 13800138000"
            required
          />
          <span className="auth-hint">后端要求邮箱或手机号至少填写一个，这里用单个输入框统一承接。</span>
        </label>

        <label className="auth-field">
          <span className="auth-label">密码</span>
          <input
            className="auth-input"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="请设置密码"
            required
          />
        </label>

        {error ? <div className="auth-message auth-message--error">{error}</div> : null}

        <div className="auth-actions">
          <button className="auth-button" type="submit" disabled={loading}>
            {loading ? "注册中..." : "注册"}
          </button>
        </div>
      </form>

      <p className="auth-footer">
        已经有买家账号？<Link href="/buyer/login">去登录</Link>
      </p>
    </BuyerAuthShell>
  );
}
