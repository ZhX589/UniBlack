'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { useAuth, useSite } from '@/app/providers'
import { isNavActive, visibleNavigation } from '@/lib/navigation'

function NavLink({ href, children, exact }: { href: string; children: React.ReactNode; exact?: boolean }) {
  const pathname = usePathname()
  const active = isNavActive(pathname, { href, label: '', area: 'public', exact })
  return (
    <Link
      href={href}
      aria-current={active ? 'page' : undefined}
      className={active ? 'font-medium text-white' : 'text-gray-300 hover:text-white'}
    >
      {children}
    </Link>
  )
}

export function SiteHeader() {
  const { status, user, logout, hasRole } = useAuth()
  const { name, registrationEnabled } = useSite()
  const router = useRouter()

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
    <nav className="bg-gray-900 p-4 text-white">
      <div className="container mx-auto flex flex-wrap items-center justify-between gap-4">
        <Link href="/" className="text-xl font-bold">
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
              <span className="text-sm text-gray-300">{user?.username}</span>
              <button type="button" onClick={signOut} className="min-h-11 text-gray-300 hover:text-white">
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
      </div>
    </nav>
  )
}
