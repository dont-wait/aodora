import { useEffect, useState } from 'react'
import { Navigate, Outlet } from 'react-router-dom'
import { AdminApiError, adminCatalog } from '@/features/admin/catalog'

// Until KR-01 exposes an authenticated role probe, the protected admin API
// serves as the verification source. Never trust a browser flag or mock role.
export default function AdminAccessGuard() {
  const [attempt, setAttempt] = useState(0)
  const [status, setStatus] = useState<'checking' | 'allowed' | 'denied' | 'unavailable'>('checking')
  useEffect(() => {
    let active = true
    adminCatalog.categories()
      .then(() => { if (active) setStatus('allowed') })
      .catch((err: unknown) => {
        if (!active) return
        setStatus(err instanceof AdminApiError && (err.status === 401 || err.status === 403) ? 'denied' : 'unavailable')
      })
    return () => { active = false }
  }, [attempt])
  if (status === 'denied') return <Navigate to="/403" replace />
  if (status === 'allowed') return <Outlet />
  return <main className="min-h-screen bg-background text-on-surface p-8 flex items-center justify-center">
    <div className="bg-surface border border-outline/30 p-8 max-w-lg w-full space-y-3">
      {status === 'checking' ? <p role="status">Đang xác minh quyền quản trị...</p> : <>
        <h1 className="text-xl text-primary">Chưa thể xác minh quyền quản trị</h1>
        <p>Không thể kết nối dịch vụ quản trị. Không có dữ liệu hoặc thao tác quản trị nào được hiển thị.</p>
        <button type="button" className="bg-primary text-on-primary px-4 py-2" onClick={() => { setStatus('checking'); setAttempt((v) => v + 1) }}>Thử lại</button>
      </>}
    </div>
  </main>
}
