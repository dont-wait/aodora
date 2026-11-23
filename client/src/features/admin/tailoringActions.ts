import {
  adminTailoring, AdminTailoringApiError, measurementKeys, tailoringStatusLabel,
  type AdminTailoring, type AdminTailoringFilter, type AdminTailoringPage, type CustomOrderBody, type CustomOrderResult, type MeasurementKey, type TailoringPatch,
} from './tailoring.ts'
import { parseMeasurement } from '../tailoring/form.ts'

export type TailoringAction = 'contact' | 'schedule' | 'start' | 'complete' | 'cancel'
const statusActions: { id: TailoringAction; target: 'contacted' | 'consultation_scheduled' | 'in_progress' | 'completed' | 'cancelled'; label: string }[] = [
  { id: 'contact', target: 'contacted', label: 'Đã liên hệ khách' },
  { id: 'schedule', target: 'consultation_scheduled', label: 'Đã hẹn tư vấn' },
  { id: 'start', target: 'in_progress', label: 'Bắt đầu thực hiện may' },
  { id: 'complete', target: 'completed', label: 'Hoàn tất' },
  { id: 'cancel', target: 'cancelled', label: 'Hủy yêu cầu' },
]
/** Buttons are generated only from the server's allowed_actions. */
export function adminTailoringActionButtons(t: Pick<AdminTailoring, 'allowed_actions'>) {
  const allowed = new Set(t.allowed_actions)
  return {
    status: statusActions.filter((a) => allowed.has(a.id)),
    canConfirmMeasurements: allowed.has('confirm_measurements'),
    canEditMeasurements: allowed.has('update_measurements'),
  }
}
export function adminTailoringConfirmation(t: AdminTailoring, action: TailoringAction): string {
  const target = statusActions.find((a) => a.id === action)!.target
  switch (action) {
    case 'cancel':
      return 'Hủy yêu cầu may đo. Thao tác không hoàn tác: yêu cầu đã hủy không thể mở lại, và số đo đã nhập bị khóa.'
    case 'start':
      return 'Xác nhận bắt đầu thực hiện may. Chỉ làm khi số đo đã chốt; sau đó không sửa số đo trực tiếp.'
    case 'complete':
      return 'Xác nhận đã hoàn tất. Yêu cầu hoàn tất không thể hủy hoặc chuyển lại trạng thái trước.'
    default:
      return `Chuyển trạng thái từ "${tailoringStatusLabel(t.status)}" sang "${tailoringStatusLabel(target)}". Không thể quay lại bước trước.`
  }
}
export const confirmMeasurementsText =
  'Xác nhận số đo sẽ khóa chỉnh sửa số đo trực tiếp (hiện chưa có luồng mở lại) và cho phép tạo đơn may đo. Không có thao tác tự động: chỉ xác nhận sau khi đã kiểm tra lại với khách.'
export const confirmMeasurementsCheckbox = 'Tôi đã kiểm tra lại số đo với khách'

export type TailoringFailure = { kind: 'unauthenticated' | 'forbidden' | 'not-found' | 'error'; message: string }
export type TailoringRead = { kind: 'ready'; tailoring: AdminTailoring } | TailoringFailure
export type TailoringListRead = { kind: 'ready'; page: AdminTailoringPage } | TailoringFailure
export type TailoringMutation =
  | { kind: 'saved'; tailoring: AdminTailoring }
  | { kind: 'conflict'; tailoring: AdminTailoring; message: string }
  | { kind: 'invalid'; field: string; message: string }
  | TailoringFailure

function failure(error: unknown, message = 'Không thể tải yêu cầu may đo. Vui lòng thử lại.'): TailoringFailure {
  if (error instanceof AdminTailoringApiError) {
    if (error.status === 401) return { kind: 'unauthenticated', message: 'Vui lòng đăng nhập lại.' }
    if (error.status === 403) return { kind: 'forbidden', message: 'Bạn không có quyền quản trị yêu cầu may đo.' }
    if (error.status === 404) return { kind: 'not-found', message: 'Không tìm thấy yêu cầu may đo.' }
  }
  return { kind: 'error', message }
}
export async function loadAdminTailoring(id: string): Promise<TailoringRead> {
  try {
    return { kind: 'ready', tailoring: await adminTailoring.get(id) }
  } catch (error) {
    return failure(error)
  }
}
export async function loadAdminTailorings(filter: AdminTailoringFilter): Promise<TailoringListRead> {
  try {
    return { kind: 'ready', page: await adminTailoring.list(filter) }
  } catch (error) {
    return failure(error)
  }
}

