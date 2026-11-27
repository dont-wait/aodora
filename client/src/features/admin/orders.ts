import { parseExactJSON } from '../orders/money.ts'
import { readOrdersPage } from '../orders/read.ts'
import type { Order } from '../orders/types.ts'

export type AdminOrder = Omit<Order, 'items'> & {
  items: (Order['items'][number] & { fulfillment_mode: string })[]
  updated_at: string
  customer_id?: string
  is_guest: boolean
  cancelled_at?: string
  inventory_restock?: string
  inventory_restocked_at?: string
  paid_at?: string
  refunded_at?: string
  allowed_actions: string[]
}
export type AdminOrderPage = { data: AdminOrder[]; total: number; page: number; page_size: number }
export type AdminOrderFilter = { status: string; payment_status: string; page: number; page_size: number }
export const adminOrderStatuses = ['pending', 'confirmed', 'shipping', 'delivered', 'cancelled'] as const
export const adminPaymentStatuses = ['unpaid', 'paid', 'refunded'] as const

export function readAdminOrderFilters(input: URLSearchParams): AdminOrderFilter {
  const unique = (key: string) => (input.getAll(key).length === 1 ? input.get(key) || '' : '')
  const status = unique('status')
  const payment = unique('payment_status')
  const sizeText = unique('page_size')
  const size = /^\d+$/.test(sizeText) ? Number(sizeText) : 20
  return {
    status: adminOrderStatuses.some((value) => value === status) ? status : '',
    payment_status: adminPaymentStatuses.some((value) => value === payment) ? payment : '',
    page: readOrdersPage(unique('page')),
    page_size: Number.isSafeInteger(size) && size >= 1 ? Math.min(100, size) : 20,
  }
}
export function adminOrderSearch(filter: AdminOrderFilter): URLSearchParams {
  const raw = new URLSearchParams({
    status: filter.status,
    payment_status: filter.payment_status,
    page: String(filter.page),
    page_size: String(filter.page_size),
  })
  const safe = readAdminOrderFilters(raw)
  const params = new URLSearchParams({ page: String(safe.page), page_size: String(safe.page_size) })
  if (safe.status) params.set('status', safe.status)
  if (safe.payment_status) params.set('payment_status', safe.payment_status)
  return params
}

export class AdminOrderApiError extends Error {
  status: number
  code: string
  field: string
  constructor(status: number, code: string, field: string, message: string) {
    super(message)
    this.name = 'AdminOrderApiError'
    this.status = status
    this.code = code
    this.field = field
  }
}
const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
function invalid(): never {
  throw new AdminOrderApiError(200, 'INVALID_RESPONSE', '', 'Dữ liệu đơn trả về chưa đầy đủ. Vui lòng tải lại.')
}
function integer(value: unknown, minimum = 0): number {
  const number = Number(value)
  if (!Number.isSafeInteger(number) || number < minimum) invalid()
  return number
}
function adminOrder(raw: unknown): AdminOrder {
  const data = raw as AdminOrder
  if (
    !data ||
    typeof data.id !== 'string' ||
    typeof data.order_number !== 'string' ||
    typeof data.updated_at !== 'string' ||
    typeof data.is_guest !== 'boolean' ||
    !data.shipping_address ||
    typeof data.status !== 'string' ||
    typeof data.payment_status !== 'string' ||
    !Array.isArray(data.allowed_actions) ||
    !data.allowed_actions.every((action) => typeof action === 'string') ||
    ![data.subtotal, data.shipping_fee, data.total].every((value) => typeof value === 'string' && /^\d+(?:\.\d+)?$/.test(value)) ||
    !Array.isArray(data.items)
  )
    invalid()
  return {
    ...data,
    items: data.items.map((item) => {
      if (typeof item.unit_price !== 'string' || !/^\d+(?:\.\d+)?$/.test(item.unit_price) || typeof item.fulfillment_mode !== 'string') invalid()
      return { ...item, quantity: integer(item.quantity, 1) }
    }),
  }
}
async function request(path: string, method = 'GET', input?: unknown): Promise<Record<string, unknown>> {
  const response = await fetch(`${baseUrl}/api/v1/admin/orders${path}`, {
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
    throw new AdminOrderApiError(response.status, error?.code || '', error?.field || '', error?.message || 'Không thể xử lý đơn hàng. Vui lòng thử lại.')
  }
  if (response.status !== 200 || !payload?.data) invalid()
  return payload
}
const pathFor = (id: string) => `/${encodeURIComponent(id)}`
export const adminOrders = {
  list: async (filter: AdminOrderFilter): Promise<AdminOrderPage> => {
    const payload = await request(`?${adminOrderSearch(filter)}`)
    if (!Array.isArray(payload.data)) invalid()
    return { data: payload.data.map(adminOrder), total: integer(payload.total), page: integer(payload.page, 1), page_size: integer(payload.page_size, 1) }
  },
  get: async (id: string) => adminOrder((await request(pathFor(id))).data),
  setStatus: async (id: string, status: 'confirmed' | 'shipping' | 'delivered') =>
    adminOrder((await request(`${pathFor(id)}/status`, 'PATCH', { status })).data),
  cancel: async (id: string) => adminOrder((await request(`${pathFor(id)}/cancel`, 'POST')).data),
  restock: async (id: string) => adminOrder((await request(`${pathFor(id)}/restock`, 'POST')).data),
  setPayment: async (id: string, payment_status: 'paid' | 'refunded') =>
    adminOrder((await request(`${pathFor(id)}/payment`, 'PATCH', { payment_status })).data),
}
