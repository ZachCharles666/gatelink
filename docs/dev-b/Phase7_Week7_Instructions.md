# Phase 7 · Week 7 执行指令
**主题：买家前端完成 + 管理后台**
**工时预算：15 小时（3h/天 × 5 天）**
**完成标准：买家控制台 5 页完成，管理后台充值+结算审核页完成**

---

## 前置检查

```bash
# 确认 Week 6 验收通过
bash scripts/week6_verify.sh

# 确认前端正常构建
cd web && npm run build
```

---

## Day 1-2 · 买家控制台主页 + 余额充值

### Step 1：买家控制台主页

```bash
mkdir -p web/app/buyer/dashboard

cat > web/app/buyer/dashboard/page.tsx << 'EOF'
'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { buyerAPI } from '@/lib/api/buyer'

interface BalanceData {
  balance_usd: number
  total_consumed_usd: number
  tier: string
}

export default function BuyerDashboardPage() {
  const router = useRouter()
  const [balance, setBalance] = useState<BalanceData | null>(null)
  const [apiKey, setApiKey] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!localStorage.getItem('buyer_token')) {
      router.push('/buyer/login')
      return
    }
    setApiKey(localStorage.getItem('buyer_api_key') || '')
    buyerAPI.getBalance()
      .then((d: any) => setBalance(d))
      .catch(() => router.push('/buyer/login'))
      .finally(() => setLoading(false))
  }, [router])

  if (loading) return <div className="min-h-screen flex items-center justify-center">加载中...</div>

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow-sm px-6 py-4 flex justify-between items-center">
        <h1 className="text-xl font-bold text-gray-900">买家控制台</h1>
        <div className="flex gap-4">
          <a href="/buyer/topup" className="text-indigo-600 hover:underline">充值</a>
          <a href="/buyer/usage" className="text-indigo-600 hover:underline">用量明细</a>
          <a href="/buyer/apikey" className="text-indigo-600 hover:underline">API Key</a>
          <button onClick={() => { localStorage.removeItem('buyer_token'); router.push('/buyer/login') }}
            className="text-gray-500 hover:text-gray-700 text-sm">退出</button>
        </div>
      </nav>

      <main className="max-w-4xl mx-auto px-6 py-8 space-y-6">
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-500">账户余额</p>
            <p className="text-3xl font-bold text-indigo-600 mt-1">
              ${balance?.balance_usd.toFixed(4) || '0.0000'}
            </p>
            <a href="/buyer/topup" className="text-xs text-indigo-600 hover:underline mt-2 block">充值 →</a>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-500">累计消耗</p>
            <p className="text-3xl font-bold text-gray-900 mt-1">
              ${balance?.total_consumed_usd.toFixed(4) || '0.0000'}
            </p>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-500">账户等级</p>
            <p className="text-2xl font-bold text-gray-700 mt-1 capitalize">{balance?.tier || 'standard'}</p>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="font-semibold text-gray-800 mb-3">快速接入</h2>
          <p className="text-sm text-gray-500 mb-2">使用以下格式调用代理 API：</p>
          <pre className="bg-gray-50 border rounded p-3 text-xs font-mono overflow-x-auto">{`curl -X POST http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer ${apiKey || '<your-api-key>'}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"Hello"}]}'`}</pre>
        </div>
      </main>
    </div>
  )
}
EOF
```

### Step 2：充值页

```bash
mkdir -p web/app/buyer/topup

cat > web/app/buyer/topup/page.tsx << 'EOF'
'use client'

import { useEffect, useState } from 'react'
import { buyerAPI } from '@/lib/api/buyer'

const EXCHANGE_RATE = 7.25 // USDT to USD（示意，实际汇率应从后端读取）

