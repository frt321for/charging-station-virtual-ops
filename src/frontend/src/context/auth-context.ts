import { createContext } from 'react'
import type { AuthUser } from '../types/auth'

export type AuthStatus = 'loading' | 'authenticated' | 'anonymous'

export interface AuthContextValue {
  status: AuthStatus
  user?: AuthUser
  error?: string
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  hasPermission: (permission?: string) => boolean
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined)
