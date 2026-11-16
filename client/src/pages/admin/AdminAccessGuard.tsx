import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '@/features/auth/useAuth'

export default function AdminAccessGuard() {
  const { user, loading, error } = useAuth()
  if (loading) return <main className="min-h-screen flex items-center justify-center"><p role="status">Đang xác minh quyền quản trị...</p></main>
  if (error) return <main className="min-h-screen flex items-center justify-center"><p role="alert">{error}</p></main>
  if (!user) return <Navigate to="/dang-nhap?next=/admin/catalog" replace />
  if (user.role !== 'admin') return <Navigate to="/403" replace />
  return <Outlet />
}
