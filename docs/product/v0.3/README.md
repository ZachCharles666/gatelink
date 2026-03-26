# GateLink Product Docs v0.3

这是 GateLink 当前正式产品方案入口。

适用范围：
- 买家端：钱包登录、Credits 充值与 API Key 消耗
- 卖家端：钱包登录、资源管理、收益与结算
- 管理端：充值监控、退款、结算、风控、对账
- 平台后台：Hyperliquid 资金流、账本、状态机、对账与幂等

阅读顺序：
1. `PRD.md`
2. `BUYER.md`
3. `SELLER.md`
4. `ADMIN.md`
5. `PLATFORM_BACKEND.md`
6. `API_SPEC.md`
7. `DATA_MODEL.md`
8. `STATE_MACHINES.md`
9. `UI_GUIDELINES.md`

版本原则：
- 当前版本：`v0.3`
- 历史版本与旧协作文档：`archive/`
- `archive/` 中的内容不再作为现行方案入口

核心规则：
- 买家采用充值 Credits 制度，`1 USDC = 1 Credit`
- v0.3 仅做钱包注册/登录，不做邮箱与手机号登录
- 买家使用 GateLink API Key，不直接获得卖家原始 Key
- 链上只处理充值、退款、卖家结算
- 高频 usage、佣金拆账、卖家待结算累加在平台账本完成
- `engine/` 是现行核心代码目录，保留在当前仓库主结构中