export default function TopupPage() {
  const [amountUSD, setAmountUSD] = useState('')
  const [txHash, setTxHash] = useState('')
  const [network, setNetwork] = useState('TRC20')
  const [records, setRecords] = useState<any[]>([])
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')

  useEffect(() => {
    buyerAPI.getTopupRecords().then((d: any) => setRecords(d.records || []))
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setMsg('')
    try {
      await buyerAPI.submitTopup(parseFloat(amountUSD), txHash, network)
      setMsg('充值申请已提交，等待管理员审核（通常 1-24 小时）')
      setAmountUSD('')
      setTxHash('')
      buyerAPI.getTopupRecords().then((d: any) => setRecords(d.records || []))
    } catch (err: any) {
      setMsg(err.message || '提交失败')
    } finally {
      setLoading(false)
    }
  }

  const statusText: Record<string, string> = {
    pending: '审核中',
    confirmed: '已确认',
    rejected: '已拒绝',
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-lg mx-auto px-6 space-y-6">
        <div className="flex items-center">
          <a href="/buyer/dashboard" className="text-indigo-600 hover:underline mr-4">← 返回</a>
          <h1 className="text-xl font-bold text-gray-900">充值</h1>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="font-semibold text-gray-800 mb-4">提交充值申请</h2>
          <div className="bg-blue-50 border border-blue-200 rounded p-3 mb-4 text-sm text-blue-800">
            请转账 USDT 后，在此填写交易哈希，管理员核实后会为您充值。
          </div>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">充值金额（USD）</label>
              <input type="number" value={amountUSD} onChange={(e) => setAmountUSD(e.target.value)}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                min="10" step="0.01" required />
              {amountUSD && <p className="text-xs text-gray-500 mt-1">
                约需转账 {(parseFloat(amountUSD) * EXCHANGE_RATE).toFixed(2)} USDT
              </p>}
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">网络</label>
              <select value={network} onChange={(e) => setNetwork(e.target.value)}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500">
                <option value="TRC20">TRC20（TRON）</option>
                <option value="ERC20">ERC20（Ethereum）</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">交易哈希（TxHash）</label>
              <input type="text" value={txHash} onChange={(e) => setTxHash(e.target.value)}
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                placeholder="0x..." required />
            </div>
            {msg && <p className={`text-sm ${msg.includes('失败') || msg.includes('error') ? 'text-red-500' : 'text-green-600'}`}>{msg}</p>}
            <button type="submit" disabled={loading}
              className="w-full bg-indigo-600 text-white py-2 px-4 rounded-md hover:bg-indigo-700 disabled:opacity-50">
              {loading ? '提交中...' : '提交充值申请'}
            </button>
          </form>
        </div>

        {records.length > 0 && (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="font-semibold text-gray-800 mb-4">充值记录</h2>
            <div className="space-y-2">
              {records.map((r: any) => (
                <div key={r.id} className="flex justify-between items-center py-2 border-b last:border-0">
                  <div>
                    <p className="text-sm font-medium">${r.amount_usd.toFixed(2)} ({r.network})</p>
                    <p className="text-xs text-gray-400">{new Date(r.created_at).toLocaleString()}</p>
                  </div>
                  <span className={`text-xs px-2 py-1 rounded ${r.status === 'confirmed' ? 'bg-green-100 text-green-800' : r.status === 'rejected' ? 'bg-red-100 text-red-800' : 'bg-yellow-100 text-yellow-800'}`}>
                    {statusText[r.status] || r.status}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
EOF
```

---

## Day 3 · 买家用量明细 + API Key 管理

### Step 1：用量明细页

```bash
mkdir -p web/app/buyer/usage

cat > web/app/buyer/usage/page.tsx << 'EOF'
'use client'

import { useEffect, useState } from 'react'
import { buyerAPI } from '@/lib/api/buyer'

export default function UsagePage() {
  const [records, setRecords] = useState<any[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)

  useEffect(() => {
    buyerAPI.getUsage(page).then((d: any) => {
      setRecords(d.records || [])
      setTotal(d.total || 0)
    })
  }, [page])

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-6">
        <div className="flex items-center mb-6">
          <a href="/buyer/dashboard" className="text-indigo-600 hover:underline mr-4">← 返回</a>
          <h1 className="text-xl font-bold text-gray-900">用量明细</h1>
          <span className="ml-3 text-sm text-gray-500">共 {total} 条</span>
        </div>

        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {['厂商', '模型', '输入 Token', '输出 Token', '实际成本', '买家付款', '时间'].map(h => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {records.length === 0 ? (
                <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-400 text-sm">暂无用量记录</td></tr>
              ) : records.map((r: any, i: number) => (
                <tr key={i} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm text-gray-700">{r.vendor}</td>
                  <td className="px-4 py-3 text-sm text-gray-600 font-mono text-xs">{r.model}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">{r.input_tokens.toLocaleString()}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">{r.output_tokens.toLocaleString()}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">${r.cost_usd.toFixed(6)}</td>
                  <td className="px-4 py-3 text-sm font-medium text-indigo-600">${r.buyer_charged_usd.toFixed(6)}</td>
                  <td className="px-4 py-3 text-xs text-gray-400">{new Date(r.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {total > 20 && (
          <div className="flex justify-center gap-2 mt-4">
            <button onClick={() => setPage(p => Math.max(1, p-1))} disabled={page === 1}
              className="px-3 py-1 border rounded disabled:opacity-50">上一页</button>
            <span className="px-3 py-1 text-sm text-gray-600">第 {page} 页</span>
            <button onClick={() => setPage(p => p+1)} disabled={page * 20 >= total}
              className="px-3 py-1 border rounded disabled:opacity-50">下一页</button>
          </div>
        )}
      </div>
    </div>
  )
}
EOF
```

### Step 2：API Key 管理页

```bash
mkdir -p web/app/buyer/apikey

cat > web/app/buyer/apikey/page.tsx << 'EOF'
'use client'

import { useState } from 'react'
import { buyerAPI } from '@/lib/api/buyer'

export default function APIKeyPage() {
  const [apiKey, setApiKey] = useState(localStorage.getItem('buyer_api_key') || '')
  const [newKey, setNewKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState('')
  const [showKey, setShowKey] = useState(false)

  const handleReset = async () => {
    if (!confirm('重置后旧 Key 立即失效，确认继续？')) return
    setLoading(true)
    setMsg('')
    try {
      const data = await buyerAPI.resetAPIKey() as any
      setNewKey(data.api_key)
      localStorage.setItem('buyer_api_key', data.api_key)
      setMsg('API Key 已重置，请立即保存新 Key')
    } catch (err: any) {
      setMsg(err.message || '重置失败')
    } finally {
      setLoading(false)
    }
  }

  const displayKey = newKey || apiKey

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-lg mx-auto px-6">
        <div className="flex items-center mb-6">
          <a href="/buyer/dashboard" className="text-indigo-600 hover:underline mr-4">← 返回</a>
          <h1 className="text-xl font-bold text-gray-900">API Key 管理</h1>
        </div>

        <div className="bg-white rounded-lg shadow p-6 space-y-6">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">当前 API Key</label>
            <div className="flex items-center gap-2">
              <code className="flex-1 bg-gray-50 border rounded px-3 py-2 text-xs font-mono break-all">
                {showKey ? displayKey : displayKey.slice(0, 8) + '...' + displayKey.slice(-4)}
              </code>
              <button onClick={() => setShowKey(!showKey)}
                className="text-xs text-gray-500 hover:text-gray-700 px-2 py-1 border rounded">
                {showKey ? '隐藏' : '显示'}
              </button>
              <button onClick={() => { navigator.clipboard.writeText(displayKey); setMsg('已复制') }}
                className="text-xs text-indigo-600 hover:text-indigo-800 px-2 py-1 border rounded">
                复制
              </button>
            </div>
          </div>

          <div className="bg-gray-50 rounded p-4 text-sm">
            <p className="font-medium text-gray-700 mb-2">使用方式</p>
            <code className="text-xs block">Authorization: Bearer {displayKey.slice(0, 8)}...</code>
          </div>

          {newKey && (
            <div className="bg-yellow-50 border border-yellow-300 rounded p-4">
              <p className="text-sm font-medium text-yellow-800">⚠️ 新 Key 已生成，请立即保存，此后不再显示完整 Key</p>
              <code className="block text-xs font-mono mt-2 break-all">{newKey}</code>
            </div>
          )}

          {msg && <p className={`text-sm ${msg.includes('失败') ? 'text-red-500' : 'text-green-600'}`}>{msg}</p>}

          <button onClick={handleReset} disabled={loading}
            className="w-full bg-red-50 text-red-600 border border-red-200 py-2 px-4 rounded-md hover:bg-red-100 disabled:opacity-50">
            {loading ? '重置中...' : '重置 API Key（旧 Key 立即失效）'}
          </button>
        </div>
      </div>
    </div>
  )
}
EOF
```

---

## Day 4-5 · 管理后台

### Step 1：管理后台充值审核页

```bash
mkdir -p web/app/admin

cat > web/app/admin/page.tsx << 'EOF'
'use client'

import { useEffect, useState } from 'react'
import axios from 'axios'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

export default function AdminPage() {
  const [topups, setTopups] = useState<any[]>([])
  const [settlements, setSettlements] = useState<any[]>([])
  const [adminToken, setAdminToken] = useState('')
  const [msg, setMsg] = useState('')

  const authHeader = () => ({ Authorization: `Bearer ${adminToken}` })

  const fetchData = () => {
    if (!adminToken) return
    axios.get(`${API_BASE}/api/v1/admin/topup/pending`, { headers: authHeader() })
      .then(r => setTopups(r.data.data?.records || []))
    axios.get(`${API_BASE}/api/v1/admin/settlements/pending`, { headers: authHeader() })
      .then(r => setSettlements(r.data.data?.settlements || []))
  }

  useEffect(fetchData, [adminToken])

  const confirmTopup = async (id: string) => {
    await axios.post(`${API_BASE}/api/v1/admin/topup/${id}/confirm`, {}, { headers: authHeader() })
    setMsg('充值已确认')
    fetchData()
  }

  const rejectTopup = async (id: string) => {
    const reason = prompt('拒绝原因：')
    await axios.post(`${API_BASE}/api/v1/admin/topup/${id}/reject`, { reason }, { headers: authHeader() })
    setMsg('充值已拒绝')
    fetchData()
  }

  const paySsettlement = async (id: string) => {
    const txHash = prompt('付款交易哈希：')
    if (!txHash) return
    await axios.post(`${API_BASE}/api/v1/admin/settlements/${id}/pay`, { tx_hash: txHash }, { headers: authHeader() })
    setMsg('结算已完成')
    fetchData()
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-4xl mx-auto px-6 space-y-6">
        <h1 className="text-2xl font-bold text-gray-900">管理后台</h1>

        <div className="bg-white rounded-lg shadow p-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Admin JWT Token</label>
          <input type="password" value={adminToken} onChange={(e) => setAdminToken(e.target.value)}
            className="w-full border rounded px-3 py-2 text-sm" placeholder="输入 admin token..." />
        </div>

        {msg && <div className="bg-green-50 border border-green-200 rounded p-3 text-sm text-green-700">{msg}</div>}

        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="font-semibold text-gray-800 mb-4">待审充值（{topups.length} 条）</h2>
          {topups.length === 0 ? <p className="text-gray-400 text-sm">暂无待审充值</p> : (
            <div className="space-y-2">
              {topups.map((t: any) => (
                <div key={t.id} className="flex items-center justify-between p-3 bg-gray-50 rounded">
                  <div>
                    <p className="text-sm font-medium">${t.amount_usd} ({t.network})</p>
                    <p className="text-xs text-gray-500">{t.email} · {t.tx_hash.slice(0, 20)}...</p>
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => confirmTopup(t.id)}
                      className="text-xs bg-green-600 text-white px-3 py-1 rounded hover:bg-green-700">确认</button>
                    <button onClick={() => rejectTopup(t.id)}
                      className="text-xs bg-red-600 text-white px-3 py-1 rounded hover:bg-red-700">拒绝</button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="font-semibold text-gray-800 mb-4">待付结算（{settlements.length} 条）</h2>
          {settlements.length === 0 ? <p className="text-gray-400 text-sm">暂无待付结算</p> : (
            <div className="space-y-2">
              {settlements.map((s: any) => (
                <div key={s.id} className="flex items-center justify-between p-3 bg-gray-50 rounded">
                  <div>
                    <p className="text-sm font-medium">${s.seller_earn_usd.toFixed(2)}</p>
                    <p className="text-xs text-gray-500">{s.display_name} · {new Date(s.created_at).toLocaleDateString()}</p>
                  </div>
                  <button onClick={() => paySsettlement(s.id)}
                    className="text-xs bg-blue-600 text-white px-3 py-1 rounded hover:bg-blue-700">标记已付</button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
EOF
```

**✅ Week 7 验收脚本**

```bash
cat > scripts/week7_verify.sh << 'EOF'
#!/bin/bash
PASS=0; FAIL=0
GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'

check() {
    if eval "$2" | grep -q "$3"; then
        echo -e "${GREEN}✅ PASS${NC} $1"; PASS=$((PASS+1))
    else
        echo -e "${RED}❌ FAIL${NC} $1"; FAIL=$((FAIL+1))
    fi
}

check "买家控制台" "curl -s http://localhost:3000/buyer/dashboard" '买家控制台\|登录'
check "买家充值页" "curl -s http://localhost:3000/buyer/topup" '充值申请\|买家充值\|登录'
check "买家用量页" "curl -s http://localhost:3000/buyer/usage" '用量明细\|登录'
check "买家 API Key 页" "curl -s http://localhost:3000/buyer/apikey" 'API Key\|登录'
check "管理后台页" "curl -s http://localhost:3000/admin" '管理后台'
check "前端构建成功" "cd web && npm run build 2>&1 | tail -10" 'Route\|compiled\|success'

echo -e "\n通过 $PASS，失败 $FAIL"
[ $FAIL -eq 0 ] && echo -e "${GREEN}🎉 Week 7 全部验收通过！${NC}" || exit 1
EOF
chmod +x scripts/week7_verify.sh
bash scripts/week7_verify.sh
```

**✅ Week 7 完成标准**
- [ ] 买家控制台（余额概览 + 快速接入示例）
- [ ] 买家充值页（提交申请 + 查看记录）
- [ ] 买家用量明细页（分页展示）
- [ ] 买家 API Key 管理页（显示/复制/重置）
- [ ] 管理后台（充值审核 + 结算付款）
- [ ] 前端 `npm run build` 无错误

## 下周预告（Week 8）

Week 8 实现：
- 全链路压测 + 性能优化
- MVP 验收（所有 8 周完成标准核对）
- 生产 Dockerfile + docker-compose 最终版

前置条件：Week 7 所有前端页面通过构建