const conflictMessages: Record<string, string> = {
  MEASUREMENT_LOCKED: 'Số đo đã chốt hoặc yêu cầu đã kết thúc nên không sửa trực tiếp được.',
  MEASUREMENT_INCOMPLETE: 'Cần đủ bảy số đo (cm) trước khi xác nhận.',
  TAILORING_STATE_CONFLICT: 'Trạng thái yêu cầu đã thay đổi hoặc thao tác không còn hợp lệ.',
}
/** 409: reload the request; a failed reload never falls back to stale data. */
async function conflict(id: string, error: AdminTailoringApiError): Promise<TailoringMutation> {
  const message = conflictMessages[error.code] || conflictMessages.TAILORING_STATE_CONFLICT
  const latest = await loadAdminTailoring(id)
  if (latest.kind === 'ready') return { kind: 'conflict', tailoring: latest.tailoring, message: `${message} Đã tải lại thông tin mới nhất.` }
  if (latest.kind === 'error') return { kind: 'error', message: `${message} Chưa tải lại được yêu cầu; vui lòng tải lại trước khi thao tác tiếp.` }
  return latest
}
async function mutate(t: AdminTailoring, run: () => Promise<AdminTailoring>): Promise<TailoringMutation> {
  try {
    return { kind: 'saved', tailoring: await run() }
  } catch (error) {
    if (error instanceof AdminTailoringApiError) {
      if (error.status === 409) return conflict(t.id, error)
      if (error.status === 400) return { kind: 'invalid', field: error.field, message: error.message }
    }
    return failure(error, 'Chưa xác định được kết quả thao tác. Vui lòng tải lại yêu cầu trước khi thao tác tiếp.')
  }
}
export function performAdminTailoringAction(t: AdminTailoring, action: TailoringAction): Promise<TailoringMutation> {
  const spec = statusActions.find((a) => a.id === action)
  if (!spec || !t.allowed_actions.includes(action)) return conflict(t.id, new AdminTailoringApiError(409, 'TAILORING_STATE_CONFLICT', '', ''))
  return mutate(t, () => adminTailoring.setStatus(t.id, spec.target))
}
export function confirmAdminMeasurements(t: AdminTailoring): Promise<TailoringMutation> {
  if (!t.allowed_actions.includes('confirm_measurements')) return conflict(t.id, new AdminTailoringApiError(409, 'MEASUREMENT_INCOMPLETE', '', ''))
  return mutate(t, () => adminTailoring.confirmMeasurements(t.id))
}
export function saveAdminTailoring(t: AdminTailoring, patch: TailoringPatch): Promise<TailoringMutation> {
  return mutate(t, () => adminTailoring.update(t.id, patch))
}

// ---- edit form ⇄ PATCH ----
export type TailoringEditValues = {
  notes: string
  assigned_staff: string
  preferred_contact: '' | 'phone' | 'zalo'
  appointment: string // datetime-local value
  body: string
  measurements: Record<MeasurementKey, string>
  cleared: Record<MeasurementKey | 'body', boolean>
}
const blankMeasurements = () => Object.fromEntries(measurementKeys.map((k) => [k, ''])) as Record<MeasurementKey, string>
const noneCleared = () => Object.fromEntries([...measurementKeys, 'body'].map((k) => [k, false])) as Record<MeasurementKey | 'body', boolean>

/** Measurement boxes start empty: blank means "unchanged"; clearing is an explicit per-field action. */
export function initialTailoringEdit(t: AdminTailoring): TailoringEditValues {
  return {
    notes: t.notes ?? '',
    assigned_staff: t.assigned_staff ?? '',
    preferred_contact: t.preferred_contact === 'phone' || t.preferred_contact === 'zalo' ? t.preferred_contact : '',
    appointment: toLocalInput(t.appointment_at),
    body: '',
    measurements: blankMeasurements(),
    cleared: noneCleared(),
  }
}
const pad = (n: number) => String(n).padStart(2, '0')
/** RFC3339 (UTC) → value for <input type="datetime-local"> in the browser's zone. */
export function toLocalInput(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}
/** datetime-local value → RFC3339 UTC, or null when it is not a real date/time. */
export function fromLocalInput(local: string): string | null {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(local)) return null
  const d = new Date(local)
  return Number.isNaN(d.getTime()) ? null : d.toISOString()
}

