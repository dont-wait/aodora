import { readOrdersPage } from '../orders/read.ts'

export type AdminReview = {
  id: string
  rating: number
  content?: string
  display_name: string
  images: { src: string; alt: string }[]
  verified_purchase: boolean
  created_at: string
  updated_at: string
  product_id: string
  product_name: string
  product_slug: string
  order_number?: string
  hidden: boolean
}
export type AdminReviewPage = { data: AdminReview[]; total: number; page: number; page_size: number }
export type AdminReviewVisibility = '' | 'visible' | 'hidden'
export type AdminReviewFilter = { rating: number; visibility: AdminReviewVisibility; page: number; page_size: number }
export type AdminReviewFailure = { kind: 'unauthenticated' | 'forbidden' | 'not-found' | 'error'; message: string }
export type AdminReviewListRead = { kind: 'ready'; page: AdminReviewPage } | AdminReviewFailure
export type AdminReviewAction = 'hide' | 'show' | 'delete'
export type AdminReviewActionResult = { kind: 'done' } | AdminReviewFailure

/** Unknown or repeated query values never reach the API. */
export function readAdminReviewFilters(input: URLSearchParams): AdminReviewFilter {
  const unique = (key: string) => (input.getAll(key).length === 1 ? input.get(key) || '' : '')
  const rating = Number(unique('rating'))
  const sizeText = unique('page_size')
  const size = /^\d+$/.test(sizeText) ? Number(sizeText) : 20
  const visibility = unique('visibility')
  return {
    rating: Number.isInteger(rating) && rating >= 1 && rating <= 5 ? rating : 0,
    visibility: visibility === 'visible' || visibility === 'hidden' ? visibility : '',
    page: readOrdersPage(unique('page')),
    page_size: Number.isSafeInteger(size) && size >= 1 ? Math.min(100, size) : 20,
  }
}
export function adminReviewSearch(filter: AdminReviewFilter): URLSearchParams {
  const safe = readAdminReviewFilters(new URLSearchParams({ rating: String(filter.rating), visibility: filter.visibility, page: String(filter.page), page_size: String(filter.page_size) }))
  const params = new URLSearchParams({ page: String(safe.page), page_size: String(safe.page_size) })
  if (safe.rating) params.set('rating', String(safe.rating))
  if (safe.visibility) params.set('visibility', safe.visibility)
  return params
}
/** Page URL uses `visibility`; the API takes `hidden=true|false`. */
export function adminReviewApiQuery(filter: AdminReviewFilter): URLSearchParams {
  const params = adminReviewSearch(filter)
  const visibility = params.get('visibility')
  params.delete('visibility')
  if (visibility) params.set('hidden', String(visibility === 'hidden'))
  return params
}

export class AdminReviewApiError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'AdminReviewApiError'
    this.status = status
    this.code = code
  }
}
const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
function invalid(): never {
  throw new AdminReviewApiError(200, 'INVALID_RESPONSE', 'Dữ liệu đánh giá trả về chưa đầy đủ. Vui lòng tải lại.')
}
function integer(value: unknown, minimum = 0): number {
  const number = Number(value)
  if (!Number.isSafeInteger(number) || number < minimum) invalid()
  return number
}
function review(raw: unknown): AdminReview {
  const data = raw as AdminReview
  if (
    !data || typeof data.id !== 'string' || !Number.isInteger(data.rating) || data.rating < 1 || data.rating > 5 ||
    typeof data.display_name !== 'string' || typeof data.product_id !== 'string' || !Array.isArray(data.images) || typeof data.hidden !== 'boolean'
  )
    invalid()
  return data
}
async function request(path: string, method = 'GET', input?: unknown): Promise<Record<string, unknown>> {
  const response = await fetch(`${baseUrl}/api/v1/admin/reviews${path}`, {
    method,
    credentials: 'include',
    headers: { Accept: 'application/json', ...(input === undefined ? {} : { 'Content-Type': 'application/json' }) },
    body: input === undefined ? undefined : JSON.stringify(input),
  })
  const payload = (await response.json().catch(() => null)) as Record<string, unknown> | null
  if (!response.ok) {
    const error = payload?.error as { code?: string; message?: string } | undefined
    throw new AdminReviewApiError(response.status, error?.code || '', error?.message || 'Không thể xử lý đánh giá. Vui lòng thử lại.')
  }
  if (response.status !== 200 || !payload?.data) invalid()
  return payload
}
export const adminReviews = {
  list: async (filter: AdminReviewFilter): Promise<AdminReviewPage> => {
    const payload = await request(`?${adminReviewApiQuery(filter)}`)
    if (!Array.isArray(payload.data)) invalid()
    return { data: payload.data.map(review), total: integer(payload.total), page: integer(payload.page, 1), page_size: integer(payload.page_size, 1) }
  },
  setHidden: async (id: string, hidden: boolean): Promise<void> => {
    await request(`/${encodeURIComponent(id)}`, 'PATCH', { hidden })
  },
  remove: async (id: string): Promise<void> => {
    await request(`/${encodeURIComponent(id)}`, 'DELETE')
  },
}

function failure(error: unknown, message: string): AdminReviewFailure {
  if (error instanceof AdminReviewApiError) {
    if (error.status === 401) return { kind: 'unauthenticated', message: 'Vui lòng đăng nhập lại.' }
    if (error.status === 403) return { kind: 'forbidden', message: 'Bạn không có quyền quản trị đánh giá.' }
    if (error.status === 404) return { kind: 'not-found', message: 'Đánh giá không còn tồn tại.' }
  }
  return { kind: 'error', message }
}
export async function loadAdminReviews(filter: AdminReviewFilter): Promise<AdminReviewListRead> {
  try {
    return { kind: 'ready', page: await adminReviews.list(filter) }
  } catch (error) {
    return failure(error, 'Không thể tải danh sách đánh giá. Vui lòng thử lại.')
  }
}
export async function performAdminReviewAction(id: string, action: AdminReviewAction): Promise<AdminReviewActionResult> {
  try {
    if (action === 'delete') await adminReviews.remove(id)
    else await adminReviews.setHidden(id, action === 'hide')
    return { kind: 'done' }
  } catch (error) {
    return failure(error, action === 'delete' ? 'Không thể xóa đánh giá. Vui lòng thử lại.' : 'Không thể đổi trạng thái hiển thị. Vui lòng thử lại.')
  }
}
export const adminReviewActions: Record<AdminReviewAction, { label: string; confirmation: string; done: string }> = {
  hide: {
    label: 'Ẩn đánh giá',
    confirmation: 'Ẩn đánh giá này khỏi trang sản phẩm. Điểm trung bình và số lượt đánh giá được tính lại ngay, không tính đánh giá bị ẩn. Có thể hiện lại bất cứ lúc nào.',
    done: 'Đã ẩn đánh giá và tính lại điểm của sản phẩm.',
  },
  show: {
    label: 'Hiện lại đánh giá',
    confirmation: 'Hiện lại đánh giá này trên trang sản phẩm. Điểm trung bình và số lượt đánh giá được tính lại ngay.',
    done: 'Đã hiện lại đánh giá và tính lại điểm của sản phẩm.',
  },
  delete: {
    label: 'Xóa đánh giá',
    confirmation: 'Xóa vĩnh viễn đánh giá này. Điểm trung bình và số lượt đánh giá được tính lại ngay. Không thể hoàn tác; nếu chỉ muốn tạm gỡ, hãy dùng "Ẩn đánh giá".',
    done: 'Đã xóa đánh giá và tính lại điểm của sản phẩm.',
  },
}
