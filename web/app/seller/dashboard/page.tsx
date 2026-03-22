"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { SellerConsoleShell } from "@/components/seller-console-shell";
import { authStorage } from "@/lib/api/client";
import { sellerAPI, type SellerAccount } from "@/lib/api/seller";

type LoadState = "idle" | "loading";

function formatUSD(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}

function statusClass(status: string) {
  return `console-status console-status--${status}`;
}

function readNotice() {
  if (typeof window === "undefined") {
    return "";
  }

  const params = new URLSearchParams(window.location.search);
  if (params.get("welcome") === "1") {
    return "欢迎回来，卖家控制台已经接通。下一步可以直接添加第一条托管账号。";
  }
  if (params.get("added") === "1") {
    return "账号信息已经提交，列表已刷新。格式检查结果可以在下方账号状态里继续确认。";
  }
  return "";
}

export default function SellerDashboardPage() {
  const router = useRouter();
  const [accounts, setAccounts] = useState<SellerAccount[]>([]);
  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function loadAccounts() {
    setLoadState("loading");
    setError("");
    try {
      const data = await sellerAPI.getAccounts();
      setAccounts(data.accounts ?? []);
    } catch (loadError) {
      const message = loadError instanceof Error ? loadError.message : "账号列表加载失败";
      setError(message);
    } finally {
      setLoadState("idle");
    }
  }

  useEffect(() => {
    if (!authStorage.getToken()) {
      router.replace("/seller/login");
      return;
    }

    setNotice(readNotice());
    void loadAccounts();
  }, [router]);

  const totalAuthorized = accounts.reduce((sum, account) => sum + account.authorized_credits_usd, 0);
  const totalConsumed = accounts.reduce((sum, account) => sum + account.consumed_credits_usd, 0);
  const activeCount = accounts.filter((account) => account.status === "active").length;
  const pendingCount = accounts.filter((account) => account.status === "pending_verify").length;

  return (
    <SellerConsoleShell
      title="账号控制台"
      subtitle="这里汇总你已托管的账号、授权额度和当前状态。当前实现基于真实 seller API 字段，方便下一阶段继续接收益和结算页。"
      actions={
        <>
          <button className="console-secondary" type="button" onClick={() => void loadAccounts()} disabled={loadState === "loading"}>
            {loadState === "loading" ? "刷新中..." : "刷新列表"}
          </button>
          <Link className="console-primary" href="/seller/accounts/add">
            + 添加账号
          </Link>
        </>
      }
    >
      <section className="console-stack">
        {notice ? <div className="console-alert console-alert--success">{notice}</div> : null}
        {error ? <div className="console-alert console-alert--error">{error}</div> : null}

        <div className="metric-grid">
          <article className="metric-card">
            <div className="metric-label">账号总数</div>
            <div className="metric-value">{accounts.length}</div>
            <div className="metric-note">当前已托管的账号数量</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">Active 账号</div>
            <div className="metric-value">{activeCount}</div>
            <div className="metric-note">可进入真实调度的账号数</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">待验证账号</div>
            <div className="metric-value">{pendingCount}</div>
            <div className="metric-note">格式检查通过后会先进入这个状态</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">总授权额度</div>
            <div className="metric-value">{formatUSD(totalAuthorized)}</div>
            <div className="metric-note">已消耗 {formatUSD(totalConsumed)}</div>
          </article>
        </div>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">托管账号列表</h2>
              <p className="console-panel-copy">
                实时读取 `GET /api/v1/seller/accounts`。当前 Day 1-2 先把列表和添加账号跑通，详情页会在后续阶段补上。
              </p>
            </div>
          </div>

          {accounts.length === 0 && loadState === "idle" ? (
            <div className="console-empty">
              还没有托管账号。<Link href="/seller/accounts/add">现在添加第一条账号</Link>
            </div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th>账号</th>
                    <th>状态</th>
                    <th>健康度</th>
                    <th>授权 / 已消耗</th>
                    <th>期望回收率</th>
                    <th>到期时间</th>
                  </tr>
                </thead>
                <tbody>
                  {accounts.map((account) => {
                    const consumedRatio =
                      account.authorized_credits_usd > 0
                        ? Math.min((account.consumed_credits_usd / account.authorized_credits_usd) * 100, 100)
                        : 0;

                    return (
                      <tr key={account.id}>
                        <td>
                          <div className="console-vendor">{account.vendor}</div>
                          <div className="console-id">{account.id.slice(0, 12)}</div>
                        </td>
                        <td>
                          <span className={statusClass(account.status)}>{account.status}</span>
                        </td>
                        <td>
                          <strong>{account.health_score}</strong>
                          <div className="console-progress">
                            <span style={{ width: `${Math.max(0, Math.min(account.health_score, 100))}%` }} />
                          </div>
                        </td>
                        <td>
                          <div>{formatUSD(account.authorized_credits_usd)}</div>
                          <div className="console-id">已消耗 {formatUSD(account.consumed_credits_usd)}</div>
                        </td>
                        <td>{(account.expected_rate * 100).toFixed(0)}%</td>
                        <td>{formatDate(account.expire_at)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </section>
    </SellerConsoleShell>
  );
}
