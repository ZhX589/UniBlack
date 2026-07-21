'use client'

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react'
import { apiRequest, configureApiClient } from '@/lib/api'
import type { AuthUser, PublicSettings } from '@/lib/types'
import { settingBool, settingString } from '@/lib/settings'

type AuthState = {
  status: 'loading' | 'anonymous' | 'authenticated'
  user: AuthUser | null
  login: (token: string, refresh: string) => void
  logout: () => void
  hasRole: (...roles: string[]) => boolean
}

type SiteState = {
  status: 'loading' | 'ready' | 'fallback'
  name: string
  description: string
  logoUrl: string
  themeColor: string
  registrationEnabled: boolean
}

const AuthContext = createContext<AuthState | null>(null)
const SiteContext = createContext<SiteState>({
  status: 'loading',
  name: 'UniBlack',
  description: '',
  logoUrl: '',
  themeColor: '#2563EB',
  registrationEnabled: true,
})

function decodeToken(token: string): AuthUser | null {
  try {
    const payload = token.split('.')[1]
    const decoded = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
    if (!decoded.user_id || !decoded.exp || decoded.exp * 1000 <= Date.now()) return null
    return {
      user_id: decoded.user_id,
      username: decoded.username || '',
      email: decoded.email || '',
      roles: decoded.roles || [],
    }
  } catch {
    return null
  }
}

function applyThemeColor(color: string) {
  if (typeof document === 'undefined') return
  document.documentElement.style.setProperty('--primary', color || '#2563EB')
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [status, setStatus] = useState<AuthState['status']>('loading')
  const [site, setSite] = useState<SiteState>({
    status: 'loading',
    name: 'UniBlack',
    description: '',
    logoUrl: '',
    themeColor: '#2563EB',
    registrationEnabled: true,
  })

  const logout = useCallback(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    setUser(null)
    setStatus('anonymous')
  }, [])

  useEffect(() => {
    configureApiClient({
      getToken: () => localStorage.getItem('token'),
      onUnauthorized: logout,
    })
    const token = localStorage.getItem('token') || ''
    const identity = decodeToken(token)
    if (!identity && token) {
      localStorage.removeItem('token')
      localStorage.removeItem('refresh_token')
    }
    setUser(identity)
    setStatus(identity ? 'authenticated' : 'anonymous')

    apiRequest<PublicSettings>('/api/settings/public')
      .then((settings) => {
        const themeColor = settingString(settings, 'site.theme_color', '#2563EB')
        applyThemeColor(themeColor)
        setSite({
          status: 'ready',
          name: settingString(settings, 'site.name', 'UniBlack'),
          description: settingString(settings, 'site.description', ''),
          logoUrl: settingString(settings, 'site.logo_url', ''),
          themeColor,
          registrationEnabled: settingBool(settings, 'auth.registration_enabled', true),
        })
      })
      .catch(() => {
        applyThemeColor('#2563EB')
        setSite((current) => ({ ...current, status: 'fallback' }))
      })
  }, [logout])

  const auth = useMemo<AuthState>(
    () => ({
      status,
      user,
      login: (token, refresh) => {
        localStorage.setItem('token', token)
        localStorage.setItem('refresh_token', refresh)
        setUser(decodeToken(token))
        setStatus('authenticated')
      },
      logout,
      hasRole: (...roles) => !!user && roles.some((role) => user.roles.includes(role)),
    }),
    [logout, status, user],
  )

  return (
    <SiteContext.Provider value={site}>
      <AuthContext.Provider value={auth}>{children}</AuthContext.Provider>
    </SiteContext.Provider>
  )
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('useAuth must be used within Providers')
  return value
}

export function useSite() {
  return useContext(SiteContext)
}
