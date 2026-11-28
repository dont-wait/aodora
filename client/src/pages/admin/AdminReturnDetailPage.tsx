import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { formatMoney } from '@/features/catalog/prices'
import { formatOrderDate } from '@/features/orders/presentation'
import { EvidenceStrip } from '@/features/returns/ReturnRequestSection'
import { adminActionLabels, adminReturnsApi, reasonLabels, returnStatusLabels, returnStatusTone, timeLeft, type AdminReturn, type ReturnStatus } from '@/features/returns/returns'
import { useToast } from '@/hooks/useToast'
import { Alert, AutoTextarea, Button, ButtonLink, Card, CardSkeleton, PageHeader, StatusBadge } from './ui'
import { useAdminCrumb } from './breadcrumb'

const hints: Partial<Record<ReturnStatus, string>> = {
  approved: 'Khách sẽ thấy yêu cầu đã được duyệt và chờ cửa hàng thu hồi áo.',
  rejected: 'Bắt buộc ghi lý do từ chối, khách sẽ đọc được ghi chú này.',
  collecting: 'Ghi chú cách thu hồi nếu cần (ví dụ: shipper đến lấy chiều nay).',
  collected: 'Kiểm tra áo đã nhận. Chọn "nhập lại kho" nếu áo còn bán được.',
  refunded: 'Chỉ xác nhận sau khi đã chuyển/trả tiền cho khách. Nếu hoàn toàn bộ đơn, trạng thái thanh toán của đơn chuyển sang "Đã hoàn tiền".',
}

