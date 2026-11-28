import { useCallback, useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { formatMoney } from '@/features/catalog/prices'
import { formatOrderDate } from '@/features/orders/presentation'
import { adminReturnsApi, reasonLabels, returnStatusLabels, returnStatusTone, type AdminReturn, type ReturnStatus } from '@/features/returns/returns'
import { Alert, Button, DataTable, EmptyState, PageHeader, StatusBadge, TableSkeleton } from './ui'
import { inputClass } from './uiStyles'

const FILTERS: { value: '' | ReturnStatus; label: string }[] = [
  { value: 'requested', label: 'Chờ duyệt' },
  { value: 'approved', label: 'Chờ thu hồi' },
  { value: 'collecting', label: 'Đang thu hồi' },
  { value: 'collected', label: 'Đã thu hồi' },
  { value: 'refunded', label: 'Đã hoàn tiền' },
  { value: 'rejected', label: 'Từ chối' },
  { value: 'cancelled', label: 'Khách đã huỷ' },
  { value: '', label: 'Tất cả' },
]
const SIZE = 20

export default function AdminReturnsPage() {
  const [params, setParams] = useSearchParams()
  const status = (params.get('status') ?? 'requested') as '' | ReturnStatus
  const page = Math.max(1, Number(params.get('page')) || 1)
  const [state, setState] = useState<{ kind: 'loading' } | { kind: 'ready'; data: AdminReturn[]; total: number } | { kind: 'error'; message: string }>({ kind: 'loading' })
  const load = useCallback(() => {
    adminReturnsApi.list({ status, page, size: SIZE }).then(
      (res) => setState({ kind: 'ready', ...res }),
      (e: unknown) => setState({ kind: 'error', message: e instanceof Error ? e.message : 'Không tải được danh sách.' }),
    )
  }, [status, page])
  useEffect(load, [load])
  const pages = state.kind === 'ready' ? Math.max(1, Math.ceil(state.total / SIZE)) : 1
  const reload = () => { setState({ kind: 'loading' }); load() }
  const go = (next: { status?: string; page?: number }) => {
    setState({ kind: 'loading' })
    const p = new URLSearchParams()
    p.set('status', next.status ?? status)
    if ((next.page ?? 1) > 1) p.set('page', String(next.page))
    setParams(p)
  }

  return (
    <>
      <PageHeader title="Hoàn hàng" subtitle="Khách gửi yêu cầu trong 3 ngày sau khi nhận áo, kèm ảnh hoặc video. Xem minh chứng rồi duyệt, thu hồi, nhập kho và hoàn tiền." />
      <div className="flex flex-wrap items-end gap-3">
        <label className="space-y-1">
          <span className="text-sm">Trạng thái</span>
          <select className={inputClass} value={status} onChange={(e) => go({ status: e.target.value })}>
            {FILTERS.map((f) => <option key={f.value} value={f.value}>{f.label}</option>)}
          </select>
        </label>
        <Button variant="secondary" onClick={reload}>Tải lại</Button>
      </div>
      {state.kind === 'loading' && <TableSkeleton label="Đang tải yêu cầu hoàn hàng" />}
      {state.kind === 'error' && <Alert action={<Button variant="secondary" onClick={reload}>Thử lại</Button>}>{state.message}</Alert>}
      {state.kind === 'ready' && (state.data.length === 0 ? (
        <EmptyState title="Không có yêu cầu nào ở trạng thái này" />
      ) : (
        <DataTable label="Yêu cầu hoàn hàng">
          <thead className="sticky top-0 bg-surface-container-lowest">
            <tr className="border-b border-outline-variant/50">
              <th className="p-3">Đơn</th><th className="p-3">Khách</th><th className="p-3">Lý do</th><th className="p-3">Minh chứng</th><th className="p-3 text-right">Hoàn</th><th className="p-3">Gửi lúc</th><th className="p-3">Trạng thái</th>
            </tr>
          </thead>
          <tbody>
            {state.data.map((r) => (
              <tr key={r.id} className="border-b border-outline-variant/30">
                <td className="p-3"><Link className="text-primary underline font-semibold" to={`/admin/returns/${r.id}`}>{r.order_number}</Link></td>
                <td className="p-3">{r.customer_name}<br /><span className="text-xs text-on-surface-variant">{r.phone}</span></td>
                <td className="p-3">{reasonLabels[r.reason]}</td>
                <td className="p-3">{r.evidence.filter((e) => e.kind === 'image').length} ảnh · {r.evidence.filter((e) => e.kind === 'video').length} video</td>
                <td className="p-3 text-right tabular-nums">{formatMoney(r.refund_amount)} ₫</td>
                <td className="p-3 whitespace-nowrap"><time dateTime={r.created_at}>{formatOrderDate(r.created_at)}</time></td>
                <td className="p-3"><StatusBadge tone={returnStatusTone[r.status]}>{returnStatusLabels[r.status]}</StatusBadge></td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      ))}
      {state.kind === 'ready' && state.total > SIZE && (
        <nav aria-label="Phân trang" className="flex items-center justify-between">
          <p className="text-sm">{state.total} yêu cầu · Trang {page}/{pages}</p>
          <div className="flex gap-2">
            <Button variant="secondary" disabled={page <= 1} onClick={() => go({ page: page - 1 })}>Trang trước</Button>
            <Button variant="secondary" disabled={page >= pages} onClick={() => go({ page: page + 1 })}>Trang sau</Button>
          </div>
        </nav>
      )}
    </>
  )
}
