"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState, useTransition } from "react";
import { BuyerConsoleShell } from "@/components/buyer-console-shell";
import { buyerAPI, buyerStorage, type BuyerTopupRecord } from "@/lib/api/buyer";

const FX_NOTE = "管理员确认后才会真正入账。当前页面按 USD 提交，网络仅支持 TRC20 / ERC20。";

function formatUSD(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString();
}

function topupStatusClass(status: string) {
  if (status === "confirmed") {
    return "console-status console-status--active";
  }
  if (status === "rejected") {
    return "console-status console-status--revoked";
  }
  return "console-status console-status--pending_verify";
}

export default function BuyerTopupPage() {
  const router = useRouter();
  const [isNavigating, startTransition] = useTransition();
  const [amountUSD, setAmountUSD] = useState("100");
  const [txHash, setTxHash] = useState("");
  const [network, setNetwork] = useState<"TRC20" | "ERC20">("TRC20");
  const [records, setRecords] = useState<BuyerTopupRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function loadRecords() {
    setLoading(true);
    setError("");
    try {
      const data = await buyerAPI.getTopupRecords();
      setRecords(data.records ?? []);
      setTotal(data.total ?? 0);
    } catch (loadError) {
      const text = loadError instanceof Error ? loadError.message : "充值记录加载失败";
      setError(text);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!buyerStorage.getToken()) {
      router.replace("/buyer/login");
      return;
    }
    void loadRecords();
  }, [router]);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setMessage("");
    setError("");

    try {
      const response = await buyerAPI.submitTopup({
        amount_usd: Number(amountUSD),
        tx_hash: txHash.trim(),
        network,
      });
      setMessage(response.message);
      setAmountUSD("100");
      setTxHash("");
      await loadRecords();
      startTransition(() => {
        router.replace("/buyer/topup?submitted=1");
      });
    } catch (submitError) {
      const text = submitError instanceof Error ? submitError.message : "提交充值申请失败";
      setError(text);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <BuyerConsoleShell
      title="充值"
      subtitle="这里直接调用 `POST /api/v1/buyer/topup` 和 `GET /api/v1/buyer/topup/records`，方便提交充值申请并跟踪审核状态。"
      actions={
        <>
          <button className="console-secondary" type="button" onClick={() => void loadRecords()} disabled={loading}>
            {loading ? "刷新中..." : "刷新记录"}
          </button>
          <Link className="console-primary" href="/buyer/dashboard">
            返回控制台
          </Link>
        </>
      }
    >
      <section className="console-stack">
        <div className="console-alert console-alert--info">{FX_NOTE}</div>
        {message ? <div className="console-alert console-alert--success">{message}</div> : null}
        {error ? <div className="console-alert console-alert--error">{error}</div> : null}

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">提交充值申请</h2>
              <p className="console-panel-copy">录入金额、网络和交易哈希后，后台会生成一条 `pending` 记录，等待管理员审核。</p>
            </div>
          </div>

          <form className="console-form" onSubmit={handleSubmit}>
            <div className="console-form-grid">
              <label className="console-form-field">
                <span className="console-form-label">充值金额（USD）</span>
                <input
                  className="console-form-input"
                  type="number"
                  min="0.01"
                  step="0.01"
                  value={amountUSD}
                  onChange={(event) => setAmountUSD(event.target.value)}
                  required
                />
              </label>

              <label className="console-form-field">
                <span className="console-form-label">网络</span>
                <select
                  className="console-form-select"
                  value={network}
                  onChange={(event) => setNetwork(event.target.value as "TRC20" | "ERC20")}
                >
                  <option value="TRC20">TRC20</option>
                  <option value="ERC20">ERC20</option>
                </select>
              </label>

              <label className="console-form-field console-form-field--full">
                <span className="console-form-label">交易哈希（TxHash）</span>
                <input
                  className="console-form-input"
                  type="text"
                  placeholder="0x..."
                  value={txHash}
                  onChange={(event) => setTxHash(event.target.value)}
                  required
                />
                <span className="console-form-hint">后端会按 `tx_hash` 去重，重复提交会返回明确错误。</span>
              </label>
            </div>

            <div className="console-form-actions">
              <button className="console-primary" type="submit" disabled={submitting || isNavigating}>
                {submitting || isNavigating ? "提交中..." : "提交充值申请"}
              </button>
            </div>
          </form>
        </section>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">充值记录</h2>
              <p className="console-panel-copy">当前总记录数：{total}。状态会在管理员确认后从 `pending` 变成 `confirmed` 或 `rejected`。</p>
            </div>
          </div>

          {records.length === 0 && !loading ? (
            <div className="console-empty">还没有充值记录。先提交一笔测试充值，管理员确认后余额页才会增加。</div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th>金额</th>
                    <th>网络</th>
                    <th>交易哈希</th>
                    <th>创建时间</th>
                    <th>状态</th>
                    <th>备注</th>
                  </tr>
                </thead>
                <tbody>
                  {records.map((record) => (
                    <tr key={record.id}>
                      <td>{formatUSD(record.amount_usd)}</td>
                      <td>{record.network}</td>
                      <td className="console-id">{record.tx_hash}</td>
                      <td>{formatDateTime(record.created_at)}</td>
                      <td>
                        <span className={topupStatusClass(record.status)}>{record.status}</span>
                      </td>
                      <td>{record.notes || "-"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </section>
    </BuyerConsoleShell>
  );
}
