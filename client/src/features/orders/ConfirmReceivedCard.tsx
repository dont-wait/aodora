import { useState } from 'react'
import { useAuth } from '@/features/auth/useAuth'
import { formatOrderMoney } from './money'
import { orderReceivable } from './presentation'
import { confirmOrderReceived } from './result'
import type { Order } from './types'

/** Buyer confirms they got the áo and paid; this also opens the 3-day return window. */
export default function ConfirmReceivedCard({ order, onDone }: { order: Order; onDone: () => void }) {
  const { user } = useAuth()
  const [asking, setAsking] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  if (!orderReceivable(order)) return null
  const counter = order.payment_method === 'cash_at_counter'
  async function confirm() {
    setBusy(true)
    setError('')
    try {
      await confirmOrderReceived(order.id, user?.id ?? null)
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Không xác nhận được.')
      setBusy(false)
    }
  }
  return (
    <section className="border border-primary/40 bg-surface-container-low p-5 sm:p-6 space-y-3" aria-labelledby="received-heading">
      <h2 id="received-heading" className="font-headline-md text-2xl text-primary">{counter ? 'Đã nhận áo tại quầy?' : 'Đã nhận được áo?'}</h2>
      {asking ? (
        <>
          <p>
            Bạn xác nhận đã nhận đủ áo và đã thanh toán <strong>{formatOrderMoney(order.total)}</strong>. Đơn sẽ chuyển sang <strong>Đã giao</strong>; nếu áo có vấn đề, bạn có 3 ngày để gửi yêu cầu hoàn hàng.
          </p>
          {error && <p role="alert" className="text-error">{error}</p>}
          <div className="flex flex-wrap gap-3">
            <button type="button" disabled={busy} className="bg-primary text-on-primary px-6 py-3 disabled:opacity-50" onClick={() => void confirm()}>
              {busy ? 'Đang xác nhận…' : 'Đúng, tôi đã nhận và thanh toán'}
            </button>
            <button type="button" disabled={busy} className="border border-primary text-primary px-6 py-3" onClick={() => setAsking(false)}>Chưa</button>
          </div>
        </>
      ) : (
        <>
          <p className="text-on-surface-variant">Khi đã nhận áo và trả tiền, hãy bấm xác nhận để hoàn tất đơn.</p>
          <button type="button" className="bg-primary text-on-primary px-6 py-3" onClick={() => setAsking(true)}>Đã nhận được hàng</button>
        </>
      )}
    </section>
  )
}
