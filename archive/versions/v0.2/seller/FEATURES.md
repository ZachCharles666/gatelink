# GateLink v0.2 · 卖家功能文档

## 1. 文档定位

本文件描述 v0.2 版本的卖家端功能逻辑、信息架构与关键页面职责。

当前设计依据：

- [web/app/seller/dashboard/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/dashboard/page.tsx)
- [web/app/seller/accounts/add/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/accounts/add/page.tsx)
- [web/app/seller/earnings/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/earnings/page.tsx)
- [web/app/seller/settlements/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/settlements/page.tsx)
- [web/components/seller-console-shell.tsx](/Users/tvwoo/Projects/gatelink/web/components/seller-console-shell.tsx)
- [web/lib/api/seller.ts](/Users/tvwoo/Projects/gatelink/web/lib/api/seller.ts)
- [api/internal/seller/handler.go](/Users/tvwoo/Projects/gatelink/api/internal/seller/handler.go)

## 2. 当前卖家端定位

卖家端当前承担的是“资源供给侧工作台”，核心目标不是消费模型，而是：

- 托管可供平台调度的账号
- 配置授权额度与回收率
- 观察账号状态与使用情况
- 查看收益
- 发起结算并跟踪结算状态

也就是说，卖家后台本质上更接近：

- 供应端控制台
- 运营结算面板
- 资源管理后台

## 3. 卖家端核心目标

### 3.1 认证与进入后台

- 注册
- 登录
- 进入卖家控制台

### 3.2 账号托管与管理

- 创建托管账号
- 查看账号状态
- 管理授权额度
- 查看账号使用情况

### 3.3 收益与结算

- 查看待结算金额
- 查看累计收益
- 发起结算申请
- 跟踪历史结算记录

### 3.4 后续商品化扩展

- 把托管资源逐步演进成更可售卖、更可运营的资源单元
- 为后续“广场 / 买家购买”提供上游供给基础

## 4. 当前卖家后台信息架构

### 布局

当前已实现的卖家后台是：

- 顶部品牌区
- 顶部横向导航
- 退出按钮
- 每个页面自己的 hero 区和主内容区

当前导航来自：

- [web/components/seller-console-shell.tsx](/Users/tvwoo/Projects/gatelink/web/components/seller-console-shell.tsx)

现有导航项：

1. `账号控制台`
2. `添加账号`
3. `收益概览`
4. `结算历史`

### 后续结构建议

如果后续继续升级为标准 SaaS 控制台，可以逐步演进成：

- 左侧导航
- 顶部卖家资料区
- 右上角个人中心 / 结算状态 / 通知

但在 v0.2 阶段，当前横向导航已经足够承载主要业务流程。

## 5. 页面职责

## 5.1 账号控制台

目标：

- 汇总当前卖家名下的托管账号
- 让卖家快速看到账号数量、活跃状态、授权额度与到期情况

当前页面来源：

- [web/app/seller/dashboard/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/dashboard/page.tsx)

当前主要内容：

- 账号总数
- Active 账号数
- 待验证账号数
- 总授权额度 / 已消耗额度
- 托管账号列表

当前列表字段：

- 厂商
- 账号 ID
- 状态
- 健康度
- 授权额度 / 已消耗
- 期望回收率
- 到期时间

页面职责建议：

- 作为卖家后台首页
- 承担“总览 + 快速跳转”的作用
- 后续可增加：
  - 账号搜索
  - 状态筛选
  - 到期提醒
  - 高风险账号提醒

## 5.2 添加账号

目标：

- 把新的上游账号接入平台托管链路

当前页面来源：

- [web/app/seller/accounts/add/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/accounts/add/page.tsx)

当前提交流程：

1. 卖家提交：
   - `vendor`
   - `api_key`
   - `authorized_credits_usd`
   - `expected_rate`
   - `expire_at`
   - `total_credits_usd`
2. 前端调用：
   - `POST /api/v1/seller/accounts`
3. 后端再调用 engine 的：
   - `POST /internal/v1/accounts`
4. 成功返回：
   - `account_id`
   - `api_key_hint`
   - `status`

当前页面语义已经对齐：

- 创建成功 = engine 已完成加密存储、写库并入池
- 创建成功 != 厂商侧 key 已完成真实有效性验证

页面职责建议：

- 清晰区分“创建成功”和“验证成功”
- 给卖家明确的风险预期
- 后续可增加：
  - 厂商说明
  - 建议额度
  - 到期策略提示
  - 测试请求入口

## 5.3 收益概览

目标：

- 让卖家快速理解当前可结算收益与历史收益状态

当前页面来源：

- [web/app/seller/earnings/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/earnings/page.tsx)

当前主要内容：

- 待结算收益
- 累计已结算
- 最近结算数
- 当前是否达到最低结算门槛
- 最近结算快照

当前页面还承载一个关键动作：

- 发起结算申请

建议保留的核心语义：

- 这里是“收益决策页”
- 不只是看数字，还要能直接发起结算

后续增强建议：

- 收益趋势图
- 账号维度收益分布
- 厂商维度收益分布
- 本周 / 本月收益对比

## 5.4 结算历史

目标：

- 让卖家追踪所有结算单与付款状态

当前页面来源：

- [web/app/seller/settlements/page.tsx](/Users/tvwoo/Projects/gatelink/web/app/seller/settlements/page.tsx)

当前展示字段：

- 结算单 ID
- 卖家 ID
- 金额
- 周期
- 创建时间
- 状态
- 付款时间 / 交易哈希

页面职责建议：

- 作为卖家财务追踪页
- 保证记录完整、易于查找和筛选
- 后续可增加：
  - 状态筛选
  - 时间筛选
  - 导出
  - 详情页

## 6. 当前关键数据与状态语义

从前端 API 形状看，当前卖家端最重要的实体是：

- `SellerAccount`
- `SellerSettlement`
- `SellerEarningsResponse`

其中 `SellerAccount` 关键字段包括：

- `vendor`
- `status`
- `health_score`
- `expected_rate`
- `authorized_credits_usd`
- `consumed_credits_usd`
- `total_credits_usd`
- `expire_at`

这说明卖家后台在 v0.2 的核心不是“商品运营”，而是“账号托管运营”。

## 7. 关键产品边界

卖家端当前提供的是：

- 账号托管能力
- 授权额度管理能力
- 收益与结算能力

卖家端当前还没有正式提供的是：

- 商品发布页
- 对外售卖配置页
- 面向买家的资源描述配置页

因此如果后续要支持“买家广场”，卖家端还需要往前补一层：

- 资源商品化能力
- 发布与上架能力
- 面向买家的展示字段管理

## 8. 推荐落地顺序

### Phase A

先稳定当前卖家控制台：

1. 账号控制台体验收口
2. 添加账号流程文案与校验收口
3. 收益与结算页信息增强

### Phase B

再补供给侧商品化能力：

1. 资源发布
2. 售卖配置
3. 上架 / 下架
4. 面向买家的展示信息维护

### Phase C

最后补更强的运营能力：

1. 风险提醒
2. 到期预警
3. 账号健康趋势
4. 卖家经营分析
