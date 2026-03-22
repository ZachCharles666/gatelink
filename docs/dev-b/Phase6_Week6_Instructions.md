# Phase 6 · Week 6 执行指令
**主题：卖家前端完成 + 买家前端启动**
**工时预算：18 小时（3-4h/天 × 5 天）**
**完成标准：卖家控制台 5 页全部完成可用，买家前端注册/登录页完成**

---

## 前置检查

```bash
# 确认 Week 5 验收通过
bash scripts/week5_verify.sh

# 确认前端项目正常启动
cd web && npm run dev &
sleep 5
curl -s http://localhost:3000/seller/login | grep -q "卖家登录" && echo "OK" || echo "FAIL"
```

---

## Day 1-2 · 卖家账号管理页

**目标：账号列表、添加账号表单**

### Step 1：账号列表页

```bash
mkdir -p web/app/seller/dashboard
mkdir -p web/app/seller/accounts

cat > web/app/seller/dashboard/page.tsx << 'EOF'
'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { sellerAPI } from '@/lib/api/seller'

interface Account {
  id: string
  vendor: string
  status: string
  health_score: number
  authorized_credits_usd: number
  consumed_credits_usd: number
  expected_rate: number
  expire_at: string
}

export default function SellerDashboardPage() {
  const router = useRouter()
  const [accounts, setAccounts] = useState<Account[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!localStorage.getItem('seller_token')) {
      router.push('/seller/login')
      return
    }
    sellerAPI.getAccounts()
      .then((data: any) => setAccounts(Array.isArray(data) ? data : []))
      .catch(() => router.push('/seller/login'))
      .finally(() => setLoading(false))
  }, [router])

  const statusBadge = (status: string) => {
    const colors: Record<string, string> = {
      active: 'bg-green-100 text-green-800',
      pending_verify: 'bg-yellow-100 text-yellow-800',
      suspended: 'bg-red-100 text-red-800',
      revoked: 'bg-gray-100 text-gray-800',
      expired: 'bg-gray-100 text-gray-600',
    }
    return (
      <span className={`px-2 py-1 rounded text-xs font-medium ${colors[status] || 'bg-gray-100'}`}>
        {status}
      </span>
    )
  }

  if (loading) return <div className="min-h-screen flex items-center justify-center">加载中...</div>

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white shadow-sm px-6 py-4 flex justify-between items-center">
        <h1 className="text-xl font-bold text-gray-900">卖家控制台</h1>
        <div className="flex gap-4">
          <a href="/seller/earnings" className="text-blue-600 hover:underline">收益概览</a>
          <a href="/seller/settlements" className="text-blue-600 hover:underline">结算历史</a>
          <button onClick={() => { localStorage.removeItem('seller_token'); router.push('/seller/login') }}
            className="text-gray-500 hover:text-gray-700 text-sm">退出</button>
        </div>
      </nav>

      <main className="max-w-6xl mx-auto px-6 py-8">
        <div className="flex justify-between items-center mb-6">
          <h2 className="text-lg font-semibold text-gray-800">我的账号</h2>
          <a href="/seller/accounts/add"
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 text-sm">
            + 添加账号
          </a>
        </div>

        {accounts.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-8 text-center text-gray-500">
            还没有托管账号，
            <a href="/seller/accounts/add" className="text-blue-600 hover:underline ml-1">立即添加</a>
          </div>
        ) : (
          <div className="bg-white rounded-lg shadow overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  {['厂商', '状态', '健康度', '授权额度', '已消耗', '期望回收率', '到期时间', '操作'].map(h => (
                    <th key={h} className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {accounts.map((acc) => (
                  <tr key={acc.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-sm font-medium text-gray-900">{acc.vendor}</td>
                    <td className="px-4 py-3">{statusBadge(acc.status)}</td>
                    <td className="px-4 py-3 text-sm text-gray-600">
                      <span className={acc.health_score >= 60 ? 'text-green-600' : 'text-red-600'}>
                        {acc.health_score}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">${acc.authorized_credits_usd.toFixed(2)}</td>
                    <td className="px-4 py-3 text-sm text-gray-600">${acc.consumed_credits_usd.toFixed(2)}</td>
                    <td className="px-4 py-3 text-sm text-gray-600">{(acc.expected_rate * 100).toFixed(0)}%</td>
                    <td className="px-4 py-3 text-sm text-gray-600">{new Date(acc.expire_at).toLocaleDateString()}</td>
                    <td className="px-4 py-3 text-sm">
                      <a href={`/seller/accounts/${acc.id}`} className="text-blue-600 hover:underline">详情</a>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  )
}
EOF
```

