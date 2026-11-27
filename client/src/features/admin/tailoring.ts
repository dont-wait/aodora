import { parseExactJSON } from '../orders/money.ts'
import { readOrdersPage } from '../orders/read.ts'

// All numbers in responses arrive as exact strings (parseExactJSON), measurements included.
export type AdminTailoringMeasurements = {
  body?: string
  neck_cm?: string
  bust_cm?: string
  waist_cm?: string
  hips_cm?: string
  shoulder_cm?: string
  sleeve_cm?: string
  garment_length_cm?: string
}
export type AdminTailoring = {
  id: string
  customer_id: string | null
  is_guest: boolean
  full_name: string
  phone: string
  preferred_contact: string | null
  category_id: string | null
  product_id: string | null
  product_slug: string | null
  measurements: AdminTailoringMeasurements | null
  notes: string | null
  status: string
  appointment_at: string | null
  assigned_staff: string | null
  created_at: string
  updated_at: string
  allowed_actions: string[]
}
export type AdminTailoringPage = { data: AdminTailoring[]; total: number; page: number; page_size: number }
export type AdminTailoringFilter = { status: string; page: number; page_size: number }

export const adminTailoringStatuses = [
  'submitted', 'contacted', 'consultation_scheduled', 'measurement_confirmed', 'in_progress', 'completed', 'cancelled',
] as const
// system.md 5.3
export const tailoringStatusLabels: Record<string, string> = {
  submitted: 'Đã gửi',
  contacted: 'Đã liên hệ',
  consultation_scheduled: 'Đã hẹn tư vấn',
  measurement_confirmed: 'Đã xác nhận số đo',
  in_progress: 'Đang thực hiện',
  completed: 'Hoàn tất',
  cancelled: 'Đã hủy',
}
export const tailoringStatusLabel = (status: string) => tailoringStatusLabels[status] || 'Trạng thái không xác định'
export const measurementKeys = ['neck_cm', 'bust_cm', 'waist_cm', 'hips_cm', 'shoulder_cm', 'sleeve_cm', 'garment_length_cm'] as const
export type MeasurementKey = (typeof measurementKeys)[number]
export const measurementLabels: Record<MeasurementKey, string> = {
  neck_cm: 'Cổ', bust_cm: 'Ngực', waist_cm: 'Eo', hips_cm: 'Mông', shoulder_cm: 'Rộng vai', sleeve_cm: 'Dài tay', garment_length_cm: 'Dài áo',
}

/** Unknown/repeated query values never reach the API. */
export function readAdminTailoringFilters(input: URLSearchParams): AdminTailoringFilter {
  const unique = (key: string) => (input.getAll(key).length === 1 ? input.get(key) || '' : '')
  const status = unique('status')
  const sizeText = unique('page_size')
  const size = /^\d+$/.test(sizeText) ? Number(sizeText) : 20
  return {
    status: adminTailoringStatuses.some((value) => value === status) ? status : '',
    page: readOrdersPage(unique('page')),
    page_size: Number.isSafeInteger(size) && size >= 1 ? Math.min(100, size) : 20,
  }
}
export function adminTailoringSearch(filter: AdminTailoringFilter): URLSearchParams {
  const safe = readAdminTailoringFilters(new URLSearchParams({ status: filter.status, page: String(filter.page), page_size: String(filter.page_size) }))
  const params = new URLSearchParams({ page: String(safe.page), page_size: String(safe.page_size) })
  if (safe.status) params.set('status', safe.status)
  return params
}

