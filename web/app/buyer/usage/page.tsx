"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { BuyerConsoleShell } from "@/components/buyer-console-shell";
import { buyerAPI, buyerStorage, type BuyerUsageRecord } from "@/lib/api/buyer";

function formatUSD(value: number) {
  return `$${value.toFixed(6)}`;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString();
}

export default function BuyerUsagePage() {
  const router = useRouter();
  const [records, setRecords] = useState<BuyerUsageRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function loadUsage(nextPage: number) {
    setLoading(true);
    setError("");
    try {
      const data = await buyerAPI.getUsage(nextPage);
      setRecords(data.records ?? []);
      setTotal(data.total ?? 0);
      setPage(nextPage);
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : "用量记录加载失败";
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
    void loadUsage(1);
  }, [router]);

  const hasPrev = page > 1;
  const hasNext = page * 20 < total;

  return (
    <BuyerConsoleShell
      title="用量明细"
      subtitle="这里直接读取 `GET /api/v1/buyer/usage?page=n`。当前表格按真实后端返回的 vendor/model/token/cost 字段渲染。"
      actions={
        <button className="console-secondary" type="button" onClick={() => void loadUsage(page)} disabled={loading}>
          {loading ? "刷新中..." : "刷新记录"}
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
            <div className="metric-label">总记录数</div>
            <div className="metric-value">{total}</div>
            <div className="metric-note">来自后端 `total` 字段</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">本页记录</div>
            <div className="metric-value">{records.length}</div>
            <div className="metric-note">当前分页返回的条数</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">最近厂商</div>
            <div className="metric-value">{records[0]?.vendor ?? "none"}</div>
            <div className="metric-note">按当前 API 顺序显示</div>
          </article>
        </div>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">请求与扣费记录</h2>
              <p className="console-panel-copy">
                这里展示 buyer 维度的模型调用消耗。`cost_usd` 是底层成本，`buyer_charged_usd` 是平台对买家的实际扣费。
              </p>
            </div>
          </div>

          {records.length === 0 && !loading ? (
            <div className="console-empty">暂无用量记录。等真实 dispatch 跑起来后，这里会开始累计数据。</div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th>厂商</th>
                    <th>模型</th>
                    <th>输入 Token</th>
                    <th>输出 Token</th>
                    <th>实际成本</th>
                    <th>买家扣费</th>
                    <th>时间</th>
                  </tr>
                </thead>
                <tbody>
                  {records.map((record, index) => (
                    <tr key={`${record.created_at}-${index}`}>
                      <td>{record.vendor}</td>
                      <td className="console-id">{record.model}</td>
                      <td>{record.input_tokens.toLocaleString()}</td>
                      <td>{record.output_tokens.toLocaleString()}</td>
                      <td>{formatUSD(record.cost_usd)}</td>
                      <td>{formatUSD(record.buyer_charged_usd)}</td>
                      <td>{formatDateTime(record.created_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div className="console-form-actions" style={{ marginTop: "1rem" }}>
            <button className="console-secondary" type="button" onClick={() => void loadUsage(page - 1)} disabled={!hasPrev || loading}>
              上一页
            </button>
            <button className="console-secondary" type="button" onClick={() => void loadUsage(page + 1)} disabled={!hasNext || loading}>
              下一页
            </button>
          </div>
        </section>
      </section>
    </BuyerConsoleShell>
  );
}
