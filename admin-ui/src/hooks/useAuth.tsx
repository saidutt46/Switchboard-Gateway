import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import * as authApi from '../api/auth'
import type { AdminUser, AuthTokens } from '../api/types'

interface AuthContextValue {
  user: AdminUser | null
  isLoading: boolean
  isAuthenticated: boolean
  isAdmin: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  getAccessToken: () => string | null
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

const TOKEN_KEY = 'switchboard_access_token'
const REFRESH_KEY = 'switchboard_refresh_token'
const EXPIRES_KEY = 'switchboard_token_expires'

function storeTokens(tokens: AuthTokens) {
  localStorage.setItem(TOKEN_KEY, tokens.access_token)
  localStorage.setItem(REFRESH_KEY, tokens.refresh_token)
  localStorage.setItem(EXPIRES_KEY, String(Date.now() + tokens.expires_in * 1000))
}

function clearTokens() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(REFRESH_KEY)
  localStorage.removeItem(EXPIRES_KEY)
}

function getStoredAccessToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

function getStoredRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY)
}

function isTokenExpiringSoon(): boolean {
  const expires = localStorage.getItem(EXPIRES_KEY)
  if (!expires) return true
  // Refresh if less than 2 minutes remaining
  return Date.now() > Number(expires) - 120000
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AdminUser | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const scheduleRefresh = useCallback((expiresIn: number) => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current)
    // Refresh 2 minutes before expiry
    const refreshIn = Math.max((expiresIn - 120) * 1000, 10000)
    refreshTimerRef.current = setTimeout(async () => {
      const refreshToken = getStoredRefreshToken()
      if (!refreshToken) return
      try {
        const tokens = await authApi.refreshTokens(refreshToken)
        storeTokens(tokens)
        scheduleRefresh(tokens.expires_in)
      } catch {
        // Refresh failed — force logout
        clearTokens()
        setUser(null)
      }
    }, refreshIn)
  }, [])

  // Initialize — check stored token on mount
  useEffect(() => {
    const init = async () => {
      const accessToken = getStoredAccessToken()
      if (!accessToken) {
        setIsLoading(false)
        return
      }

      // If token is expiring soon, try refresh first
      if (isTokenExpiringSoon()) {
        const refreshToken = getStoredRefreshToken()
        if (refreshToken) {
          try {
            const tokens = await authApi.refreshTokens(refreshToken)
            storeTokens(tokens)
            scheduleRefresh(tokens.expires_in)
          } catch {
            clearTokens()
            setIsLoading(false)
            return
          }
        } else {
          clearTokens()
          setIsLoading(false)
          return
        }
      }

      // Fetch current user
      try {
        const me = await authApi.getMe()
        setUser(me)
        const expires = localStorage.getItem(EXPIRES_KEY)
        if (expires) {
          const remainingSeconds = Math.max(0, (Number(expires) - Date.now()) / 1000)
          scheduleRefresh(remainingSeconds)
        }
      } catch {
        clearTokens()
      }
      setIsLoading(false)
    }

    init()

    return () => {
      if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current)
    }
  }, [scheduleRefresh])

  const login = useCallback(async (username: string, password: string) => {
    const tokens = await authApi.login({ username, password })
    storeTokens(tokens)
    const me = await authApi.getMe()
    setUser(me)
    scheduleRefresh(tokens.expires_in)
  }, [scheduleRefresh])

  const logout = useCallback(() => {
    clearTokens()
    setUser(null)
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current)
  }, [])

  const getAccessToken = useCallback((): string | null => {
    return getStoredAccessToken()
  }, [])

  return (
    <AuthContext.Provider value={{
      user,
      isLoading,
      isAuthenticated: !!user,
      isAdmin: user?.role === 'admin',
      login,
      logout,
      getAccessToken,
    }}>
      {children}
    </AuthContext.Provider>
  )
}
