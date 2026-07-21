'use client'

import Link from 'next/link'
import { useEffect } from 'react'
import { usePathname, useRouter } from 'next/navigation'
import { useAuth } from '@/app/providers'
import { isNavActive, visibleNavigation } from '@/lib/navigation'

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const { status, user, hasRole } = useAuth()
  const router = useRouter()
  const path = usePathname()

  useEffect(() => {
    if (status === 'anonymous') router.replace('/login?next=/admin')
  }, [status, router])

  if (status === 'loading') return <p className="p-8 text-gray-500">正在验证管理权限...</p>
  if (status === 'anonymous') return null
  if (!hasRole('admin', 'moderator')) return <p className="p-8 text-red-700">403：没有管理权限。</p>

  const links = visibleNavigation({
    authenticated: true,
    roles: user?.roles || [],
    registrationEnabled: true,
    area: 'admin',
  })

  return (
    <div className="grid gap-6 md:grid-cols-[13rem_1fr]">
      <aside className="rounded-lg bg-gray-900 p-4 text-white">
        <p className="mb-4 font-semibold">管理控制台</p>
        <nav className="space-y-2">
          {links.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              aria-current={isNavActive(path, item) ? 'page' : undefined}
              className={
                isNavActive(path, item)
                  ? 'block rounded bg-gray-700 p-2'
                  : 'block rounded p-2 text-gray-300 hover:bg-gray-800'
              }
            >
              {item.label}
            </Link>
          ))}
        </nav>
      </aside>
      <section>{children}</section>
    </div>
  )
}
