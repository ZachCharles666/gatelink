# Buyer Spec v0.3

## 1. 买家端目标

让买家通过钱包登录、充值 Credits、获取平台 API Key，并在调用平台 API 时按实际 usage 消耗 Credits。

## 2. 主要页面

- `Wallet Sign-In`
- `Buyer Dashboard`
- `Credits`
- `API Keys`
- `Usage & Billing`
- `Refunds`
- `Settings`

## 3. 核心功能

### 3.1 钱包注册/登录

- 支持 MetaMask、Rabby、WalletConnect 等 EVM 钱包
- 使用 challenge 签名完成注册/登录
- 登录后展示绑定地址、网络状态、最近登录时间

### 3.2 Credits 充值

- 买家发起充值
- 平台生成充值意图
- 买家在 Hyperliquid 完成转账
- 平台监听到账并发放 Credits

### 3.3 API Key 管理

- 创建 API Key
- 查看掩码
- 重置或停用 API Key
- 仅发放 GateLink API Key，不暴露卖家原始 Key

### 3.4 Usage 与账单

- 按请求展示模型、卖家、时间、消耗 Credits、状态
- 流式调用在流结束后统一扣费
- 余额不足时阻止新请求

### 3.5 退款

- 支持未使用 Credits 退款
- 退款默认返回已绑定钱包
- 支持最小退款门槛与大额人工审核阈值

## 4. 页面级信息规范

### Dashboard

- 总 Credits
- 近 7 天消耗
- 活跃 API Keys
- 最近失败请求

### Credits

- 当前余额
- 充值入口
- 充值历史
- 退款入口

### API Keys

- Key 名称
- 创建时间
- 最近使用时间
- 状态

### Usage & Billing

- 时间
- 模型
- 卖家
- 扣费 Credits
- 请求状态

## 5. 关键规则

- 买家拿到的是平台访问权，不是卖家原始 Key
- Credits 是平台消费额度，不等于链上即时余额
- 退款只针对未消费 Credits
