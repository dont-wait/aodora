import { Navigate } from 'react-router-dom'
import { orderLoginPath } from '@/features/orders/presentation'
import type { AdminOrderFailure } from '@/features/admin/orderActions'
import AdminOrderFrame from './AdminOrderFrame'

export default function AdminOrderNotice({ result, next, retry, title = 'Quản trị đơn hàng' }: { result: AdminOrderFailure | null; next: string; retry: () => void; title?: string }) {
  if (result?.kind === 'unauthenticated') return <Navigate to={orderLoginPath(next)} replace />
  if (result?.kind === 'forbidden') return <Navigate to="/403" replace />
  return (
    <AdminOrderFrame title={title}>
      {!result ? (
        <p role="status">Đang tải và kiểm tra quyền quản trị...</p>
      ) : (
        <section className="bg-surface p-6 space-y-4">
          <p role="alert" className="text-error">
            {result.message}
          </p>
          {result.kind === 'error' && (
            <button type="button" className="bg-primary text-on-primary px-5 py-3" onClick={retry}>
              Thử tải lại
            </button>
          )}
        </section>
      )}
    </AdminOrderFrame>
  )
}
