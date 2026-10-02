'use client'

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { apiRequest, configureApiClient, refreshApiSession } from '@/lib/api'
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

// Monotonic across provider remounts, including StrictMode effect replays.
let sessionGeneration = 0
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
  const generation = useRef(0)
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
    generation.current = ++sessionGeneration
    window.localStorage.removeItem('token')
    window.localStorage.removeItem('refresh_token')
    setUser(null)
    setStatus('anonymous')
  }, [])

  useEffect(() => {
    let active = true
    generation.current = ++sessionGeneration
    configureApiClient({
      getToken: () => window.localStorage.getItem('token'),
      getGeneration: () => generation.current,
      onUnauthorized: logout,
      onRefresh: async () => {
        const owner = generation.current
        const access = window.localStorage.getItem('token')
        const refresh = window.localStorage.getItem('refresh_token')
        const ownsSession = () => active && generation.current === owner &&
          window.localStorage.getItem('token') === access && window.localStorage.getItem('refresh_token') === refresh
        if (!refresh) return false
        try {
          const tokens = await apiRequest<{ access_token: string; refresh_token: string }>('/api/auth/refresh', {
            method: 'POST', json: { refresh_token: refresh },
          })
          if (!ownsSession()) return false
          const identity = decodeToken(tokens.access_token)
          if (!identity || !tokens.refresh_token) throw new Error('invalid refreshed session')
          window.localStorage.setItem('token', tokens.access_token)
          window.localStorage.setItem('refresh_token', tokens.refresh_token)
          setUser(identity)
          setStatus('authenticated')
          return true
        } catch {
          if (ownsSession()) logout()
          return false
        }
      },
    })
    const token = window.localStorage.getItem('token') || ''
    const identity = decodeToken(token)
    if (identity) {
      setUser(identity)
      setStatus('authenticated')
    } else if (window.localStorage.getItem('refresh_token')) {
      const owner = generation.current
      void refreshApiSession().then((renewed) => {
        if (active && !renewed && generation.current === owner) logout()
      })
    } else {
      logout()
    }

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

    return () => {
      active = false
      configureApiClient({ onUnauthorized: null, onRefresh: null })
    }
  }, [logout])

  const auth = useMemo<AuthState>(
    () => ({
      status,
      user,
      login: (token, refresh) => {
        generation.current = ++sessionGeneration
        window.localStorage.setItem('token', token)
        window.localStorage.setItem('refresh_token', refresh)
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
