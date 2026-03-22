"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { BuyerConsoleShell } from "@/components/buyer-console-shell";
import { buyerAPI, buyerStorage, type BuyerBalanceResponse } from "@/lib/api/buyer";

function formatUSD(value: number) {
  return `$${value.toFixed(4)}`;
}

function readBuyerNotice() {
  if (typeof window === "undefined") {
    return "";
  }
  const params = new URLSearchParams(window.location.search);
  if (params.get("registered") === "1") {
    return "欢迎进入买家控制台。你的 buyer token 和 buyer api_key 已经写入浏览器。";
  }
  if (params.get("topup") === "1") {
    return "充值申请已提交，可以在下方最近记录里继续跟踪审核状态。";
  }
  return "";
}

export default function BuyerDashboardPage() {
  const router = useRouter();
  const [balance, setBalance] = useState<BuyerBalanceResponse | null>(null);
  const [apiKey, setAPIKey] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function loadBalance() {
    setLoading(true);
    setError("");
    try {
      const data = await buyerAPI.getBalance();
      setBalance(data);
      setAPIKey(buyerStorage.getAPIKey() || "");
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : "余额信息加载失败";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!buyerStorage.getToken()) {
      router.replace("/buyer/login");
      return;
    }

    setNotice(readBuyerNotice());
    void loadBalance();
  }, [router]);

  return (
    <BuyerConsoleShell
      title="买家控制台"
      subtitle="这里先接通余额、累计消耗和 API Key 快速接入示例。余额与充值记录都按真实 buyer API 返回结构渲染。"
      actions={
        <>
          <button className="console-secondary" type="button" onClick={() => void loadBalance()} disabled={loading}>
            {loading ? "刷新中..." : "刷新余额"}
          </button>
          <Link className="console-primary" href="/buyer/topup">
            去充值
          </Link>
        </>
      }
    >
      <section className="console-stack">
        {notice ? <div className="console-alert console-alert--success">{notice}</div> : null}
        {error ? <div className="console-alert console-alert--error">{error}</div> : null}

        <div className="metric-grid">
          <article className="metric-card">
            <div className="metric-label">账户余额</div>
            <div className="metric-value">{formatUSD(balance?.balance_usd ?? 0)}</div>
            <div className="metric-note">余额不足时代理请求会被拦截</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">累计消耗</div>
            <div className="metric-value">{formatUSD(balance?.total_consumed_usd ?? 0)}</div>
            <div className="metric-note">这里展示 buyer 维度的累计扣费</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">账户等级</div>
            <div className="metric-value">{balance?.tier ?? "standard"}</div>
            <div className="metric-note">当前本地实现默认 standard</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">API Key 状态</div>
            <div className="metric-value">{apiKey ? "已就绪" : "未发现"}</div>
            <div className="metric-note">本地浏览器里保存的 buyer_api_key</div>
          </article>
        </div>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">快速接入示例</h2>
              <p className="console-panel-copy">
                这段示例直接展示当前浏览器中保存的 `buyer_api_key`。如果还是占位内容，先重新注册或登录一次即可。
              </p>
            </div>
          </div>
          <pre
            className="console-alert console-alert--info"
            style={{ whiteSpace: "pre-wrap", fontFamily: "\"SFMono-Regular\", Menlo, monospace" }}
          >{`curl -X POST http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer ${apiKey || "<your-buyer-api-key>"}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"Hello"}]}'`}</pre>
        </section>
      </section>
    </BuyerConsoleShell>
  );
}
