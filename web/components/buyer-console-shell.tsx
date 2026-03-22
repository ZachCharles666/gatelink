"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { buyerStorage } from "@/lib/api/buyer";

type BuyerConsoleShellProps = {
  title: string;
  subtitle: string;
  children: ReactNode;
  actions?: ReactNode;
};

const NAV_ITEMS = [
  { href: "/buyer/dashboard", label: "买家控制台" },
  { href: "/buyer/topup", label: "充值" },
  { href: "/buyer/usage", label: "用量明细" },
  { href: "/buyer/apikey", label: "API Key" },
];

export function BuyerConsoleShell({
  title,
  subtitle,
  children,
  actions,
}: BuyerConsoleShellProps) {
  const pathname = usePathname();
  const router = useRouter();

  function handleLogout() {
    buyerStorage.clearToken();
    buyerStorage.clearAPIKey();
    router.push("/buyer/login");
  }

  return (
    <main className="console-page">
      <div className="console-shell">
        <header className="console-topbar">
          <div className="console-brand">
            <span className="console-brand-kicker">GateLink Buyer</span>
            <strong className="console-brand-title">买家控制台</strong>
          </div>
          <div className="console-topbar-right">
            <nav className="console-nav" aria-label="Buyer navigation">
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