### Step 2：添加账号页

```bash
mkdir -p web/app/seller/accounts/add

cat > web/app/seller/accounts/add/page.tsx << 'EOF'
'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { sellerAPI } from '@/lib/api/seller'

const VENDORS = ['anthropic', 'openai', 'gemini', 'qwen', 'glm', 'kimi']

export default function AddAccountPage() {
  const router = useRouter()
  const [form, setForm] = useState({
    vendor: 'anthropic',
    api_key: '',
    authorized_credits_usd: '',
    expected_rate: '0.75',
    expire_at: '',
    total_credits_usd: '',
  })
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    setSuccess('')
    try {
      const result = await sellerAPI.addAccount({
        vendor: form.vendor,
        api_key: form.api_key,
        authorized_credits_usd: parseFloat(form.authorized_credits_usd),
        expected_rate: parseFloat(form.expected_rate),
        expire_at: new Date(form.expire_at).toISOString(),
        total_credits_usd: form.total_credits_usd ? parseFloat(form.total_credits_usd) : undefined,
      }) as any
      setSuccess(`格式检查通过，账号 ID：${result.account_id}。建议发起测试请求验证 Key 实际有效性。`)
      setTimeout(() => router.push('/seller/dashboard'), 3000)
    } catch (err: any) {
      setError(err.message || '添加失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-lg mx-auto bg-white rounded-lg shadow p-8">
        <div className="flex items-center mb-6">
          <a href="/seller/dashboard" className="text-blue-600 hover:underline mr-4">← 返回</a>
          <h1 className="text-xl font-bold text-gray-900">添加托管账号</h1>
        </div>

        <div className="bg-yellow-50 border border-yellow-200 rounded p-3 mb-6 text-sm text-yellow-800">
          ⚠️ MVP 验证：当前仅做格式检查，不验证 Key 是否真实有效。添加后建议通过代理发送一次测试请求。
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">厂商</label>
            <select value={form.vendor} onChange={(e) => setForm({...form, vendor: e.target.value})}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500">
              {VENDORS.map(v => <option key={v} value={v}>{v}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">API Key</label>
            <input type="password" value={form.api_key} onChange={(e) => setForm({...form, api_key: e.target.value})}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="sk-ant-..." required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">授权额度（USD）</label>
            <input type="number" value={form.authorized_credits_usd} onChange={(e) => setForm({...form, authorized_credits_usd: e.target.value})}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              min="0.01" step="0.01" required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">期望回收率（0.50-0.95）</label>
            <input type="number" value={form.expected_rate} onChange={(e) => setForm({...form, expected_rate: e.target.value})}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              min="0.5" max="0.95" step="0.01" required />
            <p className="mt-1 text-xs text-gray-500">卖家实际收益 = 厂商实际成本 × 回收率。推荐 0.75。</p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">到期时间</label>
            <input type="datetime-local" value={form.expire_at} onChange={(e) => setForm({...form, expire_at: e.target.value})}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">账号总额度（USD，可选）</label>
            <input type="number" value={form.total_credits_usd} onChange={(e) => setForm({...form, total_credits_usd: e.target.value})}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              min="0" step="0.01" />
          </div>
          {error && <p className="text-red-500 text-sm">{error}</p>}
          {success && <p className="text-green-600 text-sm">{success}</p>}
          <button type="submit" disabled={loading}
            className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 disabled:opacity-50">
            {loading ? '提交中...' : '添加账号'}
          </button>
        </form>
      </div>
    </div>
  )
}
EOF
```

---

## Day 3 · 收益概览 + 结算历史页

### Step 1：收益概览页