/**
 * Builds a PATCH containing only changed fields. Never includes status or
 * customer_id. Blank measurement ⇒ omitted (never 0); `cleared` ⇒ null.
 */
export function buildTailoringPatch(t: AdminTailoring, v: TailoringEditValues, canEditMeasurements: boolean): { patch: TailoringPatch; errors: Record<string, string> } {
  const patch: TailoringPatch = {}
  const errors: Record<string, string> = {}
  const text = (value: string, original: string | null) => {
    const next = value.trim()
    return next === (original ?? '') ? undefined : next === '' ? null : next
  }
  const notes = text(v.notes, t.notes)
  if (notes !== undefined) patch.notes = notes
  const staff = text(v.assigned_staff, t.assigned_staff)
  if (staff !== undefined) patch.assigned_staff = staff
  const originalContact = t.preferred_contact === 'phone' || t.preferred_contact === 'zalo' ? t.preferred_contact : ''
  if (v.preferred_contact !== originalContact) patch.preferred_contact = v.preferred_contact === '' ? null : v.preferred_contact
  if (v.appointment !== toLocalInput(t.appointment_at)) {
    if (v.appointment === '') patch.appointment_at = null
    else {
      const iso = fromLocalInput(v.appointment)
      if (iso) patch.appointment_at = iso
      else errors.appointment = 'Lịch hẹn không hợp lệ.'
    }
  }
  if (canEditMeasurements) {
    const m: NonNullable<TailoringPatch['measurements']> = {}
    for (const key of measurementKeys) {
      if (v.cleared[key]) m[key] = null
      else if (v.measurements[key].trim() !== '') {
        const n = parseMeasurement(v.measurements[key].trim())
        if (n === null) errors[key] = 'Số đo phải là số lớn hơn 0 và không quá 500 cm.'
        else m[key] = n
      }
    }
    if (v.cleared.body) m.body = null
    else if (v.body.trim() !== '') {
      if (v.body.trim().length > 500) errors.body = 'Mô tả vóc dáng tối đa 500 ký tự.'
      else m.body = v.body.trim()
    }
    if (Object.keys(m).length > 0) patch.measurements = m
  }
  return { patch, errors }
}

// ---- custom order ----
export type CustomOrderValues = {
  unit_price: string
  quantity: string
  payment_method: '' | 'cod' | 'cash_at_counter'
  product_id: string
  recipient: string
  phone: string
  line1: string
  ward: string
  district: string
  city: string
  note: string
}
export const orderablePhase = (status: string) => status === 'measurement_confirmed' || status === 'in_progress'
export function initialCustomOrder(t: AdminTailoring): CustomOrderValues {
  return { unit_price: '', quantity: '1', payment_method: '', product_id: '', recipient: t.full_name, phone: t.phone, line1: '', ward: '', district: '', city: '', note: '' }
}
export const paymentMethodLabels = { cod: 'Thu tiền khi giao (COD)', cash_at_counter: 'Thu tiền tại quầy' } as const

