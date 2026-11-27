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
import { Button, ButtonLink, DataTable, EmptyState, StatusBadge } from './ui'
import { inputClass, tdClass, thClass, toneForStatus } from './uiStyles'

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
  const sizeOptions = [...new Set([20, 50, 100, page_size])].sort((a, b) => a - b)
  return (
    <AdminOrderFrame title={title}>
      <section className="bg-surface border border-outline-variant/60 p-4 flex flex-wrap items-end gap-3">
        <label className="space-y-1 text-sm">
          <span className="block font-medium">Trạng thái</span>
          <select className={`${inputClass} w-auto min-w-48`} value={status} onChange={(event) => setParams(adminTailoringSearch({ ...filter, status: event.target.value, page: 1 }))}>
            <option value="">Tất cả</option>
            {adminTailoringStatuses.map((value) => <option key={value} value={value}>{tailoringStatusLabel(value)}</option>)}
          </select>
        </label>
        <label className="space-y-1 text-sm">
          <span className="block font-medium">Yêu cầu/trang</span>
          <select className={`${inputClass} w-auto`} value={page_size} onChange={(event) => setParams(adminTailoringSearch({ ...filter, page_size: Number(event.target.value), page: 1 }))}>
            {sizeOptions.map((size) => <option key={size} value={size}>{size}</option>)}
          </select>
        </label>
        <Button variant="secondary" onClick={retry}>Tải lại</Button>
      </section>
      {data.data.length === 0 ? (
        <EmptyState title="Không có yêu cầu may đo ở bộ lọc/trang này">
          {data.page > 1 && <Button variant="ghost" onClick={() => setParams(adminTailoringSearch({ ...filter, page: 1 }))}>Về trang đầu</Button>}
        </EmptyState>
      ) : (
        <DataTable label="Danh sách yêu cầu may đo">
          <thead>
            <tr>
              <th className={thClass}>Khách</th><th className={thClass}>Mẫu quan tâm</th><th className={thClass}>Ngày gửi</th>
              <th className={thClass}>Lịch hẹn</th><th className={thClass}>Trạng thái</th><th className={thClass}><span className="sr-only">Thao tác</span></th>
            </tr>
          </thead>
          <tbody>
            {data.data.map((item) => (
              <tr key={item.id}>
                <td className={tdClass}>
                  <Link to={`/admin/tailoring/${encodeURIComponent(item.id)}`} className="font-semibold text-primary underline break-words">{item.full_name}</Link>
                  <p className="text-xs text-on-surface-variant">{item.phone} · {item.is_guest ? 'Khách vãng lai' : 'Khách tài khoản'}</p>
                </td>
                <td className={`${tdClass} break-words`}>{item.product_slug || 'Chưa chọn mẫu'}</td>
                <td className={`${tdClass} whitespace-nowrap`}><time dateTime={item.created_at}>{formatOrderDate(item.created_at)}</time></td>
                <td className={`${tdClass} whitespace-nowrap`}>{item.appointment_at ? <time dateTime={item.appointment_at}>{formatOrderDate(item.appointment_at)}</time> : 'Chưa hẹn'}</td>
                <td className={tdClass}><StatusBadge tone={toneForStatus(item.status)}>{tailoringStatusLabel(item.status)}</StatusBadge></td>
                <td className={tdClass}><ButtonLink to={`/admin/tailoring/${encodeURIComponent(item.id)}`}>Xử lý</ButtonLink></td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
      {data.total > 0 && (
        <nav aria-label="Phân trang yêu cầu may đo" className="flex flex-wrap justify-between items-center gap-4 text-sm">
          <p>{data.total} yêu cầu · Trang {data.page}/{pages} · {data.page_size}/trang</p>
          <div className="flex gap-2">
            <Button variant="secondary" disabled={data.page <= 1} onClick={() => setParams(adminTailoringSearch({ ...filter, page: data.page - 1 }))}>Trang trước</Button>
            <Button variant="secondary" disabled={data.page >= pages || data.page >= 100000} onClick={() => setParams(adminTailoringSearch({ ...filter, page: data.page + 1 }))}>Trang sau</Button>
          </div>
        </nav>
      )}
    </AdminOrderFrame>
  )
}
