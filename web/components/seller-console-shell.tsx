"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { authStorage } from "@/lib/api/client";

type SellerConsoleShellProps = {
  title: string;
  subtitle: string;
  children: ReactNode;
  actions?: ReactNode;
};

const NAV_ITEMS = [
  { href: "/seller/dashboard", label: "账号控制台" },
  { href: "/seller/accounts/add", label: "添加账号" },
  { href: "/seller/earnings", label: "收益概览" },
  { href: "/seller/settlements", label: "结算历史" },
];

export function SellerConsoleShell({
  title,
  subtitle,
  children,
  actions,
}: SellerConsoleShellProps) {
  const pathname = usePathname();
  const router = useRouter();

  function handleLogout() {
    authStorage.clearToken();
    router.push("/seller/login");
  }

  return (
    <main className="console-page">
      <div className="console-shell">
        <header className="console-topbar">
          <div className="console-brand">
            <span className="console-brand-kicker">GateLink Seller</span>
            <strong className="console-brand-title">卖家控制台</strong>
          </div>
          <div className="console-topbar-right">
            <nav className="console-nav" aria-label="Seller navigation">
              {NAV_ITEMS.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`console-link${pathname === item.href ? " console-link--active" : ""}`}
                >
                  {item.label}
                </Link>
              ))}
            </nav>
            <button className="console-logout" type="button" onClick={handleLogout}>
              退出
            </button>
          </div>
        </header>

        <section className="console-hero">
          <div>
            <h1 className="console-title">{title}</h1>
            <p className="console-subtitle">{subtitle}</p>
          </div>
          {actions ? <div className="console-hero-actions">{actions}</div> : null}
        </section>

        {children}
      </div>
    </main>
  );
}
