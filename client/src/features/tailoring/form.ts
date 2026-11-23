import type { PreferredContact, TailoringMeasurements, TailoringRequestBody } from './types.ts'

export interface TailoringFormValues {
  name: string
  phone: string
  contact: '' | PreferredContact
  neck: string
  bust: string
  waist: string
  hips: string
  shoulder: string
  sleeve: string
  length: string
  body: string
  notes: string
}

export const emptyTailoringForm: TailoringFormValues = {
  name: '', phone: '', contact: '', neck: '', bust: '', waist: '', hips: '', shoulder: '', sleeve: '', length: '', body: '', notes: '',
}

export const measurementFieldMap = {
  neck: 'neck_cm', bust: 'bust_cm', waist: 'waist_cm', hips: 'hips_cm', shoulder: 'shoulder_cm', sleeve: 'sleeve_cm', length: 'garment_length_cm',
} as const

/** Maps a server field name to the form control that owns it. */
export function formFieldForServerField(field: string): string {
  if (field === 'full_name' || field === 'name') return 'name'
  if (field === 'preferred_contact') return 'contact'
  if (field === 'notes' || field === 'note') return 'notes'
  if (field === 'body') return 'body'
  if (field === 'product_id' || field === 'product_slug') return 'product'
  for (const [key, server] of Object.entries(measurementFieldMap)) if (server === field) return key
  return field
}

export interface ProductRef {
  id?: string
  slug?: string
}

/** Local errors by form field; empty when the form can be sent. */
export function validateTailoringForm(form: TailoringFormValues): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!form.name.trim()) errors.name = 'Vui lòng nhập họ và tên.'
  if (!form.phone.trim()) errors.phone = 'Vui lòng nhập số điện thoại.'
  for (const key of Object.keys(measurementFieldMap) as (keyof typeof measurementFieldMap)[]) {
    const raw = form[key].trim()
    if (raw === '') continue
    const n = parseMeasurement(raw)
    if (n === null) errors[key] = 'Số đo phải là số lớn hơn 0 và không quá 500 cm.'
  }
  return errors
}

/**
 * Shared parser for every measurement text box (tailoring request and KR-02
 * profile). Accepts "65", "65.5" and "65,5"; returns null for blank, text,
 * negative, 0, > 500 or exponent forms, so NaN/0 can never reach an API body.
 */
export function parseMeasurement(raw: string): number | null {
  const normalized = raw.replace(',', '.')
  if (!/^\d+(\.\d+)?$/.test(normalized)) return null
  const n = Number(normalized)
  return Number.isFinite(n) && n > 0 && n <= 500 ? n : null
}

/** Valid measurements keyed by API name; blank or invalid boxes are omitted (never NaN, never 0). */
export function measurementNumbers(form: TailoringFormValues): Partial<Record<(typeof measurementFieldMap)[keyof typeof measurementFieldMap], number>> {
  const out: Partial<Record<(typeof measurementFieldMap)[keyof typeof measurementFieldMap], number>> = {}
  for (const [key, server] of Object.entries(measurementFieldMap) as [keyof typeof measurementFieldMap, (typeof measurementFieldMap)[keyof typeof measurementFieldMap]][]) {
    const raw = form[key].trim()
    const n = raw === '' ? null : parseMeasurement(raw)
    if (n !== null) out[server] = n
  }
  return out
}

/**
 * Builds the API body. Empty measurements are omitted (never sent as 0), the
 * free-text `body` stays a description, and no customer_id/status is ever sent.
 */
export function buildTailoringRequest(form: TailoringFormValues, product: ProductRef = {}): TailoringRequestBody {
  const measurements: TailoringMeasurements = { ...measurementNumbers(form) }
  if (form.body.trim()) measurements.body = form.body.trim()
  const out: TailoringRequestBody = { full_name: form.name.trim(), phone: form.phone.trim() }
  if (form.contact) out.preferred_contact = form.contact
  if (product.id) out.product_id = product.id
  if (product.slug) out.product_slug = product.slug
  if (Object.keys(measurements).length > 0) out.measurements = measurements
  if (form.notes.trim()) out.notes = form.notes.trim()
  return out
}

/** Reads the sample the product page links with (?product_id=&product_slug=&mode=custom). */
export function productRefFromSearch(search: string): ProductRef {
  const params = new URLSearchParams(search)
  const id = params.get('product_id')?.trim()
  const slug = params.get('product_slug')?.trim()
  return { ...(id ? { id } : {}), ...(slug ? { slug } : {}) }
}

export const tailoringSuccessMessage =
  'Đã gửi yêu cầu tư vấn may đo — đây chưa phải đặt đơn thành công. Cửa hàng sẽ liên hệ xác nhận số đo và giá.'
