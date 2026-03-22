"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
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

export default function BuyerLoginPage() {
  const [identity, setIdentity] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [hasToken, setHasToken] = useState(false);

  useEffect(() => {
    setHasToken(Boolean(buyerStorage.getToken()));
  }, []);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    setSuccess("");

    try {
      const { email, phone } = parseIdentity(identity);
      const data = await buyerAPI.login({
        email: email || undefined,
        phone: phone || undefined,
        password: password.trim(),
      });
      buyerStorage.setToken(data.token);
      setHasToken(true);
      setSuccess("登录成功，buyer_token 已写入浏览器。买家控制台将在 Week 7 接入。");
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : "登录失败";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  function clearAuth() {
    buyerStorage.clearToken();
    buyerStorage.clearAPIKey();
    setHasToken(false);
    setSuccess("本地 buyer_token 和 buyer_api_key 已清除。");
  }

  return (
    <BuyerAuthShell
      kicker="Buyer Login"
      title="买家登录"
      subtitle="按真实后端实现，当前登录支持邮箱或手机号配合密码，而不是文档示例里的验证码模式。"
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
        </label>

        <label className="auth-field">
          <span className="auth-label">密码</span>
          <input
            className="auth-input"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="请输入密码"
            required
          />
        </label>

        {error ? <div className="auth-message auth-message--error">{error}</div> : null}
        {success ? <div className="auth-message auth-message--success">{success}</div> : null}
        {hasToken ? (
          <div className="auth-message auth-message--success">
            浏览器里已经存在 buyer token。Week 7 控制台接上后，这个登录态会直接复用。
          </div>
        ) : null}

        <div className="auth-actions">
          <button className="auth-button" type="submit" disabled={loading}>
            {loading ? "登录中..." : "登录"}
          </button>
          {hasToken ? (
            <button className="auth-button auth-button--ghost" type="button" onClick={clearAuth}>
              清除本地凭证
            </button>
          ) : null}
        </div>
      </form>

      <p className="auth-footer">
        还没有买家账号？<Link href="/buyer/register">去注册</Link>
      </p>
    </BuyerAuthShell>
  );
}
