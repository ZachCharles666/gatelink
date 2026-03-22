"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { BuyerConsoleShell } from "@/components/buyer-console-shell";
import { buyerAPI, buyerStorage } from "@/lib/api/buyer";

function maskAPIKey(value: string) {
  if (value.length <= 16) {
    return value;
  }
  return `${value.slice(0, 8)}...${value.slice(-4)}`;
}

export default function BuyerAPIKeyPage() {
  const router = useRouter();
  const [apiKey, setAPIKey] = useState("");
  const [showKey, setShowKey] = useState(false);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!buyerStorage.getToken()) {
      router.replace("/buyer/login");
      return;
    }
    setAPIKey(buyerStorage.getAPIKey() || "");
  }, [router]);

  async function handleReset() {
    const confirmed = window.confirm("重置后旧 Key 会立即失效，确认继续吗？");
    if (!confirmed) {
      return;
    }

    setLoading(true);
    setMessage("");
    setError("");

    try {
      const data = await buyerAPI.resetAPIKey();
      buyerStorage.setAPIKey(data.api_key);
      setAPIKey(data.api_key);
      setShowKey(true);
      setMessage(data.message);
    } catch (resetError) {
      const text = resetError instanceof Error ? resetError.message : "API Key 重置失败";
      setError(text);
    } finally {
      setLoading(false);
    }
  }

  async function handleCopy() {
    if (!apiKey) {
      return;
    }

    try {
      await navigator.clipboard.writeText(apiKey);
      setMessage("API Key 已复制到剪贴板。");
      setError("");
    } catch {
      setError("复制失败，请手动复制。");
    }
  }

  return (
    <BuyerConsoleShell
      title="API Key 管理"
      subtitle="这里读取浏览器中保存的 `buyer_api_key`，并通过 `POST /api/v1/buyer/apikeys/reset` 触发重置。"
      actions={
        <button className="console-secondary" type="button" onClick={() => setShowKey((value) => !value)}>
          {showKey ? "隐藏 Key" : "显示 Key"}
        </button>
      }
    >
      <section className="console-stack">
        {message ? <div className="console-alert console-alert--success">{message}</div> : null}
        {error ? <div className="console-alert console-alert--error">{error}</div> : null}

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">当前 API Key</h2>
              <p className="console-panel-copy">
                这把 key 用于调用 `/v1/chat/completions` 等代理接口。重置后旧 key 会立刻失效。
              </p>
            </div>
          </div>

          <div className="console-stack">
            <div className="console-alert console-alert--info">
              <strong>当前值：</strong>{" "}
              <code style={{ fontFamily: "\"SFMono-Regular\", Menlo, monospace", wordBreak: "break-all" }}>
                {apiKey ? (showKey ? apiKey : maskAPIKey(apiKey)) : "浏览器里还没有 buyer_api_key"}
              </code>
            </div>

            <div className="console-form-actions">
              <button className="console-secondary" type="button" onClick={() => void handleCopy()} disabled={!apiKey}>
                复制 Key
              </button>
              <button className="console-primary" type="button" onClick={() => void handleReset()} disabled={loading}>
                {loading ? "重置中..." : "重置 API Key"}
              </button>
            </div>
          </div>
        </section>
      </section>
    </BuyerConsoleShell>
  );
}