```bash
mkdir -p web/app/seller/earnings

cat > web/app/seller/earnings/page.tsx << 'EOF'
'use client'

import { useEffect, useState } from 'react'
import { sellerAPI } from '@/lib/api/seller'

interface EarningsData {
  pending_usd: number
  total_earned_usd: number
  settlements: any[]
}

export default function EarningsPage() {
  const [data, setData] = useState<EarningsData | null>(null)
  const [requesting, setRequesting] = useState(false)
  const [msg, setMsg] = useState('')

  useEffect(() => {
    sellerAPI.getEarnings().then((d: any) => setData(d))
  }, [])

  const handleRequestSettlement = async () => {
    setRequesting(true)
    setMsg('')
    try {
      await sellerAPI.requestSettlement()
      setMsg('结算申请已提交，等待管理员处理')
      sellerAPI.getEarnings().then((d: any) => setData(d))
    } catch (err: any) {
      setMsg(err.message || '申请失败')
    } finally {
      setRequesting(false)
    }
  }

  if (!data) return <div className="min-h-screen flex items-center justify-center">加载中...</div>

  return (
    <div className="min-h-screen bg-gray-50 py-8">
      <div className="max-w-2xl mx-auto px-6">
        <div className="flex items-center mb-6">
          <a href="/seller/dashboard" className="text-blue-600 hover:underline mr-4">← 返回</a>
          <h1 className="text-xl font-bold text-gray-900">收益概览</h1>
        </div>

        <div className="grid grid-cols-2 gap-4 mb-6">
          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-500">待结算收益</p>
            <p className="text-3xl font-bold text-green-600 mt-1">${data.pending_usd.toFixed(2)}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-6">
            <p className="text-sm text-gray-500">累计已结算</p>
            <p className="text-3xl font-bold text-gray-900 mt-1">${data.total_earned_usd.toFixed(2)}</p>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow p-6 mb-4">
          <button onClick={handleRequestSettlement} disabled={requesting || data.pending_usd < 10}
            className="w-full bg-green-600 text-white py-2 px-4 rounded-md hover:bg-green-700 disabled:opacity-50">
            {requesting ? '申请中...' : `申请结算 $${data.pending_usd.toFixed(2)}`}
          </button>
          {data.pending_usd < 10 && <p className="text-xs text-gray-500 mt-2 text-center">最低结算金额为 $10</p>}
          {msg && <p className="text-sm text-center mt-2 text-blue-600">{msg}</p>}
        </div>

        {data.settlements && data.settlements.length > 0 && (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="font-semibold text-gray-800 mb-4">最近结算记录</h2>
            {data.settlements.map((s: any) => (
              <div key={s.id} className="flex justify-between py-2 border-b last:border-0">
                <div>
                  <p className="text-sm text-gray-600">${s.seller_earn_usd.toFixed(2)}</p>
                  <p className="text-xs text-gray-400">{new Date(s.created_at).toLocaleDateString()}</p>
                </div>
                <span className={`text-xs px-2 py-1 rounded ${s.status === 'paid' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}`}>
                  {s.status}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
EOF
```

---

## Day 4-5 · 买家前端启动

### Step 1：买家 API 封装

```bash
cat > web/lib/api/buyer.ts << 'EOF'
import { apiClient } from './client'

// 买家使用 buyer_token（JWT）访问业务接口
// 买家使用 api_key 访问代理接口（/v1/*）
export const buyerAPI = {
  register: (email: string, password: string) =>
    apiClient.post('/api/v1/buyer/auth/register', { email, password }),

  login: (phone: string, code: string) =>
    apiClient.post('/api/v1/buyer/auth/login', { phone, code }),

  getBalance: () =>
    apiClient.get('/api/v1/buyer/balance'),

  getUsage: (page = 1) =>
    apiClient.get(`/api/v1/buyer/usage?page=${page}`),

  submitTopup: (amountUSD: number, txHash: string, network: string) =>
    apiClient.post('/api/v1/buyer/topup', { amount_usd: amountUSD, tx_hash: txHash, network }),

  getTopupRecords: (page = 1) =>
    apiClient.get(`/api/v1/buyer/topup/records?page=${page}`),

  resetAPIKey: () =>
    apiClient.post('/api/v1/buyer/apikeys/reset'),
}
EOF
```

### Step 2：买家登录页

```bash
mkdir -p web/app/buyer/login

cat > web/app/buyer/login/page.tsx << 'EOF'
'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { buyerAPI } from '@/lib/api/buyer'

export default function BuyerLoginPage() {
  const router = useRouter()
  const [phone, setPhone] = useState('')
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const data = await buyerAPI.login(phone, code) as any
      localStorage.setItem('buyer_token', data.token)
      if (data.api_key) {
        localStorage.setItem('buyer_api_key', data.api_key)
      }
      router.push('/buyer/dashboard')
    } catch (err: any) {
      setError(err.message || '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full bg-white rounded-lg shadow p-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">买家登录</h1>
        <form onSubmit={handleLogin} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">手机号</label>
            <input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">验证码（MVP 固定 123456）</label>
            <input type="text" value={code} onChange={(e) => setCode(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
              required />
          </div>
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button type="submit" disabled={loading}
            className="w-full bg-indigo-600 text-white py-2 px-4 rounded-md hover:bg-indigo-700 disabled:opacity-50">
            {loading ? '登录中...' : '登录'}
          </button>
          <p className="text-center text-sm text-gray-500">
            没有账号？<a href="/buyer/register" className="text-indigo-600 hover:underline ml-1">注册</a>
          </p>
        </form>
      </div>
    </div>
  )
}
EOF
```