export class AdminTailoringApiError extends Error {
  status: number
  code: string
  field: string
  constructor(status: number, code: string, field: string, message: string) {
    super(message)
    this.name = 'AdminTailoringApiError'
    this.status = status
    this.code = code
    this.field = field
  }
}
const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
function invalid(): never {
  throw new AdminTailoringApiError(200, 'INVALID_RESPONSE', '', 'Dữ liệu yêu cầu may đo trả về chưa đầy đủ. Vui lòng tải lại.')
}
function integer(value: unknown, minimum = 0): number {
  const number = Number(value)
  if (!Number.isSafeInteger(number) || number < minimum) invalid()
  return number
}
const nullableString = (value: unknown) => (value === null || value === undefined ? null : typeof value === 'string' ? value : invalid())
function parseTailoring(raw: unknown): AdminTailoring {
  const data = raw as Record<string, unknown> | null
  if (
    !data ||
    typeof data.id !== 'string' || !data.id ||
    typeof data.full_name !== 'string' ||
    typeof data.phone !== 'string' ||
    typeof data.status !== 'string' ||
    typeof data.created_at !== 'string' ||
    typeof data.updated_at !== 'string' ||
    typeof data.is_guest !== 'boolean' ||
    !Array.isArray(data.allowed_actions) ||
    !data.allowed_actions.every((action) => typeof action === 'string')
  )
    invalid()
  const m = data.measurements
  if (m !== null && m !== undefined && (typeof m !== 'object' || Array.isArray(m))) invalid()
  return {
    id: data.id as string,
    customer_id: nullableString(data.customer_id),
    is_guest: data.is_guest as boolean,
    full_name: data.full_name as string,
    phone: data.phone as string,
    preferred_contact: nullableString(data.preferred_contact),
    category_id: nullableString(data.category_id),
    product_id: nullableString(data.product_id),
    product_slug: nullableString(data.product_slug),
    measurements: (m as AdminTailoringMeasurements | null | undefined) ?? null,
    notes: nullableString(data.notes),
    status: data.status as string,
    appointment_at: nullableString(data.appointment_at),
    assigned_staff: nullableString(data.assigned_staff),
    created_at: data.created_at as string,
    updated_at: data.updated_at as string,
    allowed_actions: data.allowed_actions as string[],
  }
}

async function request(path: string, method = 'GET', input?: unknown, okStatuses: number[] = [200]): Promise<Record<string, unknown>> {
  const response = await fetch(`${baseUrl}/api/v1/admin/tailoring-requests${path}`, {
    method,
    credentials: 'include',
    headers: { Accept: 'application/json', ...(input === undefined ? {} : { 'Content-Type': 'application/json' }) },
    body: input === undefined ? undefined : JSON.stringify(input),
  })
  let payload: Record<string, unknown> | null = null
  try {
    payload = parseExactJSON(await response.text()) as Record<string, unknown>
  } catch {
    /* Never accept ambiguous responses as saved. */
  }
  if (!response.ok) {
    const error = payload?.error as { code?: string; field?: string; message?: string } | undefined
    throw new AdminTailoringApiError(response.status, error?.code || '', error?.field || '', error?.message || 'Không thể xử lý yêu cầu may đo. Vui lòng thử lại.')
  }
  if (!okStatuses.includes(response.status) || !payload?.data) invalid()
  return payload
}
const pathFor = (id: string) => `/${encodeURIComponent(id)}`

export type TailoringPatch = {
  notes?: string | null
  appointment_at?: string | null
  assigned_staff?: string | null
  preferred_contact?: 'phone' | 'zalo' | null
  measurements?: Partial<Record<MeasurementKey, number | null>> & { body?: string | null }
}
export type CustomOrderBody = {
  product_id?: string
  /** Exact decimal text; sent as a JSON string so no float conversion ever happens. */
  unit_price: string
  quantity: number
  payment_method: 'cod' | 'cash_at_counter'
  shipping_address?: { recipient: string; phone: string; line1?: string; ward?: string; district?: string; city?: string }
  note?: string
}
export type CustomOrderResult = { orderId: string; orderNumber: string; created: boolean }

export const adminTailoring = {
  list: async (filter: AdminTailoringFilter): Promise<AdminTailoringPage> => {
    const payload = await request(`?${adminTailoringSearch(filter)}`)
    if (!Array.isArray(payload.data)) invalid()
    return { data: payload.data.map(parseTailoring), total: integer(payload.total), page: integer(payload.page, 1), page_size: integer(payload.page_size, 1) }
  },
  get: async (id: string) => parseTailoring((await request(pathFor(id))).data),
  update: async (id: string, patch: TailoringPatch) => parseTailoring((await request(pathFor(id), 'PATCH', patch)).data),
  setStatus: async (id: string, status: 'contacted' | 'consultation_scheduled' | 'in_progress' | 'completed' | 'cancelled') =>
    parseTailoring((await request(`${pathFor(id)}/status`, 'PATCH', { status })).data),
  confirmMeasurements: async (id: string) => parseTailoring((await request(`${pathFor(id)}/confirm-measurements`, 'POST', { confirmed: true })).data),
  createOrder: async (id: string, body: CustomOrderBody): Promise<CustomOrderResult> => {
    const payload = await request(`${pathFor(id)}/order`, 'POST', body, [200, 201])
    const data = payload.data as { id?: unknown; order_number?: unknown } | null
    if (typeof data?.id !== 'string' || !data.id || typeof payload.created !== 'boolean') invalid()
    return { orderId: data.id, orderNumber: typeof data.order_number === 'string' ? data.order_number : '', created: payload.created }
  },
}
