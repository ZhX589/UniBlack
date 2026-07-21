import { describe, expect, it } from 'vitest'
import { isNavActive, visibleNavigation } from '@/lib/navigation'

describe('visibleNavigation', () => {
  it('hides management and account links for anonymous visitors', () => {
    const items = visibleNavigation({
      authenticated: false,
      roles: [],
      registrationEnabled: true,
    })
    expect(items.map((item) => item.href)).toEqual(['/search', '/subjects', '/submit', '/login', '/register'])
  })

  it('hides registration when disabled and shows account links when authenticated', () => {
    const items = visibleNavigation({
      authenticated: true,
      roles: ['user'],
      registrationEnabled: false,
    })
    expect(items.map((item) => item.href)).toEqual(['/search', '/subjects', '/submit', '/sanctions'])
  })

  it('filters admin links by role', () => {
    const moderator = visibleNavigation({
      authenticated: true,
      roles: ['moderator'],
      registrationEnabled: true,
      area: 'admin',
    })
    expect(moderator.map((item) => item.href)).toEqual(['/admin'])

    const admin = visibleNavigation({
      authenticated: true,
      roles: ['admin'],
      registrationEnabled: true,
      area: 'admin',
    })
    expect(admin.map((item) => item.href)).toContain('/admin/settings')
    expect(admin.map((item) => item.href)).toContain('/admin/users')
  })

  it('matches nested admin routes without exact equality', () => {
    expect(isNavActive('/admin/users', { href: '/admin/users', label: '用户', area: 'admin' })).toBe(true)
    expect(isNavActive('/admin/users', { href: '/admin', label: '审核', area: 'admin', exact: true })).toBe(false)
  })
})
