import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '@/features/auth/useAuth'

export default function AccountAccessGuard() {
  const { user, loading, error } = useAuth()
  const location = useLocation()
  if (loading) return <main className="min-h-screen flex items-center justify-center"><p role="status">Đang tải tài khoản...</p></main>
  if (error) return <main className="min-h-screen flex items-center justify-center"><p role="alert">{error}</p></main>
  if (!user) return <Navigate to={`/dang-nhap?next=${encodeURIComponent(location.pathname + location.search)}`} replace />
  return <Outlet />
}
