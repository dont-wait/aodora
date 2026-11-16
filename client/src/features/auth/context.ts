import { createContext, type ReactNode } from 'react'
import type { AuthUser } from './api'

export type AuthContextValue = {
  user: AuthUser | null
  loading: boolean
  error: string
  login: (email: string, password: string) => Promise<AuthUser>
  register: (input: { full_name: string; email: string; password: string; phone?: string }) => Promise<AuthUser>
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)
export type AuthProviderProps = { children: ReactNode }
