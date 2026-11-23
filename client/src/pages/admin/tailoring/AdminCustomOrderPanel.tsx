import { useRef, useState } from 'react'
import FieldErr from './FieldErr'
import { Link } from 'react-router-dom'
import type { AdminTailoring } from '@/features/admin/tailoring'
import {
  createAdminCustomOrder, initialCustomOrder, paymentMethodLabels, validateCustomOrder,
  type CustomOrderOutcome, type CustomOrderValues, type TailoringFailure,
} from '@/features/admin/tailoringActions'

const input = 'w-full border border-outline-variant bg-surface p-3'

export default function AdminCustomOrderPanel({
  tailoring, onRefreshed, onFailure,
}: {
  tailoring: AdminTailoring
  onRefreshed: (latest: AdminTailoring, message: string) => void
  onFailure: (outcome: TailoringFailure) => void
}) {
  const [values, setValues] = useState<CustomOrderValues>(() => initialCustomOrder(tailoring))
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [outcome, setOutcome] = useState<CustomOrderOutcome | null>(null)
  const [busy, setBusy] = useState(false)
  const lock = useRef(false)
  const set = (key: keyof CustomOrderValues, value: string) => {
    setValues((previous) => ({ ...previous, [key]: value }))
    setErrors((previous) => (previous[key] ? { ...previous, [key]: '' } : previous))
  }
  const placed = outcome?.kind === 'created' || outcome?.kind === 'exists' ? outcome : null
  const done = placed !== null
  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (lock.current || done) return
    setOutcome(null)
    const { body, errors: local } = validateCustomOrder(tailoring, values)
    setErrors(local)
    if (!body) return
    lock.current = true
    setBusy(true)
    try {
      const result = await createAdminCustomOrder(tailoring, body)
      if (result.kind === 'unauthenticated' || result.kind === 'forbidden' || result.kind === 'not-found') return onFailure(result)
      if (result.kind === 'conflict') return onRefreshed(result.tailoring, result.message)
      if (result.kind === 'invalid' && result.field) setErrors({ [result.field.replace('shipping_address.', '')]: result.message })
      setOutcome(result)
    } finally {
      lock.current = false
      setBusy(false)
    }
  }
  const cod = values.payment_method === 'cod'
  return (
    <section className="bg-surface p-5 sm:p-7 space-y-5" aria-labelledby="custom-order-title">
      <h2 id="custom-order-title" className="font-headline-md text-2xl text-primary">Tạo đơn may đo</h2>
      <ul className="text-sm text-on-surface-variant list-disc pl-5 space-y-1">
        <li>Đơn may đo <strong>không trừ tồn kho</strong>; giá do bạn chốt tại đây, không lấy từ sản phẩm.</li>
        <li>Phí COD lấy từ cấu hình máy chủ; chưa cấu hình thì không tạo được đơn COD. Thu tại quầy không có phí.</li>
        <li>Mỗi yêu cầu chỉ có một đơn. Gửi lại cùng nội dung sẽ trả đơn đã có, không tạo đơn thứ hai.</li>
        {tailoring.is_guest && <li><strong>Khách vãng lai: chỉ quản trị đọc được đơn này</strong> (khách không có mã truy cập).</li>}
      </ul>
      {placed ? (
        <div role="status" className="bg-secondary-fixed/30 p-4 space-y-2">
          <p>{placed.kind === 'created' ? 'Đã tạo đơn may đo.' : 'Yêu cầu này đã có đơn (nội dung trùng khớp), không tạo thêm.'}</p>
          <Link className="underline text-primary" to={`/admin/orders/${encodeURIComponent(placed.result.orderId)}`}>
            Mở đơn {placed.result.orderNumber || placed.result.orderId}
          </Link>
        </div>
      ) : (
        <form onSubmit={submit} className="space-y-4" noValidate>
          <div className="grid sm:grid-cols-2 gap-4">
            <label className="space-y-1 block">
              <span>Đơn giá chốt (₫) *</span>
              <input className={input} inputMode="decimal" value={values.unit_price} placeholder="1500000" onChange={(e) => set('unit_price', e.target.value)} />
              <FieldErr message={errors.unit_price} />
            </label>
            <label className="space-y-1 block">
              <span>Số lượng</span>
              <input className={input} inputMode="numeric" value={values.quantity} onChange={(e) => set('quantity', e.target.value)} />
              <FieldErr message={errors.quantity} />
            </label>
            <label className="space-y-1 block sm:col-span-2">
              <span>Hình thức thu tiền *</span>
              <select className={input} value={values.payment_method} onChange={(e) => set('payment_method', e.target.value)}>
                <option value="">Chọn hình thức</option>
                {(Object.keys(paymentMethodLabels) as (keyof typeof paymentMethodLabels)[]).map((key) => (
                  <option key={key} value={key}>{paymentMethodLabels[key]}</option>
                ))}
              </select>
              <FieldErr message={errors.payment_method} />
            </label>
            {!tailoring.product_id && (
              <label className="space-y-1 block sm:col-span-2">
                <span>Mã sản phẩm mẫu * (yêu cầu chưa gắn mẫu)</span>
                <input className={`${input} font-mono`} value={values.product_id} maxLength={24} onChange={(e) => set('product_id', e.target.value)} />
                <FieldErr message={errors.product_id} />
              </label>
            )}
            <label className="space-y-1 block">
              <span>Người nhận *</span>
              <input className={input} value={values.recipient} onChange={(e) => set('recipient', e.target.value)} />
              <FieldErr message={errors.recipient} />
            </label>
            <label className="space-y-1 block">
              <span>Điện thoại người nhận *</span>
              <input className={input} value={values.phone} onChange={(e) => set('phone', e.target.value)} />
              <FieldErr message={errors.phone} />
            </label>
            {cod && (
              <>
                <label className="space-y-1 block sm:col-span-2">
                  <span>Địa chỉ giao (số nhà, đường) *</span>
                  <input className={input} value={values.line1} onChange={(e) => set('line1', e.target.value)} />
                  <FieldErr message={errors.line1} />
                </label>
                <label className="space-y-1 block"><span>Phường/xã</span><input className={input} value={values.ward} onChange={(e) => set('ward', e.target.value)} /></label>
                <label className="space-y-1 block"><span>Quận/huyện</span><input className={input} value={values.district} onChange={(e) => set('district', e.target.value)} /></label>
                <label className="space-y-1 block sm:col-span-2">
                  <span>Tỉnh/thành phố *</span>
                  <input className={input} value={values.city} onChange={(e) => set('city', e.target.value)} />
                  <FieldErr message={errors.city} />
                </label>
              </>
            )}
            <label className="space-y-1 block sm:col-span-2">
              <span>Ghi chú đơn</span>
              <textarea className={input} rows={2} value={values.note} onChange={(e) => set('note', e.target.value)} />
              <FieldErr message={errors.note} />
            </label>
          </div>
          {outcome && 'message' in outcome && (
            <p role="alert" className="border border-error bg-error-container text-on-error-container p-4">{outcome.message}</p>
          )}
          <button type="submit" disabled={busy} className="bg-primary text-on-primary px-6 py-3 disabled:opacity-40">
            {busy ? 'Đang tạo đơn...' : 'Tạo đơn may đo'}
          </button>
        </form>
      )}
    </section>
  )
}