export default function AdminReturnDetailPage() {
  const { id = '' } = useParams()
  const { showToast } = useToast()
  const [state, setState] = useState<{ kind: 'loading' } | { kind: 'ready'; data: AdminReturn } | { kind: 'error'; message: string }>({ kind: 'loading' })
  const [action, setAction] = useState<ReturnStatus | null>(null)
  const [note, setNote] = useState('')
  const [restock, setRestock] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  useAdminCrumb(state.kind === 'ready' ? `Đơn ${state.data.order_number}` : null)
  const load = useCallback(() => {
    adminReturnsApi.get(id).then(
      (data) => setState({ kind: 'ready', data }),
      (e: unknown) => setState({ kind: 'error', message: e instanceof Error ? e.message : 'Không tải được yêu cầu.' }),
    )
  }, [id])
  useEffect(load, [load])

  if (state.kind === 'loading') return <CardSkeleton label="Đang tải yêu cầu hoàn hàng" />
  if (state.kind === 'error') return <Alert action={<Button variant="secondary" onClick={load}>Thử lại</Button>}>{state.message}</Alert>
  const r = state.data
  const deadline = new Date(Date.parse(r.delivered_at) + 72 * 3600_000).toISOString()
  const onTime = Date.parse(r.created_at) <= Date.parse(deadline)

  async function submit() {
    if (!action || busy) return
    setBusy(true)
    setError('')
    try {
      const data = await adminReturnsApi.transition(r.id, { to: action, note, restock: action === 'collected' ? restock : false })
      setState({ kind: 'ready', data })
      showToast(`Đã chuyển sang "${returnStatusLabels[action]}".`)
      setAction(null)
      setNote('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Không cập nhật được.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <PageHeader
        title={`Hoàn hàng · đơn ${r.order_number}`}
        subtitle={`${r.customer_name ?? ''} · ${r.phone ?? ''}`}
        actions={<div className="flex gap-2"><ButtonLink to={`/admin/orders/${r.order_id}`}>Xem đơn hàng</ButtonLink><ButtonLink to="/admin/returns">Danh sách</ButtonLink></div>}
      />
      <div className="grid gap-5 lg:grid-cols-[2fr_1fr]">
        <div className="space-y-5 min-w-0">
          <Card title="Minh chứng của khách" description="Video phát ngay tại đây; bấm ảnh để mở cỡ lớn.">
            <EvidenceStrip evidence={r.evidence} large />
          </Card>
          <Card title="Nội dung yêu cầu">
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
              <dt className="text-on-surface-variant">Lý do</dt><dd>{reasonLabels[r.reason]}</dd>
              <dt className="text-on-surface-variant">Mô tả</dt><dd className="whitespace-pre-line break-words">{r.note || '—'}</dd>
              <dt className="text-on-surface-variant">Nhận áo lúc</dt><dd>{formatOrderDate(r.delivered_at)}</dd>
              <dt className="text-on-surface-variant">Gửi yêu cầu</dt>
              <dd>{formatOrderDate(r.created_at)} {onTime ? <StatusBadge tone="success">Trong hạn 3 ngày</StatusBadge> : <StatusBadge tone="danger">Quá hạn</StatusBadge>}</dd>
            </dl>
            <ul className="divide-y divide-outline-variant/40 text-sm mt-3">
              {r.items.map((it) => (
                <li key={it.sku} className="flex justify-between gap-3 py-2">
                  <span>{it.product_name} <span className="text-on-surface-variant">{[it.size, it.color, it.sku].filter(Boolean).join(' · ')}</span> × {it.quantity}</span>
                  <span className="tabular-nums">{formatMoney(it.unit_price * it.quantity)} ₫</span>
                </li>
              ))}
              <li className="flex justify-between py-2 font-semibold"><span>Số tiền hoàn</span><span className="tabular-nums">{formatMoney(r.refund_amount)} ₫</span></li>
            </ul>
          </Card>
        </div>
        <div className="space-y-5">
          <Card title="Trạng thái">
            <StatusBadge tone={returnStatusTone[r.status]}>{returnStatusLabels[r.status]}</StatusBadge>
            {r.status === 'requested' && timeLeft(deadline) && <p className="text-xs text-on-surface-variant mt-2">Hạn khách gửi: {timeLeft(deadline)}</p>}
            {r.restocked && <p className="text-sm mt-2">Đã nhập lại kho.</p>}
            {r.allowed_next.length > 0 ? (
              <div className="space-y-3 mt-4">
                <div className="flex flex-wrap gap-2">
                  {r.allowed_next.map((to) => (
                    <Button key={to} variant={to === 'rejected' ? 'danger' : action === to ? 'primary' : 'secondary'} onClick={() => { setAction(to); setError('') }}>
                      {adminActionLabels[to]}
                    </Button>
                  ))}
                </div>
                {action && (
                  <div className="space-y-3 border-t border-outline-variant/40 pt-3">
                    <p className="text-sm text-on-surface-variant">{hints[action]}</p>
                    <AutoTextarea value={note} onChange={(e) => setNote(e.target.value)} placeholder={action === 'rejected' ? 'Lý do từ chối (bắt buộc)' : 'Ghi chú cho khách (không bắt buộc)'} maxLength={1000} />
                    {action === 'collected' && (
                      <label className="flex items-center gap-2 text-sm">
                        <input type="checkbox" checked={restock} onChange={(e) => setRestock(e.target.checked)} />
                        Nhập lại kho {r.items.reduce((s, it) => s + it.quantity, 0)} áo (bỏ chọn nếu áo hỏng, không bán lại được)
                      </label>
                    )}
                    {error && <p role="alert" className="text-sm text-red-700">{error}</p>}
                    <div className="flex gap-2">
                      <Button loading={busy} variant={action === 'rejected' ? 'danger' : 'primary'} onClick={() => void submit()}>Xác nhận: {adminActionLabels[action]}</Button>
                      <Button variant="ghost" disabled={busy} onClick={() => setAction(null)}>Huỷ</Button>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <p className="text-sm text-on-surface-variant mt-3">Yêu cầu đã kết thúc.</p>
            )}
          </Card>
          <Card title="Lịch sử">
            <ol className="space-y-3 text-sm">
              {r.history.map((h, i) => (
                <li key={i} className="border-l-2 border-primary/40 pl-3">
                  <p className="font-semibold">{returnStatusLabels[h.status]}</p>
                  <p className="text-xs text-on-surface-variant">{formatOrderDate(h.at)} · {h.by === 'admin' ? 'Cửa hàng' : 'Khách'}</p>
                  {h.note && <p className="whitespace-pre-line break-words">{h.note}</p>}
                </li>
              ))}
            </ol>
          </Card>
          <p className="text-xs"><Link className="underline" to={`/admin/returns?status=`}>Xem mọi yêu cầu</Link></p>
        </div>
      </div>
    </>
  )
}
