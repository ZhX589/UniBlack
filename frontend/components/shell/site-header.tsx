'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { useAuth, useSite } from '@/app/providers'
import { isNavActive, visibleNavigation } from '@/lib/navigation'
import { useScrollDirection } from '@/lib/use-scroll-direction'

function NavLink({ href, children, exact }: { href: string; children: React.ReactNode; exact?: boolean }) {
  const pathname = usePathname()
  const active = isNavActive(pathname, { href, label: '', area: 'public', exact })
  return (
    <Link
      href={href}
      aria-current={active ? 'page' : undefined}
      className={active ? 'font-medium text-text-inverse' : 'text-text-inverse-muted hover:text-text-inverse'}
    >
      {children}
    </Link>
  )
}

export function SiteHeader() {
  const { status, user, logout, hasRole } = useAuth()
  const { name, registrationEnabled } = useSite()
  const router = useRouter()
  const dir = useScrollDirection()

  const links = visibleNavigation({
    authenticated: status === 'authenticated',
    roles: user?.roles || [],
    registrationEnabled,
    area: ['public', 'account'],
  }).filter((item) => item.href !== '/login' && item.href !== '/register')

  const showAdmin = hasRole('admin', 'moderator')

  function signOut() {
    logout()
    router.push('/')
  }

  return (
    <header
      className={`glass shell-slide sticky top-0 z-50 border-b border-white/10 bg-gray-900/85 text-text-inverse ${
        dir === 'down' ? '-translate-y-full' : 'translate-y-0'
      }`}
    >
      <nav className="container mx-auto flex flex-wrap items-center justify-between gap-4 px-4 py-3">
        <Link href="/" className="text-xl font-bold font-display">
          {name}
        </Link>
        <div className="flex flex-wrap items-center gap-4">
          {links.map((item) => (
            <NavLink key={item.href} href={item.href} exact={item.exact}>
              {item.label}
            </NavLink>
          ))}
          {showAdmin && <NavLink href="/admin">管理</NavLink>}
          {status === 'authenticated' ? (
            <>
              <span className="text-sm text-text-inverse-muted">{user?.username}</span>
              <button
                type="button"
                onClick={signOut}
                className="min-h-11 text-text-inverse-muted hover:text-text-inverse"
              >
                退出
              </button>
            </>
          ) : (
            status !== 'loading' && (
              <>
                {registrationEnabled && <NavLink href="/register">注册</NavLink>}
                <NavLink href="/login">登录</NavLink>
              </>
            )
          )}
        </div>
      </nav>
    </header>
  )
}
