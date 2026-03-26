# ADR-003 Hyperliquid Fund Flow

## Status

Accepted

## Decision

链上仅处理充值、退款、卖家结算；高频 usage 与佣金拆账在平台账本完成。

## Rationale

- 减少链上交互频次
- 保持 API 调用体验稳定
- 更适合流式请求结算
