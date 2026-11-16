import type { CartSelection } from '@/features/catalog/selection'

export type CartLine = CartSelection & {
  name?: string
  image?: string
  unit_price: number
  line_total: number
  stock: number
  available: boolean
  issue?: string
  issue_detail?: string
}

export type CartData = {
  items: CartLine[]
  issues: CartLine[]
  item_count: number
  subtotal: number
}

export class CartApiError extends Error {
  status: number
  field: string
  constructor(status: number, field: string, message: string) {
    super(message)
    this.name = 'CartApiError'
    this.status = status
    this.field = field
  }
}

const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function request<T>(path: string, method = 'GET', input?: unknown): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, {
    method,
    credentials: 'include',
    headers: { Accept: 'application/json', ...(input === undefined ? {} : { 'Content-Type': 'application/json' }) },
    body: input === undefined ? undefined : JSON.stringify(input),
  })
  const body = await response.json().catch(() => null) as { data?: T; error?: { field?: string; message?: string } } | null
  if (!response.ok) throw new CartApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể xử lý giỏ hàng')
  return body?.data as T
}

export const cartApi = {
  get: () => request<CartData>('/api/v1/me/cart'),
  add: (selection: CartSelection) => request<CartData>('/api/v1/me/cart/items', 'POST', selection),
  update: (selection: CartSelection) => request<CartData>(`/api/v1/me/cart/items/${encodeURIComponent(selection.product_id)}/${encodeURIComponent(selection.sku)}`, 'PUT', { quantity: selection.quantity, color: selection.color, size: selection.size, product_slug: selection.product_slug, fulfillment_mode: selection.fulfillment_mode }),
  remove: (productId: string, sku: string) => request<CartData>(`/api/v1/me/cart/items/${encodeURIComponent(productId)}/${encodeURIComponent(sku)}`, 'DELETE'),
  merge: (items: CartSelection[], requestId: string) => request<CartData>('/api/v1/me/cart/merge', 'POST', { request_id: requestId, items }),
  validate: (items: CartSelection[]) => request<CartData>('/api/v1/cart/validate', 'POST', { items }),
}
