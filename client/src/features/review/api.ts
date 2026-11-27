import type { CatalogReviewPage } from '@/features/catalog/types'

export type ReviewImage = { src: string; alt: string }

export type OwnReview = {
  id: string
  rating: number
  content?: string
  display_name: string
  images: ReviewImage[]
  verified_purchase: boolean
  created_at: string
  order_id: string
  /** Hidden by the shop: still visible to its author only. */
  hidden: boolean
}

export type ProductReviewStatus = { product_id: string; slug: string; reviewed: boolean }
export type EligibleOrder = { id: string; order_number: string; created_at: string }
export type ReviewBody = { rating: number; content: string; images: ReviewImage[] }

export class ReviewApiError extends Error {
  status: number
  code: string
  field: string
  constructor(status: number, field: string, message: string, code = '') { super(message); this.name = 'ReviewApiError'; this.status = status; this.field = field; this.code = code }
}

const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function request<T>(path: string, method = 'GET', input?: unknown): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, { method, credentials: 'include', headers: { Accept: 'application/json', ...(input === undefined ? {} : { 'Content-Type': 'application/json' }) }, body: input === undefined ? undefined : JSON.stringify(input) })
  const body = await response.json().catch(() => null) as { data?: T; error?: { code?: string; field?: string; message?: string } } | null
  if (!response.ok) throw new ReviewApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể xử lý đánh giá', body?.error?.code || '')
  return body?.data as T
}

/** One image per line: `URL | mô tả`. Throws a user-facing message on bad lines. */
export function parseReviewImages(text: string): ReviewImage[] {
  return text.split('\n').map((line) => line.trim()).filter(Boolean).map((line) => {
    const separator = line.indexOf('|')
    if (separator < 1 || !line.slice(separator + 1).trim()) throw new Error('Mỗi ảnh cần theo dạng URL | mô tả ảnh')
    return { src: line.slice(0, separator).trim(), alt: line.slice(separator + 1).trim() }
  })
}

export function formatReviewImages(images: ReviewImage[]): string {
  return images.map((image) => `${image.src} | ${image.alt}`).join('\n')
}

export const reviewApi = {
  list: (slug: string, page = 1) => request<CatalogReviewPage>(`/api/v1/products/${encodeURIComponent(slug)}/reviews?page=${page}&page_size=6`),
  own: (productId: string) => request<OwnReview | null>(`/api/v1/me/reviews?product_id=${encodeURIComponent(productId)}`),
  eligibleOrders: (productId: string) => request<EligibleOrder[]>(`/api/v1/me/reviews/eligible-orders?product_id=${encodeURIComponent(productId)}`),
  statuses: (productIds: string[]) => {
    const query = new URLSearchParams(productIds.map((id) => ['product_id', id]))
    return request<ProductReviewStatus[]>(`/api/v1/me/reviews/status?${query}`)
  },
  submit: (productId: string, input: ReviewBody & { order_id: string }) => request<OwnReview>(`/api/v1/me/reviews/${encodeURIComponent(productId)}`, 'PUT', input),
}

export function reviewErrorMessage(error: unknown): string {
  if (error instanceof ReviewApiError && error.status === 401) return 'Vui lòng đăng nhập để đánh giá.'
  if (error instanceof ReviewApiError && error.status === 403) return 'Bạn chỉ có thể đánh giá sản phẩm trong đơn hàng đã nhận.'
  return error instanceof Error ? error.message : 'Không thể lưu đánh giá'
}
