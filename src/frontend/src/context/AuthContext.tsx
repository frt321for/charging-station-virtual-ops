import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { getCurrentUser, login as loginRequest, logout as logoutRequest } from '../services/auth-api'
import { ApiError, getAuthToken, setAuthToken } from '../services/api-client'
import { AuthContext, type AuthStatus } from './auth-context'
import type { AuthUser } from '../types/auth'

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState<AuthStatus>(() => (getAuthToken() ? 'loading' : 'anonymous'))
  const [user, setUser] = useState<AuthUser | undefined>()
  const [error, setError] = useState<string | undefined>()

  useEffect(() => {
    let mounted = true

    if (!getAuthToken()) {
      return () => {
        mounted = false
      }
    }

    getCurrentUser()
      .then((currentUser) => {
        if (!mounted) return
        setUser(currentUser)
        setStatus('authenticated')
        setError(undefined)
      })
      .catch((caught: unknown) => {
        if (!mounted) return
        setAuthToken(undefined)
        setUser(undefined)
        setStatus('anonymous')
        setError(caught instanceof ApiError && caught.status !== 401 ? caught.message : undefined)
      })

    return () => {
      mounted = false
    }
  }, [])

  const login = useCallback(
    async (username: string, password: string) => {
      setStatus('loading')
      setError(undefined)
      try {
        const result = await loginRequest(username, password)
        setAuthToken(result.token)
        setUser(result.user)
        setStatus('authenticated')
        await queryClient.invalidateQueries()
      } catch (caught) {
        setAuthToken(undefined)
        setUser(undefined)
        setStatus('anonymous')
        setError(caught instanceof ApiError ? caught.message : '登录失败')
        throw caught
      }
    },
    [queryClient],
  )

  const logout = useCallback(async () => {
    try {
      await logoutRequest()
    } finally {
      setAuthToken(undefined)
      setUser(undefined)
      setStatus('anonymous')
      setError(undefined)
      queryClient.clear()
    }
  }, [queryClient])

  const hasPermission = useCallback(
    (permission?: string) => {
      if (!permission) return true
      return user?.permissions.includes(permission) ?? false
    },
    [user?.permissions],
  )

  const value = useMemo(
    () => ({ status, user, error, login, logout, hasPermission }),
    [error, hasPermission, login, logout, status, user],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
