import { useCallback, useState } from 'react'
import { Link, Navigate, useLocation, useSearchParams } from 'react-router-dom'
import Breadcrumb from '@/components/ui/Breadcrumb'
import { useAuth } from '@/features/auth/useAuth'
import OrderReadNotice from '@/features/orders/OrderReadNotice'
import OrderStatusLabels from '@/features/orders/OrderStatusLabels'
import { formatOrderMoney } from '@/features/orders/money'
import { formatOrderDate, orderFulfillmentLabel, orderFulfillmentStatuses, orderLoginPath, readOrderStatusFilter } from '@/features/orders/presentation'
import { loadMyOrders, orderPageCount, readOrdersPage } from '@/features/orders/read'
import { useOrderRead } from '@/features/orders/useOrderRead'
import OrderReviewButtons, { useOrderReviewStatuses } from '@/features/review/OrderReviewButtons'
import type { Order } from '@/features/orders/types'

const noOrders: Order[] = []

export default function MyOrdersPage() {
  const { user, loading, error } = useAuth()
  const [params, setParams] = useSearchParams()
  const location = useLocation()
  const [revision, setRevision] = useState(0)
  const page = readOrdersPage(params.get('page'))
  const status = readOrderStatusFilter(params)
  const pageSize = 20
  const scope = JSON.stringify([user?.id, status, page, pageSize, revision])
  const load = useCallback(() => loadMyOrders(page, pageSize, status), [page, pageSize, status])
  const result = useOrderRead(scope, !!user && !loading && !error, load)
  const reviewStatuses = useOrderReviewStatuses(result?.kind === 'ready' ? result.page.data : noOrders)
  const next = location.pathname + location.search
  if (result?.kind === 'unauthenticated') return <Navigate to={orderLoginPath(next)} replace />
  if (!result || result.kind !== 'ready')
    return (
      <OrderReadNotice
        kind={!result ? 'loading' : 'error'}
        next={next}
        retry={() => setRevision((value) => value + 1)}
        backTo="/tai-khoan"
        backLabel="Về tài khoản"
      />
    )
  const data = result.page
  const pages = orderPageCount(data)
  function goToPage(value: number) {
    setParams(status ? { status, page: String(value) } : { page: String(value) })
  }
  function changeStatus(value: string) {
    setParams(value ? { status: value, page: '1' } : { page: '1' })
  }

  return (
    <main className="max-w-[1180px] mx-auto px-5 py-8 lg:py-14 space-y-7">
      <Breadcrumb items={[{ label: 'Trang chủ', to: '/' }, { label: 'Tài khoản', to: '/tai-khoan' }, { label: 'Đơn hàng của tôi' }]} />
      <header className="flex flex-wrap justify-between items-end gap-4">
        <div>
          <p className="text-secondary text-sm uppercase tracking-widest">Không gian riêng</p>
          <h1 className="font-headline-lg text-4xl text-primary mt-2">Đơn hàng của tôi</h1>
          <p className="text-on-surface-variant mt-3">Theo dõi nhận áo và thanh toán riêng cho từng đơn.</p>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <label className="space-y-2">
            <span className="block text-sm text-on-surface-variant">Trạng thái đơn</span>
            <select className="border border-outline-variant bg-surface px-4 py-3 min-w-48" value={status} onChange={(event) => changeStatus(event.target.value)}>
              <option value="">Tất cả</option>
              {orderFulfillmentStatuses.map((value) => (
                <option key={value} value={value}>
                  {orderFulfillmentLabel(value)}
                </option>
              ))}
            </select>
          </label>
          <button className="border border-primary text-primary px-5 py-3" type="button" onClick={() => setRevision((value) => value + 1)}>
            Tải lại
          </button>
        </div>
      </header>
      {data.data.length === 0 ? (
        <section className="bg-surface-container-low p-8 text-center space-y-4">
          <h2 className="font-headline-md text-2xl text-primary">
            {data.total > 0 ? 'Chưa có đơn hàng ở trang này' : status ? `Không có đơn "${orderFulfillmentLabel(status)}"` : 'Bạn chưa có đơn hàng nào'}
          </h2>
          {status ? (
            <button type="button" className="inline-block border border-primary text-primary px-6 py-3" onClick={() => changeStatus('')}>
              Xem tất cả đơn
            </button>
          ) : (
            <Link to="/bo-suu-tap" className="inline-block bg-primary text-on-primary px-6 py-3">
              Khám phá bộ sưu tập
            </Link>
          )}
          {data.page > 1 && (
            <button type="button" className="block mx-auto underline text-primary" onClick={() => goToPage(1)}>
              Về trang đầu
            </button>
          )}
        </section>
      ) : (
        <ul className="space-y-5">
          {data.data.map((order) => (
            <li key={order.id} className="bg-surface-container-lowest border border-outline-variant p-5 sm:p-7 space-y-5">
              <div className="flex flex-wrap justify-between gap-4">
                <div>
                  <h2 className="font-headline-sm text-2xl text-primary">
                    <Link className="underline underline-offset-4" to={`/don-hang/${encodeURIComponent(order.id)}`}>
                      {order.order_number}
                    </Link>
                  </h2>
                  <p className="text-sm text-on-surface-variant mt-2">
                    Ngày đặt: <time dateTime={order.created_at}>{formatOrderDate(order.created_at)}</time>
                  </p>
                </div>
                <div>
                  <p className="text-sm text-on-surface-variant">Tổng tiền</p>
                  <p className="font-semibold text-primary text-xl mt-1">{formatOrderMoney(order.total)}</p>
                </div>
              </div>
              <OrderStatusLabels order={order} />
              <div className="flex flex-wrap items-center gap-3">
                <Link to={`/don-hang/${encodeURIComponent(order.id)}`} className="inline-block border border-primary text-primary px-5 py-2">
                  Xem chi tiết đơn
                </Link>
                <OrderReviewButtons order={order} statuses={reviewStatuses} />
              </div>
            </li>
          ))}
        </ul>
      )}
      {data.total > 0 && (
        <nav aria-label="Phân trang đơn hàng" className="flex flex-wrap justify-between items-center gap-4 border-t border-outline-variant pt-6">
          <p className="text-on-surface-variant">
            {data.total} đơn · Trang {data.page}/{pages} · {data.page_size} đơn/trang
          </p>
          <div className="flex gap-3">
            <button
              type="button"
              className="border border-primary text-primary px-4 py-2 disabled:opacity-40"
              disabled={data.page <= 1}
              onClick={() => goToPage(data.page - 1)}
            >
              Trang trước
            </button>
            <button
              type="button"
              className="border border-primary text-primary px-4 py-2 disabled:opacity-40"
              disabled={data.page >= pages || data.page >= 100000}
              onClick={() => goToPage(data.page + 1)}
            >
              Trang sau
            </button>
          </div>
        </nav>
      )}
    </main>
  )
}
