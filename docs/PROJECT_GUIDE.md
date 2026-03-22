# AI Credits 授权调度平台 — 项目核心指南

> 基于 PRD V0.2 + Contract_DevA_DevB + DevA_Engine_Spec + DevB_Business_Spec 整合
> 本文件是项目的权威理解文档，优先级高于各子文档的模糊描述

---

## 一、产品本质

**这是一个 AI API Credits 二级市场调度平台。**

- **卖家**：持有 Anthropic/OpenAI 等平台的 Credits，到期用不完会贬值。通过托管给平台，以约75%折扣提前变现（USDT结算）
- **买家**：以约88折购买 AI API 调用能力，只需改 `base_url` 和 `api_key`，代码零改动
- **平台**：居中撮合，赚约13%毛利，负责调度、安全、对账

### 关键数字

| 指标 | 值 |
|------|----|
| 平台对买家售价 | 官方定价 × 88% |
| 卖家回收价 | 官方定价 × 75%（expected_rate 由卖家自定） |
| 平台毛利 | ~13%（88%-75%） |
| MVP 工期 | 8周，2人开发 |
| 结算方式 | MVP 阶段纯记账，线下 USDT 手动结算 |

---

## 二、系统架构

### 分层架构

```
L0  接入网关      Nginx + Go HTTP Server   TLS终止、限速、IP黑名单
L1  业务API层     Go Gin（Dev-B）          卖家/买家/管理 API、鉴权
L2  核心引擎层    Go goroutine（Dev-A）    调度、健康评分、内容审核、代理转发
L3  数据层        PostgreSQL + Redis       业务持久化、实时状态、缓存
L4  外部集成层    Go HTTP Client           厂商API对接、Console同步
```

### 服务划分（Docker Compose）

| 服务 | 负责人 | 端口 | 说明 |
|------|--------|------|------|
| engine | Dev-A | 8081（内部） | 核心引擎，仅内网暴露 |
| api | Dev-B | 8080（对外） | 业务API + 买家代理端点 |
| web | Dev-B | 3000（对外） | Next.js 前端 |
| postgres | 共同 | 5432（内部） | 数据库 |
| redis | 共同 | 6379（内部） | 缓存/队列/事件 |
| nginx | 共同 | 80/443（对外） | 反向代理统一入口 |

### 买家请求完整链路

```
买家请求
  → Nginx（限速、黑名单）
  → Dev-B api:8080（鉴权、余额预检）
  → Dev-A POST /internal/v1/audit（内容审核）
  → Dev-A POST /internal/v1/dispatch（调度+转发）
    → 调度引擎选账号 → 解密Key → 向厂商发请求
    → 厂商返回 → 记录 usage_records
  → Dev-B 记账（扣买家余额、增卖家收益）事务
  → 响应返回买家
```

---

## 三、核心数据模型

### 主要表及所有权

| 表名 | 主写方 | 说明 |
|------|--------|------|
| accounts | Dev-A | 卖家托管账号，Key 加密存储 |
| sellers | Dev-B | 卖家基本信息、收款地址、收益 |
| buyers | Dev-B | 买家信息、平台API Key、余额 |
| usage_records | Dev-A | 核心消耗账本，每次转发后写入 |
| settlements | Dev-B | 结算记录 |
| health_events | Dev-A | 账号健康事件日志 |
| vendor_pricing | 共同 | 厂商定价配置 |

### 关键字段说明

**accounts 表**
- `api_key_encrypted`：AES-256-GCM 加密，永不明文返回前端
- `authorized_credits_usd`：卖家授权给平台调度的额度（非全部credits）
- `health_score`：0-100 动态评分，调度权重依据
- `expected_rate`：卖家期望回收折扣率（0.5~0.95）

**usage_records 表（核心账本）**
- `cost_usd`：官方定价计算的实际成本
- `buyer_charged_usd`：买家实际扣费（平台售价）
- `seller_earn_usd`：卖家应得收益
- `platform_earn_usd`：平台毛利

---

## 四、Dev-A / Dev-B 职责边界

### 职责总览

| 维度 | Dev-A | Dev-B |
|------|-------|-------|
| 核心职责 | 代理转发、调度引擎、健康评分、安全执行 | 业务API、前端控制台、厂商适配、记账结算 |
| DB | 主导Schema，主写 accounts/usage_records/health_events | 消费Schema，主写 sellers/buyers/settlements |
| 安全 | API Key 加密存储、代理执行隔离 | 买家鉴权、输入验证、审核调用 |
| 外部依赖 | 厂商API（直接调用） | 通过 Dev-A 内部接口间接访问 |

