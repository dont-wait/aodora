import { useEffect, useMemo, useState } from 'react'
import { authApi, AuthApiError, type AuthUser } from './api'
import { AuthContext, type AuthProviderProps } from './context'

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  async function refresh() {
    setLoading(true)
    try {
      setUser(await authApi.me())
      setError('')
    } catch (err) {
      if (err instanceof AuthApiError && err.status === 401) {
        setUser(null)
        setError('')
      } else {
        setError(err instanceof Error ? err.message : 'Không thể kiểm tra phiên đăng nhập')
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    let active = true
    authApi.me()
      .then((next) => { if (active) { setUser(next); setError('') } })
      .catch((err: unknown) => {
        if (!active) return
        if (err instanceof AuthApiError && err.status === 401) { setUser(null); setError('') }
        else setError(err instanceof Error ? err.message : 'Không thể kiểm tra phiên đăng nhập')
      })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [])

  async function login(email: string, password: string) {
    const next = await authApi.login({ email, password })
    setUser(next)
    setError('')
    return next
  }

  async function register(input: { full_name: string; email: string; password: string; phone?: string }) {
    const next = await authApi.register(input)
    setError('')
    return next
  }

  async function logout() {
    await authApi.logout()
    setUser(null)
    setError('')
  }

  const value = useMemo(() => ({ user, loading, error, login, register, logout, refresh }), [user, loading, error])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
