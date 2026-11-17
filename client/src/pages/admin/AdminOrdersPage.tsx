import { useCallback, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { useAuth } from '@/features/auth/useAuth'
import { adminOrderSearch, adminOrderStatuses, adminPaymentStatuses, readAdminOrderFilters } from '@/features/admin/orders'
import { loadAdminOrders } from '@/features/admin/orderActions'
import { formatOrderMoney } from '@/features/orders/money'
import { formatOrderDate, orderPaymentLabel, orderStatusLabels } from '@/features/orders/presentation'
import { orderPageCount } from '@/features/orders/read'
import { useOrderRead } from '@/features/orders/useOrderRead'
import OrderStatusLabels from '@/features/orders/OrderStatusLabels'
import AdminOrderFrame from './AdminOrderFrame'
import AdminOrderNotice from './AdminOrderNotice'

const inputClass = 'w-full border border-outline-variant bg-surface p-3'
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
  return (
    <AdminOrderFrame title="Quản trị đơn hàng">
      <section className="bg-surface p-5 grid sm:grid-cols-2 lg:grid-cols-4 gap-4 items-end">
        <label className="space-y-2">
          <span>Trạng thái nhận áo</span>
          <select className={inputClass} value={status} onChange={(event) => setParams(adminOrderSearch({ ...filter, status: event.target.value, page: 1 }))}>
            <option value="">Tất cả</option>
            {adminOrderStatuses.map((value) => (
              <option key={value} value={value}>
                {orderStatusLabels({ status: value, payment_status: '' }).fulfillment}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-2">
          <span>Trạng thái thanh toán</span>
          <select
            className={inputClass}
            value={payment_status}
            onChange={(event) => setParams(adminOrderSearch({ ...filter, payment_status: event.target.value, page: 1 }))}
          >
            <option value="">Tất cả</option>
            {adminPaymentStatuses.map((value) => (
              <option key={value} value={value}>
                {orderStatusLabels({ status: '', payment_status: value }).payment}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-2">
          <span>Số đơn mỗi trang</span>
          <select
            className={inputClass}
            value={page_size}
            onChange={(event) => setParams(adminOrderSearch({ ...filter, page_size: Number(event.target.value), page: 1 }))}
          >
            {[...new Set([20, 50, 100, page_size])]
              .sort((a, b) => a - b)
              .map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
          </select>
        </label>
        <button type="button" className="border border-primary text-primary px-5 py-3" onClick={retry}>
          Tải lại danh sách
        </button>
      </section>
      {data.data.length === 0 ? (
        <section className="bg-surface p-8 text-center space-y-4">
          <h2 className="font-headline-md text-primary text-2xl">Không có đơn hàng ở bộ lọc/trang này</h2>
          {data.page > 1 && (
            <button type="button" className="text-primary underline" onClick={() => setParams(adminOrderSearch({ ...filter, page: 1 }))}>
              Về trang đầu
            </button>
          )}
        </section>
      ) : (
        <ul className="grid xl:grid-cols-2 gap-5">
          {data.data.map((order) => (
            <li key={order.id} className="bg-surface border border-outline-variant p-5 sm:p-6 space-y-5">
              <div className="flex flex-wrap justify-between gap-4">
                <div>
                  <h2 className="font-title-editorial text-2xl text-primary">
                    <Link to={`/admin/orders/${encodeURIComponent(order.id)}`} className="underline">
                      {order.order_number}
                    </Link>
                  </h2>
                  <p className="text-sm text-on-surface-variant mt-1">
                    Ngày đặt: <time dateTime={order.created_at}>{formatOrderDate(order.created_at)}</time>
                  </p>
                </div>
                <p className="text-primary font-semibold text-xl">{formatOrderMoney(order.total)}</p>
              </div>
              <div>
                <p className="font-semibold">
                  {order.shipping_address.recipient} · {order.shipping_address.phone}
                </p>
                <p className="text-sm text-on-surface-variant mt-1">
                  {orderPaymentLabel(order.payment_method)} · {order.is_guest ? 'Khách vãng lai' : 'Khách tài khoản'}
                </p>
              </div>
              <OrderStatusLabels order={order} />
              <Link to={`/admin/orders/${encodeURIComponent(order.id)}`} className="inline-block border border-primary text-primary px-5 py-2">
                Xem và xử lý đơn
              </Link>
            </li>
          ))}
        </ul>
      )}
      {data.total > 0 && (
        <nav aria-label="Phân trang quản trị đơn" className="flex flex-wrap justify-between items-center gap-4 border-t border-outline-variant pt-5">
          <p>
            {data.total} đơn · Trang {data.page}/{pages} · {data.page_size} đơn/trang
          </p>
          <div className="flex gap-3">
            <button
              type="button"
              className="border border-primary px-4 py-2 disabled:opacity-40"
              disabled={data.page <= 1}
              onClick={() => setParams(adminOrderSearch({ ...filter, page: data.page - 1 }))}
            >
              Trang trước
            </button>
            <button
              type="button"
              className="border border-primary px-4 py-2 disabled:opacity-40"
              disabled={data.page >= pages || data.page >= 100000}
              onClick={() => setParams(adminOrderSearch({ ...filter, page: data.page + 1 }))}
            >
              Trang sau
            </button>
          </div>
        </nav>
      )}
    </AdminOrderFrame>
  )
}