### DB 变更流程（契约锁定）

| 场景 | 流程 | 耗时上限 |
|------|------|--------|
| 新增字段 | 提出方写migration → 另一方review → 合并 | 24h |
| 修改字段类型/约束 | 提出方写migration + 影响分析 → 讨论确认 | 48h |
| 删除字段 | 必须面对面确认，评估所有引用点 | 面对面 |

---

## 五、内部接口契约

> 所有接口地址：`http://engine:8081/internal/v1/`，仅 Docker 内网暴露，无 JWT 鉴权

| 接口 | 交付周 | 优先级 | 用途 |
|------|--------|--------|------|
| POST /dispatch | Week 2 | P0 | 核心：调度+代理转发，返回 usage |
| GET /pool/status | Week 2 | P0 | Dashboard 展示账号池概况 |
| GET /accounts/:id/health | Week 2 | P0 | 卖家详情页健康度展示 |
| POST /accounts/:id/verify | Week 2 | P0 | 卖家添加账号时验证 Key |
| POST /audit | Week 3 | P1 | 内容审核，dispatch 前调用 |
| GET /accounts/:id/console-usage | Week 4 | P1 | Console 用量同步结果 |
| GET /accounts/:id/diff | Week 4 | P1 | 平台记录 vs Console 差异 |

### dispatch 接口详情

```
POST /internal/v1/dispatch
入参: { buyer_id, vendor, model, payload, stream }
出参: { account_id, usage: {input_tokens, output_tokens, cost_usd}, response }
错误码:
  4001 无可用账号
  4002 余额不足（Dev-B 先校验，Dev-A 二次校验）
  4003 内容审核未通过
  5001 厂商侧错误
```

### Dev-A → Dev-B 事件通知（Redis Pub/Sub）

| 事件 | 频道 | Dev-B 响应 |
|------|------|-----------|
| account.suspended | engine:events:account | 更新账号展示状态为「异常」，可选通知卖家 |
| account.low_balance | engine:events:account | 提示卖家额度不足 |
| account.verified | engine:events:account | 更新账号 status: pending_verify → active |
| usage.synced | engine:events:usage | 刷新对账数据缓存 |

---

## 六、调度引擎核心算法

### 评分公式

```
score = 0.35 × urgency + 0.25 × price + 0.30 × health + 0.10 × capacity

urgency  = 1 - days_to_expire / 120      // 越临期越优先
price    = 1 - expected_rate / 1.0       // 报价越低越优先
health   = health_score / 100            // 健康度越高越优先
capacity = remaining_usd / authorized_usd // 剩余越多越优先
```

### 硬性排除条件

- health_score < 30
- remaining_usd < $1
- 当前分钟 RPM >= 厂商限制 × 80%
- 当日消耗 >= 历史峰值 × 150%
- 同一买家连续3次路由到同一账号（强制切换，防关联检测）

### 账号健康度关键事件

| 事件 | 分值变化 |
|------|--------|
| 请求成功 | +0.5 |
| 4xx 错误 | -5 |
| 5xx 错误 | -2 |
| 429 限速 | -15 |
| 连续失败3次 | -20 |
| 401/403 封号 | 直接归零下线 |
| 对账通过（误差<1%） | +10 |
| 对账异常（误差>5%） | -30 |
| 24小时稳定 | +5 |

### 健康度阈值策略

| 分值 | 状态 | 调度策略 |
|------|------|--------|
| 80-100 | healthy | 正常调度 |
| 60-79 | degraded | 调度频率降至 50% |
| 30-59 | warning | 仅备用 |
| 1-29 | critical | 停止调度，发事件通知卖家 |
| 0 | offline | 下线，触发结算清算 |

---

## 七、安全要求

### API Key 管理

| 场景 | 处理方式 |
|------|--------|
| 存储 | AES-256-GCM 加密，密钥存环境变量 |
| 使用 | 运行时解密，仅存在于 goroutine 栈，不写日志 |
| 展示 | 永远只展示掩码版本（sk-ant-xxx...xxxx） |
| 撤回 | status=revoked，调度引擎立即停止使用 |
| 泄露 | 一键紧急撤回，平台停止调度并通知卖家去厂商轮换 |

