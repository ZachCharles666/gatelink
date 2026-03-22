"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState, useTransition } from "react";
import { SellerConsoleShell } from "@/components/seller-console-shell";
import { authStorage } from "@/lib/api/client";
import { sellerAPI, type AddAccountResponse } from "@/lib/api/seller";

const VENDORS = ["anthropic", "openai", "gemini", "qwen", "glm", "kimi"];

function defaultExpireAt() {
  const date = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000);
  const year = date.getFullYear();
  const month = `${date.getMonth() + 1}`.padStart(2, "0");
  const day = `${date.getDate()}`.padStart(2, "0");
  const hours = `${date.getHours()}`.padStart(2, "0");
  const minutes = `${date.getMinutes()}`.padStart(2, "0");
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

export default function AddAccountPage() {
  const router = useRouter();
  const [isNavigating, startTransition] = useTransition();
  const [form, setForm] = useState({
    vendor: "anthropic",
    api_key: "",
    authorized_credits_usd: "100",
    expected_rate: "0.75",
    expire_at: defaultExpireAt(),
    total_credits_usd: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<AddAccountResponse | null>(null);

  useEffect(() => {
    if (!authStorage.getToken()) {
      router.replace("/seller/login");
    }
  }, [router]);

  useEffect(() => {
    if (!result) {
      return;
    }

    const timer = window.setTimeout(() => {
      startTransition(() => {
        router.push("/seller/dashboard?added=1");
      });
    }, 1200);

    return () => window.clearTimeout(timer);
  }, [result, router, startTransition]);

  function updateField(field: keyof typeof form, value: string) {
    setForm((current) => ({
      ...current,
      [field]: value,
    }));
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    setResult(null);

    try {
      const response = await sellerAPI.addAccount({
        vendor: form.vendor,
        api_key: form.api_key.trim(),
        authorized_credits_usd: Number(form.authorized_credits_usd),
        expected_rate: Number(form.expected_rate),
        expire_at: new Date(form.expire_at).toISOString(),
        total_credits_usd: form.total_credits_usd ? Number(form.total_credits_usd) : undefined,
      });
      setResult(response);
      setForm((current) => ({
        ...current,
        api_key: "",
      }));
    } catch (submitError) {
      const message = submitError instanceof Error ? submitError.message : "添加账号失败";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <SellerConsoleShell
      title="添加托管账号"
      subtitle="这里直接对接 `POST /api/v1/seller/accounts`。MVP 阶段会先做格式检查和卖家侧记录，真实 live verify 仍受共享 accounts/engine 池状态影响。"
      actions={
        <Link className="console-secondary" href="/seller/dashboard">
          返回控制台
        </Link>
      }
    >
      <section className="console-stack">
        <div className="console-alert console-alert--warn">
          当前账号接入链路会先经过格式检查。若出现 “engine verify requires shared account persistence before live verification”，说明是共享
          `accounts` / engine 联调状态未就绪，不是表单字段格式错误。
        </div>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">托管账号表单</h2>
              <p className="console-panel-copy">
                推荐先用一条测试额度较小的账号做联调。回收率建议从 0.75 开始，后面再按真实成交情况微调。
              </p>
            </div>
          </div>

          <form className="console-form" onSubmit={handleSubmit}>
            <div className="console-form-grid">
              <label className="console-form-field">
                <span className="console-form-label">厂商</span>
                <select
                  className="console-form-select"
                  value={form.vendor}
                  onChange={(event) => updateField("vendor", event.target.value)}
                >
                  {VENDORS.map((vendor) => (
                    <option key={vendor} value={vendor}>
                      {vendor}
                    </option>
                  ))}
                </select>
              </label>

              <label className="console-form-field">
                <span className="console-form-label">期望回收率</span>
                <input
                  className="console-form-input"
                  type="number"
                  min="0.5"
                  max="0.95"
                  step="0.01"
                  value={form.expected_rate}
                  onChange={(event) => updateField("expected_rate", event.target.value)}
                  required
                />
                <span className="console-form-hint">卖家收益 = 实际成本 × 回收率。推荐值 0.75。</span>
              </label>

              <label className="console-form-field console-form-field--full">
                <span className="console-form-label">API Key</span>
                <input
                  className="console-form-input"
                  type="password"
                  placeholder="sk-ant-..."
                  value={form.api_key}
                  onChange={(event) => updateField("api_key", event.target.value)}
                  required
                />
              </label>

              <label className="console-form-field">
                <span className="console-form-label">授权额度（USD）</span>
                <input
                  className="console-form-input"
                  type="number"
                  min="0.01"
                  step="0.01"
                  value={form.authorized_credits_usd}
                  onChange={(event) => updateField("authorized_credits_usd", event.target.value)}
                  required
                />
              </label>

              <label className="console-form-field">
                <span className="console-form-label">账号总额度（可选）</span>
                <input
                  className="console-form-input"
                  type="number"
                  min="0"
                  step="0.01"
                  value={form.total_credits_usd}
                  onChange={(event) => updateField("total_credits_usd", event.target.value)}
                />
              </label>

              <label className="console-form-field console-form-field--full">
                <span className="console-form-label">到期时间</span>
                <input
                  className="console-form-input"
                  type="datetime-local"
                  value={form.expire_at}
                  onChange={(event) => updateField("expire_at", event.target.value)}
                  required
                />
                <span className="console-form-hint">提交时会转成 ISO 8601，后端按 RFC3339 校验。</span>
              </label>
            </div>

            {error ? <div className="console-alert console-alert--error">{error}</div> : null}
            {result ? (
              <div className="console-alert console-alert--success">
                {result.message} 当前账号 ID：<strong>{result.account_id}</strong>，状态：{result.status}，健康度：
                {result.health_score}。正在返回控制台。
              </div>
            ) : null}

            <div className="console-form-actions">
              <button className="console-primary" type="submit" disabled={loading || isNavigating}>
                {loading || isNavigating ? "提交中..." : "添加账号"}
              </button>
              <Link className="console-secondary" href="/seller/dashboard">
                稍后再说
              </Link>
            </div>
          </form>
        </section>
      </section>
    </SellerConsoleShell>
  );
}
