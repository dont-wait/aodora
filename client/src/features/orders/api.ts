import { parseExactJSON } from './money.ts'
import type { CheckoutBody, CheckoutAttempt, Order, OrderCredential, OrderPage, Quote } from './types.ts'

export class OrderApiError extends Error {
  status: number
  code: string
  field: string
  constructor(status: number, code: string, field: string, message: string) {
    super(message)
    this.name = 'OrderApiError'
    this.status = status
    this.code = code
    this.field = field
  }
}
const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function request(path: string, method = 'GET', body?: string, credential?: OrderCredential, key?: string): Promise<Record<string, unknown>> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (credential?.kind === 'guest') headers['X-Order-Access-Token'] = credential.token
  if (key) headers['Idempotency-Key'] = key
  const response = await fetch(`${baseUrl}/api/v1/orders${path}`, { method, credentials: 'include', headers, body })
  let payload: Record<string, unknown> | null = null
  try {
    payload = parseExactJSON(await response.text()) as Record<string, unknown>
  } catch {
    /* Keep ambiguous responses retryable. */
  }
  if (!response.ok) {
    const error = payload?.error as { code?: string; field?: string; message?: string } | undefined
    throw new OrderApiError(response.status, error?.code || '', error?.field || '', error?.message || 'Không thể xử lý đơn hàng. Vui lòng thử lại.')
  }
  if ((response.status !== 200 && response.status !== 201) || !payload?.data) {
    throw new OrderApiError(response.status, 'INVALID_RESPONSE', '', 'Chưa nhận được kết quả đơn hàng rõ ràng. Vui lòng tra cứu lại.')
  }
  return payload
}
function count(value: unknown): number {
  const parsed = Number(value)
  if (!Number.isSafeInteger(parsed) || parsed < 0) throw new Error('Dữ liệu số lượng từ máy chủ không hợp lệ')
  return parsed
}
function invalidResponse(): never {
  throw new OrderApiError(200, 'INVALID_RESPONSE', '', 'Kết quả đơn hàng chưa đầy đủ. Vui lòng tra cứu lại.')
}
function validTotals(data: Quote | Order): boolean {
  return (
    [data?.subtotal, data?.shipping_fee, data?.total].every((value) => typeof value === 'string' && /^\d+(?:\.\d+)?$/.test(value)) &&
    Array.isArray(data?.items) &&
    data.items.length > 0 &&
    data.items.every(
      (item) =>
        typeof item.product_id === 'string' &&
        typeof item.sku === 'string' &&
        typeof item.unit_price === 'string' &&
        /^\d+(?:\.\d+)?$/.test(item.unit_price) &&
        count(item.quantity) > 0,
    )
  )
}
function quoteData(raw: unknown): Quote {
  const data = raw as Quote
  if (!validTotals(data) || typeof data.quote_token !== 'string' || !data.quote_token || !Number.isFinite(Date.parse(data.expires_at))) invalidResponse()
  return { ...data, items: data.items.map((item) => ({ ...item, quantity: count(item.quantity) })) }
}
function orderData(raw: unknown): Order {
  const data = raw as Order
  if (
    !validTotals(data) ||
    typeof data.id !== 'string' ||
    !data.id ||
    typeof data.order_number !== 'string' ||
    !data.order_number ||
    !data.shipping_address ||
    typeof data.status !== 'string' ||
    typeof data.payment_status !== 'string'
  )
    invalidResponse()
  return { ...data, items: data.items.map((item) => ({ ...item, quantity: count(item.quantity) })) }
}
export const ordersApi = {
  quote: async (body: CheckoutBody) => quoteData((await request('/quote', 'POST', JSON.stringify(body))).data),
  create: async (attempt: CheckoutAttempt) => orderData((await request('', 'POST', attempt.body, attempt.credential, attempt.request_id)).data),
  confirmReceived: async (id: string, credential: OrderCredential) =>
    orderData((await request(`/${encodeURIComponent(id)}/received`, 'POST', undefined, credential)).data),
  get: async (id: string, credential: OrderCredential) => orderData((await request(`/${encodeURIComponent(id)}`, 'GET', undefined, credential)).data),
  attempt: async (attempt: CheckoutAttempt) =>
    orderData((await request(`/attempts/${encodeURIComponent(attempt.request_id)}`, 'GET', undefined, attempt.credential)).data),
  list: async (page = 1, pageSize = 20, status = ''): Promise<OrderPage> => {
    const query = new URLSearchParams({ page: String(page), page_size: String(pageSize) })
    if (status) query.set('status', status)
    const payload = await request(`?${query}`)
    return { data: (payload.data as unknown[]).map(orderData), total: count(payload.total), page: count(payload.page), page_size: count(payload.page_size) }
  },
}
