"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { SellerConsoleShell } from "@/components/seller-console-shell";
import { authStorage } from "@/lib/api/client";
import { sellerAPI, type SellerEarningsResponse, type SellerSettlement } from "@/lib/api/seller";

function formatUSD(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString();
}

function settlementStatusClass(status: string) {
  return `console-status console-status--${status === "paid" ? "active" : status}`;
}

function RecentSettlementCard({ settlement }: { settlement: SellerSettlement }) {
  return (
    <article className="metric-card">
      <div className="metric-label">{formatDate(settlement.created_at)}</div>
      <div className="metric-value">{formatUSD(settlement.amount_usd)}</div>
      <div className="metric-note">
        周期 {formatDate(settlement.period_start)} - {formatDate(settlement.period_end)}
      </div>
      <div style={{ marginTop: "0.75rem" }}>
        <span className={settlementStatusClass(settlement.status)}>{settlement.status}</span>
      </div>
    </article>
  );
}

export default function EarningsPage() {
  const router = useRouter();
  const [data, setData] = useState<SellerEarningsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [requesting, setRequesting] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function loadEarnings() {
    setLoading(true);
    setError("");
    try {
      const response = await sellerAPI.getEarnings();
      setData(response);
    } catch (loadError) {
      const text = loadError instanceof Error ? loadError.message : "收益数据加载失败";
      setError(text);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!authStorage.getToken()) {
      router.replace("/seller/login");
      return;
    }
    void loadEarnings();
  }, [router]);

  async function handleRequestSettlement() {
    if (!data) {
      return;
    }

    setRequesting(true);
    setMessage("");
    setError("");
    try {
      const response = await sellerAPI.requestSettlement();
      setMessage(response.message);
      await loadEarnings();
    } catch (submitError) {
      const text = submitError instanceof Error ? submitError.message : "结算申请失败";
      setError(text);
    } finally {
      setRequesting(false);
    }
  }

  const pendingUSD = data?.pending_usd ?? 0;
  const totalEarnedUSD = data?.total_earned_usd ?? 0;
  const recentSettlements = data?.settlements ?? [];

  return (
    <SellerConsoleShell
      title="收益概览"
      subtitle="这里读取 `GET /api/v1/seller/earnings`，展示待结算收益、累计已结算金额，以及最近几笔结算记录。"
      actions={
        <>
          <button className="console-secondary" type="button" onClick={() => void loadEarnings()} disabled={loading}>
            {loading ? "刷新中..." : "刷新数据"}
          </button>
          <Link className="console-primary" href="/seller/settlements">
            查看全部结算
          </Link>
        </>
      }
    >
      <section className="console-stack">
        {error ? <div className="console-alert console-alert--error">{error}</div> : null}
        {message ? <div className="console-alert console-alert--success">{message}</div> : null}

        <div className="metric-grid">
          <article className="metric-card">
            <div className="metric-label">待结算收益</div>
            <div className="metric-value">{formatUSD(pendingUSD)}</div>
            <div className="metric-note">满 $10 可发起结算申请</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">累计已结算</div>
            <div className="metric-value">{formatUSD(totalEarnedUSD)}</div>
            <div className="metric-note">管理员确认付款后累计到这里</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">最近结算数</div>
            <div className="metric-value">{recentSettlements.length}</div>
            <div className="metric-note">当前接口默认回看最近 5 条</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">结算状态</div>
            <div className="metric-value">{pendingUSD >= 10 ? "可申请" : "未达门槛"}</div>
            <div className="metric-note">最低结算金额 $10</div>
          </article>
        </div>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">申请结算</h2>
              <p className="console-panel-copy">
                这一步会调用 `POST /api/v1/seller/settlements/request`。如果待结算金额不足 $10，后端会直接返回明确错误。
              </p>
            </div>
          </div>
          <div className="console-stack">
            <div className={pendingUSD >= 10 ? "console-alert console-alert--info" : "console-alert console-alert--warn"}>
              当前待结算金额为 {formatUSD(pendingUSD)}。
              {pendingUSD >= 10 ? " 已达到最低门槛，可以直接提交。" : " 还没达到 $10 的最低门槛。"}
            </div>
            <div className="console-form-actions">
              <button
                className="console-primary"
                type="button"
                onClick={() => void handleRequestSettlement()}
                disabled={requesting || pendingUSD < 10 || loading}
              >
                {requesting ? "申请中..." : `申请结算 ${formatUSD(pendingUSD)}`}
              </button>
            </div>
          </div>
        </section>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">最近结算记录</h2>
              <p className="console-panel-copy">这里显示收益页随接口一起返回的最近结算快照，方便快速确认付款状态。</p>
            </div>
          </div>
          {recentSettlements.length === 0 && !loading ? (
            <div className="console-empty">
              还没有结算记录。先完成几次 dispatch 记账，再从这里申请第一笔结算。
            </div>
          ) : (
            <div className="metric-grid">
              {recentSettlements.map((settlement) => (
                <RecentSettlementCard key={settlement.id} settlement={settlement} />
              ))}
            </div>
          )}
        </section>
      </section>
    </SellerConsoleShell>
  );
}
