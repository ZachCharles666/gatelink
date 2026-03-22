"use client";

import axios from "axios";
import { useEffect, useMemo, useState } from "react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const ADMIN_TOKEN_KEY = "admin_token";

type PendingTopup = {
  id: string;
  buyer_id: string;
  email?: string;
  phone?: string;
  amount_usd: number;
  tx_hash: string;
  network: string;
  created_at: string;
};

type PendingSettlement = {
  id: string;
  seller_id: string;
  display_name: string;
  amount_usd: number;
  period_start: string;
  period_end: string;
  created_at: string;
};

function formatUSD(value: number) {
  return `$${value.toFixed(2)}`;
}

function formatDateTime(value: string) {
  return new Date(value).toLocaleString();
}

export default function AdminPage() {
  const [adminToken, setAdminToken] = useState("");
  const [suspendAccountID, setSuspendAccountID] = useState("");
  const [topups, setTopups] = useState<PendingTopup[]>([]);
  const [settlements, setSettlements] = useState<PendingSettlement[]>([]);
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const authHeaders = useMemo(
    () => ({
      Authorization: `Bearer ${adminToken}`,
    }),
    [adminToken],
  );

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    setAdminToken(window.localStorage.getItem(ADMIN_TOKEN_KEY) || "");
  }, []);

  async function fetchData(token = adminToken) {
    if (!token.trim()) {
      setTopups([]);
      setSettlements([]);
      return;
    }

    setLoading(true);
    setError("");

    try {
      const headers = { Authorization: `Bearer ${token}` };
      const [topupResponse, settlementResponse] = await Promise.all([
        axios.get(`${API_BASE}/api/v1/admin/topup/pending`, { headers }),
        axios.get(`${API_BASE}/api/v1/admin/settlements/pending`, { headers }),
      ]);
      setTopups(topupResponse.data.data?.records || []);
      setSettlements(settlementResponse.data.data?.settlements || []);
    } catch (fetchError) {
      if (axios.isAxiosError(fetchError)) {
        const apiMessage = fetchError.response?.data?.msg;
        setError(apiMessage || fetchError.message || "管理后台数据加载失败");
      } else if (fetchError instanceof Error) {
        setError(fetchError.message);
      } else {
        setError("管理后台数据加载失败");
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void fetchData();
  }, [adminToken]);

  function saveToken(value: string) {
    setAdminToken(value);
    if (typeof window !== "undefined") {
      if (value.trim()) {
        window.localStorage.setItem(ADMIN_TOKEN_KEY, value);
      } else {
        window.localStorage.removeItem(ADMIN_TOKEN_KEY);
      }
    }
  }

  async function confirmTopup(id: string) {
    setMessage("");
    setError("");
    try {
      const response = await axios.post(
        `${API_BASE}/api/v1/admin/topup/${id}/confirm`,
        {},
        { headers: authHeaders },
      );
      setMessage(response.data.data?.message || "充值已确认");
      await fetchData();
    } catch (actionError) {
      if (axios.isAxiosError(actionError)) {
        setError(actionError.response?.data?.msg || actionError.message);
      } else if (actionError instanceof Error) {
        setError(actionError.message);
      } else {
        setError("充值确认失败");
      }
    }
  }

  async function rejectTopup(id: string) {
    const reason = window.prompt("拒绝原因：", "未匹配到有效转账") || "";
    setMessage("");
    setError("");
    try {
      const response = await axios.post(
        `${API_BASE}/api/v1/admin/topup/${id}/reject`,
        { reason },
        { headers: authHeaders },
      );
      setMessage(response.data.data?.message || "充值已拒绝");
      await fetchData();
    } catch (actionError) {
      if (axios.isAxiosError(actionError)) {
        setError(actionError.response?.data?.msg || actionError.message);
      } else if (actionError instanceof Error) {
        setError(actionError.message);
      } else {
        setError("充值拒绝失败");
      }
    }
  }

  async function paySettlement(id: string) {
    const txHash = window.prompt("付款交易哈希：", "0xsettlement") || "";
    if (!txHash.trim()) {
      return;
    }

    setMessage("");
    setError("");
    try {
      const response = await axios.post(
        `${API_BASE}/api/v1/admin/settlements/${id}/pay`,
        { tx_hash: txHash.trim() },
        { headers: authHeaders },
      );
      setMessage(response.data.data?.message || "结算已完成");
      await fetchData();
    } catch (actionError) {
      if (axios.isAxiosError(actionError)) {
        setError(actionError.response?.data?.msg || actionError.message);
      } else if (actionError instanceof Error) {
        setError(actionError.message);
      } else {
        setError("结算付款失败");
      }
    }
  }

  async function forceSuspend(accountID: string) {
    if (!accountID.trim()) {
      setError("请先输入要暂停的账号 ID");
      return;
    }

    setMessage("");
    setError("");
    try {
      const response = await axios.post(
        `${API_BASE}/api/v1/admin/accounts/${accountID.trim()}/force-suspend`,
        {},
        { headers: authHeaders },
      );
      setMessage(`账号已强制暂停：${response.data.data?.account_id || accountID.trim()}`);
      setSuspendAccountID("");
    } catch (actionError) {
      if (axios.isAxiosError(actionError)) {
        setError(actionError.response?.data?.msg || actionError.message);
      } else if (actionError instanceof Error) {
        setError(actionError.message);
      } else {
        setError("账号暂停失败");
      }
    }
  }

  return (
    <main className="console-page">
      <div className="console-shell">
        <section className="console-hero">
          <div>
            <h1 className="console-title">管理后台</h1>
            <p className="console-subtitle">
              这里直接对接现有管理员接口：待审充值、待付结算，以及账号强制暂停。页面不会生成 token，需要你手动贴入 admin JWT。
            </p>
          </div>
          <div className="console-hero-actions">
            <button className="console-secondary" type="button" onClick={() => void fetchData()} disabled={loading}>
              {loading ? "刷新中..." : "刷新数据"}
            </button>
          </div>
        </section>

        {message ? <div className="console-alert console-alert--success">{message}</div> : null}
        {error ? <div className="console-alert console-alert--error">{error}</div> : null}

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">Admin JWT</h2>
              <p className="console-panel-copy">
                当前页面使用手动输入的 admin JWT 调管理员接口。清空输入框后会一并清除本地保存的 token。
              </p>
            </div>
          </div>

          <div className="console-form-grid">
            <label className="console-form-field console-form-field--full">
              <span className="console-form-label">Admin Token</span>
              <input
                className="console-form-input"
                type="password"
                value={adminToken}
                onChange={(event) => saveToken(event.target.value)}
                placeholder="输入 admin JWT..."
              />
            </label>
          </div>
        </section>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">强制暂停账号</h2>
              <p className="console-panel-copy">
                这个接口真实需要 `account_id`，不能直接用 `seller_id` 代替。输入真实账号 ID 后再执行暂停。
              </p>
            </div>
          </div>

          <div className="console-form-grid">
            <label className="console-form-field">
              <span className="console-form-label">Account ID</span>
              <input
                className="console-form-input"
                type="text"
                value={suspendAccountID}
                onChange={(event) => setSuspendAccountID(event.target.value)}
                placeholder="输入要强制暂停的账号 ID"
              />
            </label>
          </div>

          <div className="console-form-actions">
            <button className="console-secondary" type="button" onClick={() => void forceSuspend(suspendAccountID)}>
              强制暂停账号
            </button>
          </div>
        </section>

        <div className="metric-grid">
          <article className="metric-card">
            <div className="metric-label">待审充值</div>
            <div className="metric-value">{topups.length}</div>
            <div className="metric-note">来自 `GET /api/v1/admin/topup/pending`</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">待付结算</div>
            <div className="metric-value">{settlements.length}</div>
            <div className="metric-note">来自 `GET /api/v1/admin/settlements/pending`</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">认证状态</div>
            <div className="metric-value">{adminToken ? "已提供" : "未提供"}</div>
            <div className="metric-note">缺 token 时不会自动请求列表</div>
          </article>
          <article className="metric-card">
            <div className="metric-label">环境地址</div>
            <div className="metric-value" style={{ fontSize: "1.2rem" }}>{API_BASE}</div>
            <div className="metric-note">使用 `NEXT_PUBLIC_API_URL` 或默认本地地址</div>
          </article>
        </div>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">待审充值</h2>
              <p className="console-panel-copy">真实字段是 `records`，条目里包含 `amount_usd`、`tx_hash`、`email/phone` 和 `buyer_id`。</p>
            </div>
          </div>

          {topups.length === 0 ? (
            <div className="console-empty">暂无待审充值，或者当前 admin token 还未提供。</div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th>买家</th>
                    <th>金额</th>
                    <th>网络</th>
                    <th>交易哈希</th>
                    <th>创建时间</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {topups.map((topup) => (
                    <tr key={topup.id}>
                      <td>
                        <div>{topup.email || topup.phone || topup.buyer_id}</div>
                        <div className="console-id">{topup.buyer_id.slice(0, 12)}</div>
                      </td>
                      <td>{formatUSD(topup.amount_usd)}</td>
                      <td>{topup.network}</td>
                      <td className="console-id">{topup.tx_hash}</td>
                      <td>{formatDateTime(topup.created_at)}</td>
                      <td>
                        <div className="console-form-actions">
                          <button className="console-primary" type="button" onClick={() => void confirmTopup(topup.id)}>
                            确认
                          </button>
                          <button className="console-secondary" type="button" onClick={() => void rejectTopup(topup.id)}>
                            拒绝
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="console-panel">
          <div className="console-panel-header">
            <div>
              <h2 className="console-panel-title">待付结算</h2>
              <p className="console-panel-copy">真实金额字段是 `amount_usd`。管理员付款时需要输入链上哈希，页面会调用 `POST /api/v1/admin/settlements/:id/pay`。</p>
            </div>
          </div>

          {settlements.length === 0 ? (
            <div className="console-empty">暂无待付结算，或者当前 admin token 还未提供。</div>
          ) : (
            <div className="console-table-wrap">
              <table className="console-table">
                <thead>
                  <tr>
                    <th>卖家</th>
                    <th>金额</th>
                    <th>周期</th>
                    <th>创建时间</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {settlements.map((settlement) => (
                    <tr key={settlement.id}>
                      <td>
                        <div>{settlement.display_name || settlement.seller_id}</div>
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
                        <button className="console-primary" type="button" onClick={() => void paySettlement(settlement.id)}>
                          标记已付
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}
