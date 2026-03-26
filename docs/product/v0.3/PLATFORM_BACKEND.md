# Platform Backend Spec v0.3

## 1. 后台职责

平台后台负责：
- 钱包 challenge 与验签
- Hyperliquid 充值监听
- Credits 账本
- usage 拆账
- 退款处理
- 卖家结算
- 风控与对账

## 2. 资金账户建议

- `Platform Main Wallet`：主资金账户，持有平台在 HL 的资金
- `Agent Wallet`：自动化签名执行结算/退款
- `Cold Wallet`：冷备资金钱包

## 3. 核心服务模块

### 3.1 Auth Service

- 生成钱包登录 challenge
- 验证签名
- 绑定用户与钱包

### 3.2 Deposit Worker

- 监听 HL 用户事件
- 识别平台主钱包的到账
- 通过幂等键去重
- 将确认到账映射到充值意图

### 3.3 Credits Ledger

- 记录买家 Credits 增减
- 记录退款冻结与释放
- 为 usage 扣减提供余额校验

### 3.4 Usage Accounting

- 请求完成后执行扣费
- 更新买家 Credits
- 增加卖家待结算
- 增加平台佣金

### 3.5 Settlement Service

- 聚合卖家待结算
- 发起 HL 结算转账
- 更新结算状态

### 3.6 Refund Service

- 校验可退款 Credits
- 发起退款转账
- 处理失败重试

### 3.7 Reconciliation

- 对比链上余额和平台总负债
- 输出差异
- 提供异常处理入口

## 4. 设计原则

- 大额入金/出金上链，高频 usage 留在平台账本
- 所有链上事件必须幂等
- WebSocket 监听与补偿查询并存
- `engine/` 保持现行核心代码目录，不纳入归档
