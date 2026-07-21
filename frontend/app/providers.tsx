'use client'

import { createContext, useContext, useEffect, useState } from 'react'
import { settingBool, settingString } from '@/lib/settings'

type User = { user_id: string; username: string; email: string; roles: string[] }
type AuthState = { status: 'loading' | 'anonymous' | 'authenticated'; user: User | null; login: (token: string, refresh: string) => void; logout: () => void; hasRole: (...roles: string[]) => boolean }
type SiteState = { name: string; registrationEnabled: boolean }

const AuthContext = createContext<AuthState | null>(null)
const SiteContext = createContext<SiteState>({ name: 'UniBlack', registrationEnabled: true })

function decodeToken(token: string): User | null {
  try {
    const payload = token.split('.')[1]
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
    if (!decoded.user_id || !decoded.exp || decoded.exp * 1000 <= Date.now()) return null
    return { user_id: decoded.user_id, username: decoded.username || '', email: decoded.email || '', roles: decoded.roles || [] }
  } catch { return null }
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [status, setStatus] = useState<AuthState['status']>('loading')
  const [site, setSite] = useState<SiteState>({ name: 'UniBlack', registrationEnabled: true })

  useEffect(() => {
    const token = localStorage.getItem('token') || ''
    const identity = decodeToken(token)
    if (!identity && token) { localStorage.removeItem('token'); localStorage.removeItem('refresh_token') }
    setUser(identity); setStatus(identity ? 'authenticated' : 'anonymous')
    fetch('/api/settings/public').then((r) => r.ok ? r.json() : {}).then((settings) => {
      setSite({ name: settingString(settings, 'site.name', 'UniBlack'), registrationEnabled: settingBool(settings, 'auth.registration_enabled', true) })
    }).catch(() => undefined)
  }, [])

  const auth: AuthState = {
    status, user,
    login: (token, refresh) => { localStorage.setItem('token', token); localStorage.setItem('refresh_token', refresh); setUser(decodeToken(token)); setStatus('authenticated') },
    logout: () => { localStorage.removeItem('token'); localStorage.removeItem('refresh_token'); setUser(null); setStatus('anonymous') },
    hasRole: (...roles) => !!user && roles.some((role) => user.roles.includes(role)),
  }
  return <SiteContext.Provider value={site}><AuthContext.Provider value={auth}>{children}</AuthContext.Provider></SiteContext.Provider>
}

export function useAuth() { const value = useContext(AuthContext); if (!value) throw new Error('useAuth must be used within Providers'); return value }
export function useSite() { return useContext(SiteContext) }
