import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '@/features/auth/useAuth'
import { PageSkeleton } from '@/components/ui/Skeleton'

export default function AccountAccessGuard() {
  const { user, loading, error } = useAuth()
  const location = useLocation()
  if (loading) return <PageSkeleton label="Đang tải tài khoản" blocks={2} />
  if (error) return <main className="min-h-screen flex items-center justify-center"><p role="alert">{error}</p></main>
  if (!user) return <Navigate to={`/dang-nhap?next=${encodeURIComponent(location.pathname + location.search)}`} replace />
  return <Outlet />
}