### 防滥用

| 场景 | 阈值 |
|------|------|
| 买家暴力破解 | 5次失败锁定15分钟 |
| Standard 买家限速 | 60次/分钟 |
| Pro 买家限速 | 300次/分钟 |
| 充值套现冷却 | 充值后48小时内不可结算 |
| 单账号调用突增 | 超历史峰值150%触发人工审核 |

---

## 八、厂商适配

| 厂商 | API格式 | 转换复杂度 | MVP优先级 |
|------|--------|-----------|---------|
| Anthropic | 原生格式 | 中（system角色处理）| P0 |
| OpenAI | OpenAI格式 | 低（直接透传）| P0 |
| Qwen | OpenAI兼容 | 低 | P0 |
| GLM | OpenAI兼容 | 低 | P1 |
| Kimi | OpenAI兼容 | 低 | P1 |
| Google Gemini | Gemini格式 | 高（格式差异大）| P1 |

---

## 九、定价配置

| 厂商 | 模型 | 官方输入价/1M | 官方输出价/1M | 平台折扣 | 平台售价（输入）|
|------|------|-------------|-------------|--------|--------------|
| Anthropic | claude-sonnet-4-6 | $3.00 | $15.00 | 88% | $2.64 |
| Anthropic | claude-haiku-4-5 | $0.80 | $4.00 | 88% | $0.70 |
| OpenAI | gpt-4o | $2.50 | $10.00 | 88% | $2.20 |
| OpenAI | gpt-4o-mini | $0.15 | $0.60 | 88% | $0.13 |
| Gemini | gemini-2.5-pro | $1.25 | $10.00 | 88% | $1.10 |

---

## 十、MVP 验收标准（11项全部必过）

| 验收项 | 标准 |
|--------|------|
| 代理转发 | Python/Node/curl 三种方式均可成功调用 Claude 和 GPT |
| 流式响应 | SSE 流式输出与官方原生行为一致 |
| 卖家授权 | 添加账号→设置额度→撤回授权全流程无错误 |
| 调度逻辑 | 3个不同健康度账号，验证优先路由高健康度 |
| 临期优先 | 30天 vs 90天到期账号，验证临期账号优先被消耗 |
| 余额扣减 | 误差 < 0.01% |
| Key 安全 | DB 密文存储，前端接口永不返回明文 |
| 内容审核 | 违禁词请求返回400，不进入调度层 |
| 账号下线 | 模拟401，验证健康度归零并停止调度 |
| 并发稳定 | 100并发成功率 > 99%，P99延迟增加 < 200ms |
| 记账准确 | 100笔：卖家收益+平台毛利=买家扣费，分毫不差 |

---

## 十一、8周开发时间线

| 周次 | Dev-A 里程碑 | Dev-B 里程碑 | 联调节点 |
|------|------------|------------|--------|
| Week 1-2 | 基础设施 + 数据模型 + 代理转发核心 | 鉴权系统 + 厂商适配层框架 | Week 1: 接口契约对齐；Week 2: 卖家添加账号端到端 |
| Week 3-4 | 调度引擎 + API Key 管理 | 卖家业务API + 记账系统 + 代理端点 | Week 3: 买家请求端到端；Week 4: 流式响应 |
| Week 5-6 | 健康度系统 + 内容审核 + Console同步 | 买家业务API + 卖家前端 + 结算系统 | Week 5: 健康度联调；Week 6: Console对账联调 |
| Week 7 | 用量同步 + 监控告警 + 压测优化 | 买家前端 + 充值流程 + 管理后台 | 全链路压测 |
| Week 8 | 性能优化 + 稳定性 | 接入文档 + 体验打磨 | MVP 验收 |

---

## 十二、后续迭代计划

| 阶段 | 触发条件 | 新增功能 |
|------|--------|--------|
| V1.1 | 月流水 > $2,000 | 自动充值提醒、邮件通知、用量预警 |
| V1.2 | 月流水 > $5,000 | Paddle（海外）+ 支付宝/微信（国内）支付接入 |
| V1.3 | 卖家 > 50人 | 批量账号导入、账号组管理、调度策略自定义 |
| V2.0 | 月流水 > $20,000 | Solana 智能合约资金托管 + 链上消耗哈希 |
| V2.1 | 有企业客户 | 私有化部署、企业账单、发票支持 |
| V3.0 | 市场验证后 | Token 化 |
