import type { CatalogReviewPage } from '@/features/catalog/types'

export type OwnReview = {
  id: string
  rating: number
  content?: string
  display_name: string
  images: { src: string; alt: string }[]
  verified_purchase: boolean
  created_at: string
  order_id: string
}

export type EligibleOrder = { id: string; order_number: string; created_at: string }

export class ReviewApiError extends Error {
  status: number
  field: string
  constructor(status: number, field: string, message: string) { super(message); this.name = 'ReviewApiError'; this.status = status; this.field = field }
}

const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function request<T>(path: string, method = 'GET', input?: unknown): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, { method, credentials: 'include', headers: { Accept: 'application/json', ...(input === undefined ? {} : { 'Content-Type': 'application/json' }) }, body: input === undefined ? undefined : JSON.stringify(input) })
  const body = await response.json().catch(() => null) as { data?: T; error?: { field?: string; message?: string } } | null
  if (!response.ok) throw new ReviewApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể xử lý đánh giá')
  return body?.data as T
}

export const reviewApi = {
  list: (slug: string, page = 1) => request<CatalogReviewPage>(`/api/v1/products/${encodeURIComponent(slug)}/reviews?page=${page}&page_size=6`),
  own: (productId: string) => request<OwnReview | null>(`/api/v1/me/reviews?product_id=${encodeURIComponent(productId)}`),
  eligibleOrders: (productId: string) => request<EligibleOrder[]>(`/api/v1/me/reviews/eligible-orders?product_id=${encodeURIComponent(productId)}`),
  submit: (productId: string, input: { order_id: string; rating: number; content: string; images: { src: string; alt: string }[] }) => request<OwnReview>(`/api/v1/me/reviews/${encodeURIComponent(productId)}`, 'PUT', input),
}
