export type NavArea = 'public' | 'account' | 'admin'

export type NavItem = {
  href: string
  label: string
  area: NavArea
  requiresAuth?: boolean
  roles?: string[]
  feature?: 'registration'
  exact?: boolean
}

export const navigationRegistry: NavItem[] = [
  { href: '/search', label: '查询', area: 'public' },
  { href: '/subjects', label: '名单', area: 'public' },
  { href: '/submit', label: '提交', area: 'public' },
  { href: '/sanctions', label: '我的处罚', area: 'account', requiresAuth: true },
  { href: '/login', label: '登录', area: 'public' },
  { href: '/register', label: '注册', area: 'public', feature: 'registration' },
  { href: '/admin', label: '审核', area: 'admin', requiresAuth: true, roles: ['admin', 'moderator'], exact: true },
  { href: '/admin/users', label: '用户', area: 'admin', requiresAuth: true, roles: ['admin'] },
  { href: '/admin/access-lists', label: '访问名单', area: 'admin', requiresAuth: true, roles: ['admin'] },
  { href: '/admin/sanctions', label: '处罚', area: 'admin', requiresAuth: true, roles: ['admin'] },
  { href: '/admin/archives', label: '归档导入导出', area: 'admin', requiresAuth: true, roles: ['admin'] },
  { href: '/admin/settings', label: '站点与配置', area: 'admin', requiresAuth: true, roles: ['admin'] },
]

export type NavigationContext = {
  authenticated: boolean
  roles: string[]
  registrationEnabled: boolean
  area?: NavArea | NavArea[]
}

export function visibleNavigation(ctx: NavigationContext): NavItem[] {
  const areas = ctx.area ? (Array.isArray(ctx.area) ? ctx.area : [ctx.area]) : undefined
  return navigationRegistry.filter((item) => {
    if (areas && !areas.includes(item.area)) return false
    if (item.feature === 'registration' && !ctx.registrationEnabled) return false
    if (item.requiresAuth && !ctx.authenticated) return false
    if (item.area === 'public' && (item.href === '/login' || item.href === '/register') && ctx.authenticated) {
      return false
    }
    if (item.roles && item.roles.length > 0) {
      if (!ctx.authenticated) return false
      if (!item.roles.some((role) => ctx.roles.includes(role))) return false
    }
    return true
  })
}

export function isNavActive(pathname: string, item: NavItem): boolean {
  if (item.exact) return pathname === item.href
  return pathname === item.href || pathname.startsWith(`${item.href}/`)
}
