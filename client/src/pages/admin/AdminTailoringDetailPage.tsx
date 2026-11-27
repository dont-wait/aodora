import { useCallback, useRef, useState } from 'react'
import { useLocation, useParams } from 'react-router-dom'
import Modal from '@/components/ui/Modal'
import { useAuth } from '@/features/auth/useAuth'
import { measurementKeys, measurementLabels, tailoringStatusLabel, type AdminTailoring, type TailoringPatch } from '@/features/admin/tailoring'
import {
  adminTailoringActionButtons, adminTailoringConfirmation, confirmAdminMeasurements, confirmMeasurementsCheckbox, confirmMeasurementsText,
  loadAdminTailoring, orderablePhase, performAdminTailoringAction, saveAdminTailoring,
  type TailoringAction, type TailoringMutation, type TailoringRead,
} from '@/features/admin/tailoringActions'
import { formatOrderDate } from '@/features/orders/presentation'
import { useOrderRead } from '@/features/orders/useOrderRead'
import AdminOrderFrame from './AdminOrderFrame'
import AdminOrderNotice from './AdminOrderNotice'
import AdminCustomOrderPanel from './tailoring/AdminCustomOrderPanel'
import AdminTailoringEditForm from './tailoring/AdminTailoringEditForm'

const buttonClass = 'border border-primary text-primary px-4 py-3 disabled:opacity-40 disabled:cursor-not-allowed'
type Pending = { source: TailoringRead; tailoring: AdminTailoring; action: TailoringAction | 'confirm_measurements' }
type Override = { source: TailoringRead; outcome: TailoringMutation | { kind: 'refreshed'; tailoring: AdminTailoring; message: string } }

