import type { ReactNode } from "react";

type BuyerAuthShellProps = {
  title: string;
  subtitle: string;
  kicker: string;
  children: ReactNode;
};

export function BuyerAuthShell({
  title,
  subtitle,
  kicker,
  children,
}: BuyerAuthShellProps) {
  return (
    <main className="auth-page">
      <section className="auth-side">
        <div
          className="auth-story"
          style={{
            background:
              "linear-gradient(145deg, rgba(61, 56, 142, 0.96), rgba(93, 82, 194, 0.88)), linear-gradient(180deg, rgba(239, 143, 63, 0.2), transparent)",
          }}
        >
          <div>
            <span className="auth-kicker">{kicker}</span>
            <h1 className="auth-headline">GateLink Buyer Console</h1>
            <p className="auth-copy">
              用统一的 buyer token 和 buyer api_key 进入 GateLink 的代理入口。Week 6 先把认证入口搭起来，
              Week 7 再继续补余额、充值、用量和 API Key 管理页。
            </p>
          </div>
          <div className="auth-grid">
            <div className="auth-grid-card">
              <strong>注册即发 API Key</strong>
              注册成功后会返回一把新的 buyer api_key，页面只展示这一次。
            </div>
            <div className="auth-grid-card">
              <strong>JWT + Proxy 双轨</strong>
              业务页走 buyer token，模型代理走 buyer api_key，两者都会在浏览器保存。
            </div>
            <div className="auth-grid-card">
              <strong>Week 7 接力</strong>
              下一阶段会继续把余额、充值申请、用量列表和 Key 重置页补齐。
            </div>
          </div>
        </div>
      </section>
      <section className="auth-panel-wrap">
        <div className="auth-panel">
          <h2 className="auth-panel-title">{title}</h2>
          <p className="auth-panel-subtitle">{subtitle}</p>
          {children}
        </div>
      </section>
    </main>
  );
}
