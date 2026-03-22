import type { ReactNode } from "react";

type SellerAuthShellProps = {
  title: string;
  subtitle: string;
  kicker: string;
  children: ReactNode;
};

export function SellerAuthShell({
  title,
  subtitle,
  kicker,
  children,
}: SellerAuthShellProps) {
  return (
    <main className="auth-page">
      <section className="auth-side">
        <div className="auth-story">
          <div>
            <span className="auth-kicker">{kicker}</span>
            <h1 className="auth-headline">GateLink Seller Console</h1>
            <p className="auth-copy">
              把快到期或暂时用不完的模型 credits 授权出来，平台帮你接住需求波峰。
              Week 5 先把卖家入口搭起来，下一周再把账号、收益和结算页面接上。
            </p>
          </div>
          <div className="auth-grid">
            <div className="auth-grid-card">
              <strong>手机号 + 验证码</strong>
              MVP 阶段先用固定验证码 123456，方便联调。
            </div>
            <div className="auth-grid-card">
              <strong>JWT 已接通</strong>
              登录成功后会把 seller token 写入浏览器，后续页面可直接复用。
            </div>
            <div className="auth-grid-card">
              <strong>Week 6 接力</strong>
              账号列表、收益概览和结算历史将在下一阶段继续补全。
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
