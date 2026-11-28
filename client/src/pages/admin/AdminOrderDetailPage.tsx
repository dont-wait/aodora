import { useCallback, useRef, useState } from 'react'
import { useLocation, useParams } from 'react-router-dom'
import Modal from '@/components/ui/Modal'
import { useAuth } from '@/features/auth/useAuth'
import {
  adminOrderActionGroups,
  adminOrderConfirmation,
  inventoryRestockLabel,
  loadAdminOrder,
  performAdminOrderAction,
  type AdminOrderAction,
  type AdminOrderActionResult,
  type AdminOrderRead,
} from '@/features/admin/orderActions'
import type { AdminOrder } from '@/features/admin/orders'
import { formatOrderDate } from '@/features/orders/presentation'
import { useOrderRead } from '@/features/orders/useOrderRead'
import OrderSummary from '@/pages/checkout/OrderSummary'
import AdminOrderFrame from './AdminOrderFrame'
import AdminOrderNotice from './AdminOrderNotice'

const buttonClass = 'border border-primary text-primary px-4 py-3 disabled:opacity-40 disabled:cursor-not-allowed'
const timeline = [
  ['updated_at', 'Cập nhật gần nhất'],
  ['cancelled_at', 'Hủy đơn'],
  ['inventory_restocked_at', 'Nhập lại hàng'],
  ['paid_at', 'Ghi nhận thu đủ tiền'],
  ['refunded_at', 'Ghi nhận hoàn đủ tiền'],
] as const
export default function AdminOrderDetailPage() {
  const { id = '' } = useParams()
  const location = useLocation()
  const { user, loading, error } = useAuth()
  const [revision, setRevision] = useState(0)
  const load = useCallback(() => loadAdminOrder(id), [id])
  const scope = JSON.stringify([id, user?.id, revision])
  const read = useOrderRead(scope, !!user && !loading && !error, load)
  const [override, setOverride] = useState<{ source: AdminOrderRead; outcome: AdminOrderActionResult } | null>(null)
  const [selected, setSelected] = useState<{ source: AdminOrderRead; order: AdminOrder; action: AdminOrderAction } | null>(null)
  const [busy, setBusy] = useState(false)
  const lock = useRef(false)
  const retry = () => {
    if (!lock.current) setRevision((value) => value + 1)
  }
  const next = location.pathname + location.search
  if (!read || read.kind !== 'ready') return <AdminOrderNotice result={read} next={next} retry={retry} />
  const outcome = override?.source === read ? override.outcome : null
  if (outcome && outcome.kind !== 'saved' && outcome.kind !== 'conflict') return <AdminOrderNotice result={outcome} next={next} retry={retry} />
  const order = outcome?.order ?? read.order
  const groups = adminOrderActionGroups(order)
  const dialog = selected?.source === read && selected.order === order ? selected : null
  const picked = dialog ? [...groups.delivery, ...groups.payment].find((action) => action.id === dialog.action) : null

  async function confirm() {
    if (!dialog || !picked || lock.current) return
    lock.current = true
    setBusy(true)
    try {
      setOverride({ source: dialog.source, outcome: await performAdminOrderAction(dialog.order, dialog.action) })
    } finally {
      lock.current = false
      setBusy(false)
      setSelected(null)
    }
  }
  return (
    <AdminOrderFrame title={`Đơn ${order.order_number}`} crumb={order.order_number}>
      <div className="flex flex-wrap justify-between gap-4">
        <p className="text-on-surface-variant">{order.is_guest ? 'Khách vãng lai' : 'Khách tài khoản'} · Thao tác giao nhận và thu tiền độc lập.</p>
        <button type="button" disabled={busy} className={buttonClass} onClick={retry}>
          Tải lại đơn
        </button>
      </div>
      {outcome?.kind === 'conflict' && (
        <p role="alert" className="border border-error bg-error-container text-on-error-container p-4">
          {outcome.message}
        </p>
      )}
      {outcome?.kind === 'saved' && (
        <p role="status" className="bg-secondary-fixed/30 p-4 text-primary">
          Đã lưu thay đổi.
        </p>
      )}
      <div className="grid lg:grid-cols-[1.2fr_1fr] gap-7 items-start">
        <section className="bg-surface p-5 sm:p-7">
          <OrderSummary order={order} />
        </section>
        <aside className="space-y-5">
          <section className="bg-surface p-5 sm:p-6 space-y-4">
            <h2 className="font-headline-md text-2xl text-primary">Giao nhận</h2>
            <div className="flex flex-wrap gap-3">
              {groups.delivery.map((action) => (
                <button
                  type="button"
                  key={action.id}
                  disabled={busy}
                  className={buttonClass}
                  onClick={() => setSelected({ source: read, order, action: action.id })}
                >
                  {action.label}
                </button>
              ))}
              {!groups.delivery.length && <p className="text-on-surface-variant">Hiện không có thao tác giao nhận được phép.</p>}
            </div>
          </section>
          <section className="bg-surface p-5 sm:p-6 space-y-4">
            <h2 className="font-headline-md text-2xl text-primary">Thanh toán</h2>
            <p className="text-sm text-on-surface-variant">Chỉ ghi nhận khi bạn đã thực tế thu/hoàn đủ tiền. Không tự chuyển tiền.</p>
            <div className="flex flex-wrap gap-3">
              {groups.payment.map((action) => (
                <button
                  type="button"
                  key={action.id}
                  disabled={busy}
                  className={buttonClass}
                  onClick={() => setSelected({ source: read, order, action: action.id })}
                >
                  {action.label}
                </button>
              ))}
              {!groups.payment.length && <p className="text-on-surface-variant">Hiện không có thao tác thanh toán được phép.</p>}
            </div>
          </section>
          <section className="bg-surface p-5 sm:p-6 space-y-4">
            <h2 className="font-headline-md text-2xl text-primary">Hoàn tồn và các mốc xử lý</h2>
            <p>
              Hoàn tồn: <strong>{inventoryRestockLabel(order.inventory_restock)}</strong>
            </p>
            <dl className="space-y-3">
              {timeline.map(([field, label]) =>
                order[field] ? (
                  <div key={field}>
                    <dt className="text-sm text-on-surface-variant">{label}</dt>
                    <dd>
                      <time dateTime={order[field]}>{formatOrderDate(order[field]!)}</time>
                    </dd>
                  </div>
                ) : null,
              )}
            </dl>
          </section>
        </aside>
      </div>
      <Modal
        open={!!dialog && !!picked}
        onClose={() => {
          if (!busy) setSelected(null)
        }}
      >
        {dialog && picked && (
          <div className="space-y-5">
            <h2 className="font-headline-md text-2xl text-primary">
              {picked.label} · {dialog.order.order_number}
            </h2>
            <p>{adminOrderConfirmation(dialog.order, dialog.action)}</p>
            <div className="flex flex-wrap gap-3">
              <button type="button" disabled={busy} className="bg-primary text-on-primary px-5 py-3 disabled:opacity-40" onClick={() => void confirm()}>
                {busy ? 'Đang gửi...' : 'Xác nhận thao tác'}
              </button>
              <button type="button" disabled={busy} className={buttonClass} onClick={() => setSelected(null)}>
                Quay lại
              </button>
            </div>
          </div>
        )}
      </Modal>
    </AdminOrderFrame>
  )
}
