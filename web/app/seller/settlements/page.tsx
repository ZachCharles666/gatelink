"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { SellerConsoleShell } from "@/components/seller-console-shell";
import { authStorage } from "@/lib/api/client";
import { sellerAPI, type SellerSettlement, type SellerSettlementsResponse } from "@/lib/api/seller";

function formatUSD(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString();
}

function settlementStatusClass(status: string) {
  return `console-status console-status--${status === "paid" ? "active" : status}`;
}

export default function SettlementsPage() {
  const router = useRouter();
  const [data, setData] = useState<SellerSettlementsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");

  async function loadSettlements(nextPage: number) {
    setLoading(true);
    setError("");
    try {
      const response = await sellerAPI.getSettlements(nextPage);
      setData(response);
      setPage(nextPage);
    } catch (loadError) {
      const text = loadError instanceof Error ? loadError.message : "结算历史加载失败";
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
    void loadSettlements(1);
  }, [router]);

  const settlements: SellerSettlement[] = data?.settlements ?? [];
  const total = data?.total ?? 0;
  const pageSize = 20;
  const hasPrev = page > 1;
  const hasNext = page * pageSize < total;

  return (
    <SellerConsoleShell
      title="结算历史"
      subtitle="这里直接读取 `GET /api/v1/seller/settlements?page=n`，展示当前卖家的分页结算记录。"
      actions={
        <button className="console-secondary" type="button" onClick={() => void loadSettlements(page)} disabled={loading}>
          {loading ? "刷新中..." : "刷新列表"}
        </button>
      }
    >
      <section className="console-stack">
        {error ? <div className="console-alert console-alert--error">{error}</div> : null}

        <div className="metric-grid">
          <article className="metric-card">
            <div className="metric-label">当前页</div>
            <div className="metric-value">{page}</div>
            <div className="metric-note">每页最多 20 条</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">结算总数</div>
            <div className="metric-value">{total}</div>
            <div className="metric-note">来自 `total` 字段</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">本页条目</div>
            <div className="metric-value">{settlements.length}</div>
            <div className="metric-note">当前分页返回的记录数</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">最近状态</div>
            <div className="metric-value">{settlements[0]?.status ?? "none"}</div>
            <div className="metric-note">按当前 API 顺序显示</div>
          </article>
        </div>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">结算记录表</h2>
              <p className="console-panel-copy">
                真实金额字段是 `amount_usd`。如果状态是 `paid`，页面会一并展示付款时间和链上哈希。
              </p>
            </div>
          </div>

          {settlements.length === 0 && !loading ? (
            <div className="console-empty">还没有结算记录。收益页发起结算申请后，这里会开始累计记录。</div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th>结算单</th>
                    <th>金额</th>
                    <th>周期</th>
                    <th>创建时间</th>
                    <th>状态</th>
                    <th>付款信息</th>
                  </tr>
                </thead>
                <tbody>
                  {settlements.map((settlement) => (
                    <tr key={settlement.id}>
                      <td>
                        <div className="console-id">{settlement.id.slice(0, 16)}</div>
                        <div className="console-id">{settlement.seller_id.slice(0, 12)}</div>
                      </td>
                      <td>{formatUSD(settlement.amount_usd)}</td>
                      <td>
                        {formatDateTime(settlement.period_start)}
                        <br />
                        <span className="console-id">至 {formatDateTime(settlement.period_end)}</span>
                      </td>
                      <td>{formatDateTime(settlement.created_at)}</td>
                      <td>
                        <span className={settlementStatusClass(settlement.status)}>{settlement.status}</span>
                      </td>
                      <td>
                        {settlement.paid_at ? (
                          <>
                            <div>{formatDateTime(settlement.paid_at)}</div>
                            <div className="console-id">{settlement.tx_hash || "tx pending"}</div>
                          </>
                        ) : (
                          <span className="console-id">等待管理员付款</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="console-form-actions" style={{ marginTop: "1rem" }}>
            <button className="console-secondary" type="button" onClick={() => void loadSettlements(page - 1)} disabled={!hasPrev || loading}>
              上一页
            </button>
            <button className="console-secondary" type="button" onClick={() => void loadSettlements(page + 1)} disabled={!hasNext || loading}>
              下一页
            </button>
          </div>
        </section>
      </section>
    </SellerConsoleShell>
  );
}