### Step 3：买家注册页

```bash
mkdir -p web/app/buyer/register

cat > web/app/buyer/register/page.tsx << 'EOF'
'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { buyerAPI } from '@/lib/api/buyer'

export default function BuyerRegisterPage() {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [apiKey, setApiKey] = useState('')

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const data = await buyerAPI.register(email, password) as any
      localStorage.setItem('buyer_token', data.token)
      localStorage.setItem('buyer_api_key', data.api_key)
      setApiKey(data.api_key) // 展示一次，提示保存
    } catch (err: any) {
      setError(err.message || '注册失败')
    } finally {
      setLoading(false)
    }
  }

  if (apiKey) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full bg-white rounded-lg shadow p-8">
          <h1 className="text-xl font-bold text-green-600 mb-4">注册成功！</h1>
          <div className="bg-yellow-50 border border-yellow-300 rounded p-4 mb-4">
            <p className="text-sm font-medium text-yellow-800 mb-2">请立即保存您的 API Key（仅显示一次）：</p>
            <code className="block text-xs font-mono bg-white border rounded p-2 break-all">{apiKey}</code>
          </div>
          <p className="text-sm text-gray-500 mb-4">使用此 Key 调用 <code>/v1/chat/completions</code> 代理接口。</p>
          <button onClick={() => router.push('/buyer/dashboard')}
            className="w-full bg-indigo-600 text-white py-2 px-4 rounded-md hover:bg-indigo-700">
            已保存，进入控制台
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full bg-white rounded-lg shadow p-8">
        <h1 className="text-2xl font-bold text-gray-900 mb-6">买家注册</h1>
        <form onSubmit={handleRegister} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">邮箱</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">密码</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-indigo-500"
              required />
          </div>
          {error && <p className="text-red-500 text-sm">{error}</p>}
          <button type="submit" disabled={loading}
            className="w-full bg-indigo-600 text-white py-2 px-4 rounded-md hover:bg-indigo-700 disabled:opacity-50">
            {loading ? '注册中...' : '注册'}
          </button>
          <p className="text-center text-sm text-gray-500">
            已有账号？<a href="/buyer/login" className="text-indigo-600 hover:underline ml-1">登录</a>
          </p>
        </form>
      </div>
    </div>
  )
}
EOF
```

**✅ Week 6 验收脚本**

```bash
cat > scripts/week6_verify.sh << 'EOF'
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

check "卖家控制台主页" "curl -s http://localhost:3000/seller/dashboard" '卖家控制台\|登录'
check "卖家添加账号页" "curl -s http://localhost:3000/seller/accounts/add" '添加托管账号\|登录'
check "卖家收益概览页" "curl -s http://localhost:3000/seller/earnings" '收益概览\|登录'
check "买家登录页" "curl -s http://localhost:3000/buyer/login" '买家登录'
check "买家注册页" "curl -s http://localhost:3000/buyer/register" '买家注册'
check "前端构建成功" "cd web && npm run build 2>&1 | tail -5" 'Route\|compiled'

echo -e "\n通过 $PASS，失败 $FAIL"
[ $FAIL -eq 0 ] && echo -e "${GREEN}🎉 Week 6 全部验收通过！${NC}" || exit 1
EOF
chmod +x scripts/week6_verify.sh
bash scripts/week6_verify.sh
```

**✅ Week 6 完成标准**
- [ ] 卖家控制台（账号列表、添加账号、收益概览页）完成
- [ ] 买家注册页展示 API Key 提示（仅显示一次）
- [ ] 买家登录页正常跳转
- [ ] 前端 `npm run build` 无错误

## 下周预告（Week 7）

Week 7 实现：
- 买家前端完整（余额、充值、用量明细、API Key 管理）
- 管理后台（充值审核、结算审核页）

前置条件：Week 6 卖家前端通过构建