export default function AdminTailoringDetailPage() {
  const { id = '' } = useParams()
  const location = useLocation()
  const { user, loading, error } = useAuth()
  const [revision, setRevision] = useState(0)
  const load = useCallback(() => loadAdminTailoring(id), [id])
  const scope = JSON.stringify([id, user?.id, revision])
  const read = useOrderRead(scope, !!user && !loading && !error, load)
  const [override, setOverride] = useState<Override | null>(null)
  const [selected, setSelected] = useState<Pending | null>(null)
  const [checked, setChecked] = useState(false)
  const [busy, setBusy] = useState(false)
  const lock = useRef(false)
  const retry = () => {
    if (!lock.current) setRevision((value) => value + 1)
  }
  const next = location.pathname + location.search
  const title = 'Yêu cầu may đo'
  if (!read || read.kind !== 'ready') return <AdminOrderNotice title={title} result={read} next={next} retry={retry} />
  const outcome = override?.source === read ? override.outcome : null
  if (outcome && (outcome.kind === 'unauthenticated' || outcome.kind === 'forbidden' || outcome.kind === 'not-found' || outcome.kind === 'error')) {
    return <AdminOrderNotice title={title} result={outcome} next={next} retry={retry} />
  }
  const shown = outcome && (outcome.kind === 'saved' || outcome.kind === 'conflict' || outcome.kind === 'refreshed') ? outcome.tailoring : read.tailoring
  const t = shown
  const buttons = adminTailoringActionButtons(t)
  const dialog = selected?.source === read && selected.tailoring === t ? selected : null
  const statusSpec = dialog && dialog.action !== 'confirm_measurements' ? buttons.status.find((a) => a.id === dialog.action) : null

  // One lock for every mutation on this page (status, confirm, save): no double submit.
  async function run(task: () => Promise<TailoringMutation>, inlineErrors = false): Promise<TailoringMutation> {
    if (lock.current) return { kind: 'error', message: 'Đang xử lý thao tác trước, vui lòng đợi.' }
    lock.current = true
    setBusy(true)
    try {
      const result = await task()
      // inlineErrors: a save failure stays in the edit form so nothing typed is lost.
      if (result.kind !== 'invalid' && !(inlineErrors && result.kind === 'error')) setOverride({ source: read as TailoringRead, outcome: result })
      return result
    } finally {
      lock.current = false
      setBusy(false)
    }
  }
  async function confirmDialog() {
    if (!dialog) return
    if (dialog.action === 'confirm_measurements') {
      if (!checked) return
      await run(() => confirmAdminMeasurements(dialog.tailoring))
    } else if (statusSpec) await run(() => performAdminTailoringAction(dialog.tailoring, dialog.action as TailoringAction))
    setSelected(null)
    setChecked(false)
  }
  const save = (patch: TailoringPatch) => run(() => saveAdminTailoring(t, patch), true)
  const open = (action: Pending['action']) => {
    setChecked(false)
    setSelected({ source: read, tailoring: t, action })
  }
  const modalOpen = !!dialog && (dialog.action === 'confirm_measurements' || !!statusSpec)

  return (
    <AdminOrderFrame title={`Yêu cầu của ${t.full_name}`}>
      <div className="flex flex-wrap justify-between gap-4">
        <p className="text-on-surface-variant">
          {tailoringStatusLabel(t.status)} · {t.is_guest ? 'Khách vãng lai' : 'Khách tài khoản'}
        </p>
        <button type="button" disabled={busy} className={buttonClass} onClick={retry}>Tải lại yêu cầu</button>
      </div>
      {outcome?.kind === 'conflict' && <p role="alert" className="border border-error bg-error-container text-on-error-container p-4">{outcome.message}</p>}
      {outcome?.kind === 'refreshed' && <p role="alert" className="border border-error bg-error-container text-on-error-container p-4">{outcome.message}</p>}
      {outcome?.kind === 'saved' && <p role="status" className="bg-secondary-fixed/30 p-4 text-primary">Đã lưu thay đổi.</p>}
      <div className="grid lg:grid-cols-[1fr_1.2fr] gap-7 items-start">
        <div className="space-y-5 min-w-0">
          <section className="bg-surface p-5 sm:p-6 space-y-3">
            <h2 className="font-headline-md text-2xl text-primary">Liên hệ và mẫu quan tâm</h2>
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
              <dt className="text-on-surface-variant">Họ tên</dt><dd className="break-words">{t.full_name}</dd>
              <dt className="text-on-surface-variant">Điện thoại</dt><dd>{t.phone}</dd>
              <dt className="text-on-surface-variant">Kênh ưu tiên</dt><dd>{t.preferred_contact === 'zalo' ? 'Zalo' : t.preferred_contact === 'phone' ? 'Điện thoại' : 'Chưa chọn'}</dd>
              <dt className="text-on-surface-variant">Mẫu quan tâm</dt><dd className="break-words">{t.product_slug || 'Chưa chọn mẫu'}{t.product_id && <span className="block text-xs text-on-surface-variant font-mono">{t.product_id}</span>}</dd>
              <dt className="text-on-surface-variant">Tài khoản</dt><dd>{t.is_guest ? 'Khách vãng lai (không có tài khoản)' : 'Có tài khoản'}</dd>
              <dt className="text-on-surface-variant">Phụ trách</dt><dd>{t.assigned_staff || 'Chưa phân công'}</dd>
              <dt className="text-on-surface-variant">Lịch hẹn</dt><dd>{t.appointment_at ? <time dateTime={t.appointment_at}>{formatOrderDate(t.appointment_at)}</time> : 'Chưa hẹn'}</dd>
              <dt className="text-on-surface-variant">Ngày gửi</dt><dd><time dateTime={t.created_at}>{formatOrderDate(t.created_at)}</time></dd>
              <dt className="text-on-surface-variant">Cập nhật</dt><dd><time dateTime={t.updated_at}>{formatOrderDate(t.updated_at)}</time></dd>
            </dl>
            <div>
              <h3 className="font-semibold mt-3">Ghi chú</h3>
              <p className="whitespace-pre-wrap break-words">{t.notes || 'Chưa có ghi chú'}</p>
            </div>
          </section>
          <section className="bg-surface p-5 sm:p-6 space-y-3">
            <h2 className="font-headline-md text-2xl text-primary">Số đo</h2>
            <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
              <dt className="text-on-surface-variant">Mô tả vóc dáng</dt><dd className="break-words">{t.measurements?.body || 'chưa có'}</dd>
              {measurementKeys.map((key) => (
                <div key={key} className="contents">
                  <dt className="text-on-surface-variant">{measurementLabels[key]}</dt>
                  <dd>{t.measurements?.[key] !== undefined ? `${t.measurements[key]} cm` : 'chưa có'}</dd>
                </div>
              ))}
            </dl>
          </section>
          <section className="bg-surface p-5 sm:p-6 space-y-4">
            <h2 className="font-headline-md text-2xl text-primary">Trạng thái</h2>
            <div className="flex flex-wrap gap-3">
              {buttons.status.map((action) => (
                <button type="button" key={action.id} disabled={busy} className={buttonClass} onClick={() => open(action.id)}>{action.label}</button>
              ))}
              {buttons.canConfirmMeasurements && (
                <button type="button" disabled={busy} className={buttonClass} onClick={() => open('confirm_measurements')}>Xác nhận số đo</button>
              )}
              {!buttons.status.length && !buttons.canConfirmMeasurements && <p className="text-on-surface-variant">Hiện không có thao tác trạng thái được phép.</p>}
            </div>
            {t.status === 'consultation_scheduled' || t.status === 'contacted' ? (
              !buttons.canConfirmMeasurements && <p className="text-sm text-on-surface-variant">Cần đủ bảy số đo (cm) mới xác nhận được số đo.</p>
            ) : null}
          </section>
        </div>
        <div className="space-y-5 min-w-0">
          <section className="bg-surface p-5 sm:p-7 space-y-4">
            <h2 className="font-headline-md text-2xl text-primary">Chỉnh sửa</h2>
            <AdminTailoringEditForm key={`${t.id}:${t.updated_at}`} tailoring={t} canEditMeasurements={buttons.canEditMeasurements} busy={busy} onSave={save} />
          </section>
          {orderablePhase(t.status) && (
            <AdminCustomOrderPanel
              key={t.id}
              tailoring={t}
              onRefreshed={(latest, message) => setOverride({ source: read, outcome: { kind: 'refreshed', tailoring: latest, message } })}
              onFailure={(failure) => setOverride({ source: read, outcome: failure })}
            />
          )}
        </div>
      </div>
      <Modal open={modalOpen} onClose={() => { if (!busy) setSelected(null) }}>
        {dialog && modalOpen && (
          <div className="space-y-5">
            <h2 className="font-headline-md text-2xl text-primary">
              {dialog.action === 'confirm_measurements' ? 'Xác nhận số đo' : statusSpec?.label} · {t.full_name}
            </h2>
            <p>{dialog.action === 'confirm_measurements' ? confirmMeasurementsText : adminTailoringConfirmation(dialog.tailoring, dialog.action as TailoringAction)}</p>
            {dialog.action === 'confirm_measurements' && (
              <label className="flex items-start gap-3">
                <input type="checkbox" checked={checked} onChange={(e) => setChecked(e.target.checked)} className="mt-1" />
                <span>{confirmMeasurementsCheckbox}</span>
              </label>
            )}
            <div className="flex flex-wrap gap-3">
              <button type="button" disabled={busy || (dialog.action === 'confirm_measurements' && !checked)} className="bg-primary text-on-primary px-5 py-3 disabled:opacity-40" onClick={() => void confirmDialog()}>
                {busy ? 'Đang gửi...' : 'Xác nhận thao tác'}
              </button>
              <button type="button" disabled={busy} className={buttonClass} onClick={() => setSelected(null)}>Quay lại</button>
            </div>
          </div>
        )}
      </Modal>
    </AdminOrderFrame>
  )
}
