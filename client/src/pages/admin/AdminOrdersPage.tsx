import { useCallback, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { useAuth } from '@/features/auth/useAuth'
import { adminOrderSearch, adminOrderStatuses, adminPaymentStatuses, readAdminOrderFilters } from '@/features/admin/orders'
import { loadAdminOrders } from '@/features/admin/orderActions'
import { formatOrderMoney } from '@/features/orders/money'
import { formatOrderDate, orderStatusLabels } from '@/features/orders/presentation'
import { orderPageCount } from '@/features/orders/read'
import { useOrderRead } from '@/features/orders/useOrderRead'
import AdminOrderFrame from './AdminOrderFrame'
import AdminOrderNotice from './AdminOrderNotice'
import { Button, ButtonLink, DataTable, EmptyState, StatusBadge } from './ui'
import { inputClass, tdClass, thClass, toneForStatus } from './uiStyles'

export default function AdminOrdersPage() {
  const { user, loading, error } = useAuth()
  const [params, setParams] = useSearchParams()
  const location = useLocation()
  const { status, payment_status, page, page_size } = readAdminOrderFilters(params)
  const [revision, setRevision] = useState(0)
  const load = useCallback(() => loadAdminOrders({ status, payment_status, page, page_size }), [status, payment_status, page, page_size])
  const scope = JSON.stringify([user?.id, status, payment_status, page, page_size, revision])
  const result = useOrderRead(scope, !!user && !loading && !error, load)
  const retry = () => setRevision((value) => value + 1)
  const next = location.pathname + location.search
  if (!result || result.kind !== 'ready') return <AdminOrderNotice result={result} next={next} retry={retry} />
  const data = result.page
  const pages = orderPageCount(data)
  const filter = { status, payment_status, page, page_size }
  const sizeOptions = [...new Set([20, 50, 100, page_size])].sort((a, b) => a - b)
  return (
    <AdminOrderFrame title="Đơn hàng">
      <section className="bg-surface border border-outline-variant/60 p-4 flex flex-wrap items-end gap-3">
        <label className="space-y-1 text-sm">
          <span className="block font-medium">Nhận áo</span>
          <select className={`${inputClass} w-auto min-w-40`} value={status} onChange={(event) => setParams(adminOrderSearch({ ...filter, status: event.target.value, page: 1 }))}>
            <option value="">Tất cả</option>
            {adminOrderStatuses.map((value) => <option key={value} value={value}>{orderStatusLabels({ status: value, payment_status: '' }).fulfillment}</option>)}
          </select>
        </label>
        <label className="space-y-1 text-sm">
          <span className="block font-medium">Thanh toán</span>
          <select className={`${inputClass} w-auto min-w-40`} value={payment_status} onChange={(event) => setParams(adminOrderSearch({ ...filter, payment_status: event.target.value, page: 1 }))}>
            <option value="">Tất cả</option>
            {adminPaymentStatuses.map((value) => <option key={value} value={value}>{orderStatusLabels({ status: '', payment_status: value }).payment}</option>)}
          </select>
        </label>
        <label className="space-y-1 text-sm">
          <span className="block font-medium">Đơn/trang</span>
          <select className={`${inputClass} w-auto`} value={page_size} onChange={(event) => setParams(adminOrderSearch({ ...filter, page_size: Number(event.target.value), page: 1 }))}>
            {sizeOptions.map((size) => <option key={size} value={size}>{size}</option>)}
          </select>
        </label>
        <Button variant="secondary" onClick={retry}>Tải lại</Button>
      </section>
      {data.data.length === 0 ? (
        <EmptyState title="Không có đơn hàng ở bộ lọc/trang này">
          {data.page > 1 && <Button variant="ghost" onClick={() => setParams(adminOrderSearch({ ...filter, page: 1 }))}>Về trang đầu</Button>}
        </EmptyState>
      ) : (
        <DataTable label="Danh sách đơn hàng">
          <thead>
            <tr>
              <th className={thClass}>Mã đơn</th><th className={thClass}>Khách</th><th className={thClass}>Hình thức</th>
              <th className={thClass}>Nhận áo</th><th className={thClass}>Thanh toán</th><th className={`${thClass} text-right`}>Tổng</th><th className={thClass}><span className="sr-only">Thao tác</span></th>
            </tr>
          </thead>
          <tbody>
            {data.data.map((order) => {
              const labels = orderStatusLabels(order)
              return (
                <tr key={order.id}>
                  <td className={tdClass}>
                    <Link to={`/admin/orders/${encodeURIComponent(order.id)}`} className="font-semibold text-primary underline">{order.order_number}</Link>
                    <p className="text-xs text-on-surface-variant mt-0.5"><time dateTime={order.created_at}>{formatOrderDate(order.created_at)}</time></p>
                  </td>
                  <td className={tdClass}>
                    <p className="font-medium">{order.shipping_address.recipient}</p>
                    <p className="text-xs text-on-surface-variant">{order.shipping_address.phone} · {order.is_guest ? 'Khách vãng lai' : 'Khách tài khoản'}</p>
                  </td>
                  <td className={tdClass}>{order.payment_method === 'cod' ? 'Giao tận nơi (COD)' : order.payment_method === 'cash_at_counter' ? 'Nhận tại quầy' : 'Khác'}</td>
                  <td className={tdClass}><StatusBadge tone={toneForStatus(order.status)}>{labels.fulfillment}</StatusBadge></td>
                  <td className={tdClass}><StatusBadge tone={toneForStatus(order.payment_status)}>{labels.payment}</StatusBadge></td>
                  <td className={`${tdClass} text-right font-semibold text-primary whitespace-nowrap`}>{formatOrderMoney(order.total)}</td>
                  <td className={tdClass}><ButtonLink to={`/admin/orders/${encodeURIComponent(order.id)}`}>Xử lý</ButtonLink></td>
                </tr>
              )
            })}
          </tbody>
        </DataTable>
      )}
      {data.total > 0 && (
        <nav aria-label="Phân trang quản trị đơn" className="flex flex-wrap justify-between items-center gap-4 text-sm">
          <p>{data.total} đơn · Trang {data.page}/{pages} · {data.page_size} đơn/trang</p>
          <div className="flex gap-2">
            <Button variant="secondary" disabled={data.page <= 1} onClick={() => setParams(adminOrderSearch({ ...filter, page: data.page - 1 }))}>Trang trước</Button>
            <Button variant="secondary" disabled={data.page >= pages || data.page >= 100000} onClick={() => setParams(adminOrderSearch({ ...filter, page: data.page + 1 }))}>Trang sau</Button>
          </div>
        </nav>
      )}
    </AdminOrderFrame>
  )
}