/** Local checks only; the server stays authoritative. unit_price is kept as the typed text. */
export function validateCustomOrder(t: Pick<AdminTailoring, 'product_id'>, v: CustomOrderValues): { body?: CustomOrderBody; errors: Record<string, string> } {
  const errors: Record<string, string> = {}
  const price = v.unit_price.trim()
  if (!/^\d+(\.\d+)?$/.test(price) || /^0+(\.0+)?$/.test(price) || price.replace('.', '').replace(/^0+/, '').length > 34) {
    errors.unit_price = 'Giá phải là số thập phân lớn hơn 0 (dùng dấu chấm), ví dụ 1500000 hoặc 1500000.50.'
  }
  const q = v.quantity.trim()
  const quantity = /^\d+$/.test(q) ? Number(q) : 0
  if (!Number.isSafeInteger(quantity) || quantity < 1 || quantity > 1_000_000) errors.quantity = 'Số lượng phải là số nguyên từ 1 đến 1.000.000.'
  if (v.payment_method !== 'cod' && v.payment_method !== 'cash_at_counter') errors.payment_method = 'Chọn hình thức thu tiền.'
  const product = v.product_id.trim()
  if (!t.product_id && !/^[0-9a-fA-F]{24}$/.test(product)) errors.product_id = 'Yêu cầu chưa có mẫu: nhập mã sản phẩm (24 ký tự hex).'
  if (!v.recipient.trim()) errors.recipient = 'Nhập tên người nhận.'
  if (!v.phone.trim()) errors.phone = 'Nhập số điện thoại người nhận.'
  if (v.payment_method === 'cod') {
    if (!v.line1.trim()) errors.line1 = 'COD cần địa chỉ giao (số nhà, đường).'
    if (!v.city.trim()) errors.city = 'COD cần tỉnh/thành phố.'
  }
  if (v.note.trim().length > 1000) errors.note = 'Ghi chú tối đa 1000 ký tự.'
  if (Object.keys(errors).length > 0) return { errors }
  const address: NonNullable<CustomOrderBody['shipping_address']> = { recipient: v.recipient.trim(), phone: v.phone.trim() }
  for (const key of ['line1', 'ward', 'district', 'city'] as const) if (v[key].trim()) address[key] = v[key].trim()
  const body: CustomOrderBody = { unit_price: price, quantity, payment_method: v.payment_method as 'cod' | 'cash_at_counter', shipping_address: address }
  if (!t.product_id) body.product_id = product
  if (v.note.trim()) body.note = v.note.trim()
  return { body, errors }
}

export type CustomOrderOutcome =
  | { kind: 'created'; result: CustomOrderResult }
  | { kind: 'exists'; result: CustomOrderResult } // 200: this request already had an identical order
  | { kind: 'different-order-exists'; message: string }
  | { kind: 'shipping-unavailable'; message: string }
  | { kind: 'invalid'; field: string; message: string }
  | { kind: 'conflict'; tailoring: AdminTailoring; message: string }
  | { kind: 'unknown'; message: string }
  | TailoringFailure

export async function createAdminCustomOrder(t: AdminTailoring, body: CustomOrderBody): Promise<CustomOrderOutcome> {
  try {
    const result = await adminTailoring.createOrder(t.id, body)
    return { kind: result.created ? 'created' : 'exists', result }
  } catch (error) {
    if (error instanceof AdminTailoringApiError) {
      const { status, code } = error
      if (status === 401 || status === 403 || status === 404) return failure(error)
      if (status === 400) return { kind: 'invalid', field: error.field, message: error.message }
      if (status === 409 && code === 'CUSTOM_ORDER_EXISTS') {
        return { kind: 'different-order-exists', message: 'Yêu cầu này đã có đơn may đo với nội dung khác. Hệ thống không tạo đơn thứ hai; hãy kiểm tra đơn đã có trong trang đơn hàng.' }
      }
      if (status === 409) {
        const message = conflictMessages[code] || conflictMessages.TAILORING_STATE_CONFLICT
        const latest = await loadAdminTailoring(t.id)
        if (latest.kind === 'ready') return { kind: 'conflict', tailoring: latest.tailoring, message: `${message} Đã tải lại thông tin mới nhất.` }
        return latest.kind === 'error' ? { kind: 'error', message: `${message} Chưa tải lại được yêu cầu; vui lòng tải lại.` } : latest
      }
      if (status === 503 && code === 'SHIPPING_UNAVAILABLE') {
        return { kind: 'shipping-unavailable', message: 'Phí COD chưa được cấu hình trên máy chủ nên chưa tạo được đơn COD. Có thể chọn thu tại quầy hoặc nhờ quản trị cấu hình phí.' }
      }
    }
    return { kind: 'unknown', message: 'Chưa rõ đơn đã được tạo hay chưa (lỗi mạng hoặc máy chủ). Hệ thống chống tạo trùng, nhưng hãy kiểm tra danh sách đơn hoặc bấm tạo lại với đúng nội dung này.' }
  }
}
