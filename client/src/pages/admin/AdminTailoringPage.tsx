import { useCallback, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { useAuth } from '@/features/auth/useAuth'
import { adminTailoringSearch, adminTailoringStatuses, readAdminTailoringFilters, tailoringStatusLabel } from '@/features/admin/tailoring'
import { loadAdminTailorings } from '@/features/admin/tailoringActions'
import { formatOrderDate } from '@/features/orders/presentation'
import { orderPageCount } from '@/features/orders/read'
import { useOrderRead } from '@/features/orders/useOrderRead'
import AdminOrderFrame from './AdminOrderFrame'
import AdminOrderNotice from './AdminOrderNotice'

const inputClass = 'w-full border border-outline-variant bg-surface p-3'
const title = 'Yêu cầu may đo'
export default function AdminTailoringPage() {
  const { user, loading, error } = useAuth()
  const [params, setParams] = useSearchParams()
  const location = useLocation()
  const { status, page, page_size } = readAdminTailoringFilters(params)
  const [revision, setRevision] = useState(0)
  const load = useCallback(() => loadAdminTailorings({ status, page, page_size }), [status, page, page_size])
  const scope = JSON.stringify([user?.id, status, page, page_size, revision])
  const result = useOrderRead(scope, !!user && !loading && !error, load)
  const retry = () => setRevision((value) => value + 1)
  const next = location.pathname + location.search
  if (!result || result.kind !== 'ready') return <AdminOrderNotice title={title} result={result} next={next} retry={retry} />
  const data = result.page
  const pages = orderPageCount(data)
  const filter = { status, page, page_size }
  return (
    <AdminOrderFrame title={title}>
      <section className="bg-surface p-5 grid sm:grid-cols-2 lg:grid-cols-3 gap-4 items-end">
        <label className="space-y-2">
          <span>Trạng thái</span>
          <select className={inputClass} value={status} onChange={(event) => setParams(adminTailoringSearch({ ...filter, status: event.target.value, page: 1 }))}>
            <option value="">Tất cả</option>
            {adminTailoringStatuses.map((value) => (
              <option key={value} value={value}>
                {tailoringStatusLabel(value)}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-2">
          <span>Số yêu cầu mỗi trang</span>
          <select className={inputClass} value={page_size} onChange={(event) => setParams(adminTailoringSearch({ ...filter, page_size: Number(event.target.value), page: 1 }))}>
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
          <h2 className="font-headline-md text-primary text-2xl">Không có yêu cầu may đo ở bộ lọc/trang này</h2>
          {data.page > 1 && (
            <button type="button" className="text-primary underline" onClick={() => setParams(adminTailoringSearch({ ...filter, page: 1 }))}>
              Về trang đầu
            </button>
          )}
        </section>
      ) : (
        <ul className="grid xl:grid-cols-2 gap-5">
          {data.data.map((item) => (
            <li key={item.id} className="bg-surface border border-outline-variant p-5 sm:p-6 space-y-4 min-w-0">
              <div className="flex flex-wrap justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="font-title-editorial text-2xl text-primary break-words">
                    <Link to={`/admin/tailoring/${encodeURIComponent(item.id)}`} className="underline">
                      {item.full_name}
                    </Link>
                  </h2>
                  <p className="text-on-surface-variant">{item.phone}</p>
                </div>
                <span className="self-start bg-secondary-fixed/40 px-3 py-1 text-sm">{tailoringStatusLabel(item.status)}</span>
              </div>
              <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
                <dt className="text-on-surface-variant">Mẫu quan tâm</dt>
                <dd className="break-words">{item.product_slug || 'Chưa chọn mẫu'}</dd>
                <dt className="text-on-surface-variant">Ngày gửi</dt>
                <dd>
                  <time dateTime={item.created_at}>{formatOrderDate(item.created_at)}</time>
                </dd>
                <dt className="text-on-surface-variant">Lịch hẹn</dt>
                <dd>{item.appointment_at ? <time dateTime={item.appointment_at}>{formatOrderDate(item.appointment_at)}</time> : 'Chưa hẹn'}</dd>
                <dt className="text-on-surface-variant">Khách</dt>
                <dd>{item.is_guest ? 'Khách vãng lai' : 'Khách tài khoản'}</dd>
              </dl>
              <Link to={`/admin/tailoring/${encodeURIComponent(item.id)}`} className="inline-block border border-primary text-primary px-5 py-2">
                Xem và xử lý yêu cầu
              </Link>
            </li>
          ))}
        </ul>
      )}
      {data.total > 0 && (
        <nav aria-label="Phân trang yêu cầu may đo" className="flex flex-wrap justify-between items-center gap-4 border-t border-outline-variant pt-5">
          <p>
            {data.total} yêu cầu · Trang {data.page}/{pages} · {data.page_size}/trang
          </p>
          <div className="flex gap-3">
            <button type="button" className="border border-primary px-4 py-2 disabled:opacity-40" disabled={data.page <= 1} onClick={() => setParams(adminTailoringSearch({ ...filter, page: data.page - 1 }))}>
              Trang trước
            </button>
            <button type="button" className="border border-primary px-4 py-2 disabled:opacity-40" disabled={data.page >= pages || data.page >= 100000} onClick={() => setParams(adminTailoringSearch({ ...filter, page: data.page + 1 }))}>
              Trang sau
            </button>
          </div>
        </nav>
      )}
    </AdminOrderFrame>
  )
}
