'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { useAuth, useSite } from '@/app/providers'

function NavLink({ href, children }: { href: string; children: React.ReactNode }) {
  const active = usePathname() === href
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
          <NavLink href="/search">查询</NavLink>
          <NavLink href="/subjects">名单</NavLink>
          <NavLink href="/submit">提交</NavLink>
          {hasRole('admin', 'moderator') && <NavLink href="/admin">管理</NavLink>}
          {status === 'authenticated' ? (
            <>
              <NavLink href="/sanctions">我的处罚</NavLink>
              <span className="text-sm text-gray-300">{user?.username}</span>
              <button type="button" onClick={signOut} className="text-gray-300 hover:text-white">
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
