import { useRef, useState } from 'react'
import FieldErr from './FieldErr'
import { measurementKeys, measurementLabels, type AdminTailoring } from '@/features/admin/tailoring'
import { buildTailoringPatch, initialTailoringEdit, type TailoringEditValues, type TailoringMutation } from '@/features/admin/tailoringActions'
import type { TailoringPatch } from '@/features/admin/tailoring'

const input = 'w-full border border-outline-variant bg-surface p-3'
const serverFieldKey: Record<string, string> = { appointment_at: 'appointment' }

export default function AdminTailoringEditForm({
  tailoring, canEditMeasurements, busy, onSave,
}: {
  tailoring: AdminTailoring
  canEditMeasurements: boolean
  busy: boolean
  onSave: (patch: TailoringPatch) => Promise<TailoringMutation>
}) {
  // Remounted (and so reset) by the parent only when the saved DTO changes; failures keep every typed value.
  const [values, setValues] = useState<TailoringEditValues>(() => initialTailoringEdit(tailoring))
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [message, setMessage] = useState('')
  const sending = useRef(false)
  const set = <K extends keyof TailoringEditValues>(key: K, value: TailoringEditValues[K]) => {
    setValues((previous) => ({ ...previous, [key]: value }))
    setErrors((previous) => (previous[key as string] ? { ...previous, [key as string]: '' } : previous))
  }
  const setMeasurement = (key: (typeof measurementKeys)[number], value: string) => {
    setValues((previous) => ({ ...previous, measurements: { ...previous.measurements, [key]: value } }))
    setErrors((previous) => (previous[key] ? { ...previous, [key]: '' } : previous))
  }
  const setCleared = (key: (typeof measurementKeys)[number] | 'body', cleared: boolean) =>
    setValues((previous) => ({ ...previous, cleared: { ...previous.cleared, [key]: cleared } }))

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (sending.current || busy) return
    setMessage('')
    const { patch, errors: local } = buildTailoringPatch(tailoring, values, canEditMeasurements)
    setErrors(local)
    if (Object.keys(local).length > 0) { setMessage('Vui lòng kiểm tra lại các trường được đánh dấu.'); return }
    if (Object.keys(patch).length === 0) { setMessage('Chưa có thay đổi nào để lưu.'); return }
    sending.current = true
    try {
      const result = await onSave(patch)
      if (result.kind === 'invalid') {
        if (result.field) setErrors({ [serverFieldKey[result.field] || result.field]: result.message })
        setMessage(`${result.message} Chưa lưu; nội dung bạn đang sửa vẫn được giữ.`)
      } else if (result.kind === 'error') setMessage(`${result.message} Nội dung bạn đang sửa vẫn được giữ.`)
    } finally {
      sending.current = false
    }
  }
  return (
    <form onSubmit={submit} className="space-y-5" noValidate>
      <div className="grid sm:grid-cols-2 gap-4">
        <label className="space-y-1 block">
          <span>Người phụ trách</span>
          <input className={input} value={values.assigned_staff} maxLength={200} onChange={(e) => set('assigned_staff', e.target.value)} />
          <FieldErr message={errors.assigned_staff} />
        </label>
        <label className="space-y-1 block">
          <span>Kênh liên hệ ưu tiên</span>
          <select className={input} value={values.preferred_contact} onChange={(e) => set('preferred_contact', e.target.value as TailoringEditValues['preferred_contact'])}>
            <option value="">Không chọn</option>
            <option value="phone">Điện thoại</option>
            <option value="zalo">Zalo</option>
          </select>
          <FieldErr message={errors.preferred_contact} />
        </label>
        <label className="space-y-1 block sm:col-span-2">
          <span>Lịch hẹn tư vấn (giờ trình duyệt; để trống để xóa lịch hẹn)</span>
          <input type="datetime-local" className={input} value={values.appointment} onChange={(e) => set('appointment', e.target.value)} />
          <FieldErr message={errors.appointment} />
        </label>
        <label className="space-y-1 block sm:col-span-2">
          <span>Ghi chú nội bộ / yêu cầu của khách (để trống để xóa)</span>
          <textarea className={input} rows={3} value={values.notes} onChange={(e) => set('notes', e.target.value)} />
          <FieldErr message={errors.notes} />
        </label>
      </div>
      {canEditMeasurements ? (
        <fieldset className="space-y-3">
          <legend className="font-semibold">Số đo (cm)</legend>
          <p className="text-sm text-on-surface-variant">Ô trống = giữ nguyên. Muốn xóa một số đo, bấm &quot;Xóa số đo&quot; riêng của ô đó.</p>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {measurementKeys.map((key) => (
              <div key={key} className="min-w-0 space-y-1">
                <label htmlFor={`e-${key}`}>
                  {measurementLabels[key]} <span className="text-on-surface-variant text-sm">(hiện: {tailoring.measurements?.[key] ?? 'chưa có'})</span>
                </label>
                <div className="flex gap-2">
                  <input id={`e-${key}`} inputMode="decimal" className={`${input} min-w-0`} disabled={values.cleared[key]} placeholder={values.cleared[key] ? 'Sẽ xóa khi lưu' : 'Nhập số đo mới'} value={values.measurements[key]} onChange={(e) => setMeasurement(key, e.target.value)} />
                  {tailoring.measurements?.[key] !== undefined && (
                    <button type="button" className="border border-outline-variant px-3 shrink-0" onClick={() => setCleared(key, !values.cleared[key])}>
                      {values.cleared[key] ? 'Giữ lại' : 'Xóa số đo'}
                    </button>
                  )}
                </div>
                <FieldErr message={errors[key]} />
              </div>
            ))}
            <div className="min-w-0 space-y-1 sm:col-span-2">
              <label htmlFor="e-body">Mô tả vóc dáng <span className="text-on-surface-variant text-sm">(hiện: {tailoring.measurements?.body ?? 'chưa có'})</span></label>
              <div className="flex gap-2">
                <input id="e-body" className={`${input} min-w-0`} disabled={values.cleared.body} value={values.body} placeholder={values.cleared.body ? 'Sẽ xóa khi lưu' : 'Ví dụ: 1m62 / 50kg'} onChange={(e) => set('body', e.target.value)} />
                {tailoring.measurements?.body !== undefined && (
                  <button type="button" className="border border-outline-variant px-3 shrink-0" onClick={() => setCleared('body', !values.cleared.body)}>
                    {values.cleared.body ? 'Giữ lại' : 'Xóa mô tả'}
                  </button>
                )}
              </div>
              <FieldErr message={errors.body} />
            </div>
          </div>
        </fieldset>
      ) : (
        <p className="bg-surface-container p-4">Đã chốt số đo, không sửa trực tiếp. (Hiện chưa có luồng mở lại số đo.)</p>
      )}
      {message && <p role="alert" className="text-error">{message}</p>}
      <button type="submit" disabled={busy} className="bg-primary text-on-primary px-6 py-3 disabled:opacity-40">
        {busy ? 'Đang lưu...' : 'Lưu thay đổi'}
      </button>
    </form>
  )
}
