import { Navigate } from 'react-router-dom'
import { orderLoginPath } from '@/features/orders/presentation'
import type { AdminOrderFailure } from '@/features/admin/orderActions'
import AdminOrderFrame from './AdminOrderFrame'
import { Alert, Button, TableSkeleton } from './ui'

export default function AdminOrderNotice({ result, next, retry, title = 'Quản trị đơn hàng' }: { result: AdminOrderFailure | null; next: string; retry: () => void; title?: string }) {
  if (result?.kind === 'unauthenticated') return <Navigate to={orderLoginPath(next)} replace />
  if (result?.kind === 'forbidden') return <Navigate to="/403" replace />
  return (
    <AdminOrderFrame title={title}>
      {!result ? (
        <TableSkeleton label="Đang tải và kiểm tra quyền quản trị" />
      ) : (
        <Alert action={result.kind === 'error' ? <Button onClick={retry}>Thử tải lại</Button> : undefined}>{result.message}</Alert>
      )}
    </AdminOrderFrame>
  )
}
