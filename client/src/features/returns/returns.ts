/** Return requests: customers ask within 3 days of delivery with photo/video evidence; the shop reviews. */

export type ReturnStatus = 'requested' | 'approved' | 'rejected' | 'collecting' | 'collected' | 'refunded' | 'cancelled'
export type ReturnReason = 'wrong_size' | 'defect' | 'not_as_described' | 'wrong_item' | 'changed_mind' | 'other'
export type Evidence = { url: string; kind: 'image' | 'video'; content_type?: string }
export type ReturnItem = { product_id: string; product_name: string; sku: string; size?: string; color?: string; quantity: number; unit_price: number }
export type ReturnEvent = { status: ReturnStatus; at: string; by: 'customer' | 'admin'; note?: string }
export type ReturnRequest = {
  id: string
  order_id: string
  order_number: string
  customer_id: string
  customer_name?: string
  phone?: string
  items: ReturnItem[]
  reason: ReturnReason
  note?: string
  evidence: Evidence[]
  status: ReturnStatus
  refund_amount: number
  restocked: boolean
  admin_note?: string
  history: ReturnEvent[]
  delivered_at: string
  created_at: string
  updated_at: string
}
export type AdminReturn = ReturnRequest & { allowed_next: ReturnStatus[] }
export type Eligibility = { eligible: boolean; reason?: string; deadline?: string; requests: ReturnRequest[] }

export const RETURN_WINDOW_HOURS = 72
export const MAX_EVIDENCE = 6
export const MAX_VIDEOS = 2
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024
export const MAX_VIDEO_BYTES = 50 * 1024 * 1024
export const EVIDENCE_ACCEPT = 'image/jpeg,image/png,image/webp,video/mp4,video/quicktime,video/webm'

export const returnStatusLabels: Record<ReturnStatus, string> = {
  requested: 'Chờ cửa hàng duyệt',
  approved: 'Đã duyệt, chờ thu hồi',
  collecting: 'Đang tiến hành thu hồi',
  collected: 'Đã thu hồi',
  refunded: 'Đã hoàn tiền',
  rejected: 'Bị từ chối',
  cancelled: 'Đã huỷ yêu cầu',
}
export const returnStatusTone: Record<ReturnStatus, 'neutral' | 'success' | 'warning' | 'danger' | 'info'> = {
  requested: 'warning', approved: 'info', collecting: 'info', collected: 'info', refunded: 'success', rejected: 'danger', cancelled: 'neutral',
}
/** Customer-facing progress steps (rejected/cancelled end the line). */
export const RETURN_STEPS: ReturnStatus[] = ['requested', 'approved', 'collecting', 'collected', 'refunded']

export const reasonLabels: Record<ReturnReason, string> = {
  wrong_size: 'Không vừa size',
  defect: 'Áo bị lỗi (rách, bung chỉ, ố màu…)',
  not_as_described: 'Không giống mô tả / hình ảnh',
  wrong_item: 'Giao sai mẫu, màu hoặc size',
  changed_mind: 'Đổi ý, không muốn dùng nữa',
  other: 'Lý do khác',
}
export const RETURN_REASONS = Object.keys(reasonLabels) as ReturnReason[]

/** What the admin button says for each next status. */
export const adminActionLabels: Record<ReturnStatus, string> = {
  approved: 'Duyệt yêu cầu',
  rejected: 'Từ chối',
  collecting: 'Bắt đầu thu hồi',
  collected: 'Xác nhận đã thu hồi',
  refunded: 'Xác nhận đã hoàn tiền',
  requested: '',
  cancelled: '',
}

export function returnDeadline(deliveredAt: string): Date {
  return new Date(Date.parse(deliveredAt) + RETURN_WINDOW_HOURS * 3600_000)
}

/** "còn 2 ngày 5 giờ" / "còn 40 phút" / "" when past. */
export function timeLeft(deadline: string | Date, now = Date.now()): string {
  const ms = (typeof deadline === 'string' ? Date.parse(deadline) : deadline.getTime()) - now
  if (!Number.isFinite(ms) || ms <= 0) return ''
  const minutes = Math.floor(ms / 60000)
  const days = Math.floor(minutes / 1440)
  const hours = Math.floor((minutes % 1440) / 60)
  if (days > 0) return `còn ${days} ngày${hours ? ` ${hours} giờ` : ''}`
  if (hours > 0) return `còn ${hours} giờ`
  return `còn ${Math.max(1, minutes)} phút`
}

export function evidenceKindOf(file: { type: string; name: string }): 'image' | 'video' | null {
  if (/^image\/(jpeg|png|webp)$/.test(file.type)) return 'image'
  if (/^video\/(mp4|quicktime|webm)$/.test(file.type)) return 'video'
  const ext = file.name.toLowerCase().split('.').pop() ?? ''
  if (['jpg', 'jpeg', 'png', 'webp'].includes(ext)) return 'image'
  if (['mp4', 'mov', 'webm'].includes(ext)) return 'video'
  return null
}

/** Checks one picked file against what is already attached; returns an error message or ''. */
export function evidenceProblem(file: { type: string; name: string; size: number }, attached: Pick<Evidence, 'kind'>[]): string {
  const kind = evidenceKindOf(file)
  if (!kind) {
    if (/heic|heif/i.test(file.type) || /\.(heic|heif)$/i.test(file.name)) return `${file.name}: ảnh HEIC chưa được hỗ trợ, hãy chụp ở chế độ JPG hoặc gửi video.`
    return `${file.name}: chỉ nhận ảnh JPG, PNG, WebP hoặc video MP4, MOV, WebM.`
  }
  if (attached.length >= MAX_EVIDENCE) return `Tối đa ${MAX_EVIDENCE} ảnh/video.`
  if (kind === 'image' && file.size > MAX_IMAGE_BYTES) return `${file.name}: ảnh tối đa 10 MB.`
  if (kind === 'video' && file.size > MAX_VIDEO_BYTES) return `${file.name}: video tối đa 50 MB (khoảng 30 giây).`
  if (kind === 'video' && attached.filter((e) => e.kind === 'video').length >= MAX_VIDEOS) return `Tối đa ${MAX_VIDEOS} video.`
  return ''
}

export type ReturnDraft = { quantities: Record<string, number>; reason: ReturnReason | ''; note: string; evidence: Evidence[] }

/** Field → message; empty when the draft can be sent. */
export function validateReturnDraft(draft: ReturnDraft): Record<string, string> {
  const errors: Record<string, string> = {}
  if (!Object.values(draft.quantities).some((q) => q > 0)) errors.items = 'Chọn ít nhất một áo cần hoàn.'
  if (!draft.reason) errors.reason = 'Chọn lý do hoàn hàng.'
  const note = draft.note.trim()
  if (draft.reason === 'other' && note.length < 10) errors.note = 'Hãy mô tả lý do (ít nhất 10 ký tự).'
  if (note.length > 1000) errors.note = 'Mô tả tối đa 1000 ký tự.'
  if (draft.evidence.length === 0) errors.evidence = 'Cần ít nhất 1 ảnh hoặc 1 video minh chứng.'
  return errors
}

export function draftBody(orderId: string, draft: ReturnDraft) {
  return {
    order_id: orderId,
    items: Object.entries(draft.quantities).filter(([, q]) => q > 0).map(([sku, quantity]) => ({ sku, quantity })),
    reason: draft.reason,
    note: draft.note.trim(),
    evidence: draft.evidence.map((e) => ({ url: e.url })),
  }
}

export class ReturnApiError extends Error {
  status: number
  code: string
  field: string
  constructor(status: number, code: string, field: string, message: string) {
    super(message)
    this.name = 'ReturnApiError'
    this.status = status
    this.code = code
    this.field = field
  }
}

const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

async function call<T>(path: string, init: RequestInit = {}): Promise<{ data: T; meta?: { total?: number } }> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (typeof init.body === 'string') headers['Content-Type'] = 'application/json'
  let response: Response
  try {
    response = await fetch(`${baseUrl}${path}`, { credentials: 'include', ...init, headers })
  } catch {
    throw new ReturnApiError(0, 'NETWORK', '', 'Không kết nối được máy chủ. Kiểm tra mạng rồi thử lại.')
  }
  const payload = (await response.json().catch(() => null)) as { data?: T; meta?: { total?: number }; error?: { code?: string; field?: string; message?: string } } | null
  if (!response.ok) {
    throw new ReturnApiError(response.status, payload?.error?.code || '', payload?.error?.field || '', payload?.error?.message || 'Không xử lý được yêu cầu hoàn hàng. Vui lòng thử lại.')
  }
  return { data: payload?.data as T, meta: payload?.meta }
}

export const returnsApi = {
  eligibility: async (orderId: string) => (await call<Eligibility>(`/api/v1/me/returns/eligibility?order_id=${encodeURIComponent(orderId)}`)).data,
  mine: async () => (await call<ReturnRequest[]>('/api/v1/me/returns')).data ?? [],
  create: async (body: ReturnType<typeof draftBody>) => (await call<ReturnRequest>('/api/v1/me/returns', { method: 'POST', body: JSON.stringify(body) })).data,
  cancel: async (id: string) => (await call<ReturnRequest>(`/api/v1/me/returns/${encodeURIComponent(id)}/cancel`, { method: 'POST' })).data,
  uploadEvidence: async (file: File): Promise<Evidence> => {
    const form = new FormData()
    form.append('file', file)
    return (await call<Evidence>('/api/v1/me/returns/evidence', { method: 'POST', body: form })).data
  },
}

export const adminReturnsApi = {
  list: async (filter: { status?: string; page?: number; size?: number; order_id?: string }) => {
    const q = new URLSearchParams()
    if (filter.status) q.set('status', filter.status)
    if (filter.order_id) q.set('order_id', filter.order_id)
    q.set('page', String(filter.page ?? 1))
    q.set('size', String(filter.size ?? 20))
    const res = await call<AdminReturn[]>(`/api/v1/admin/returns?${q}`)
    return { data: res.data ?? [], total: res.meta?.total ?? 0 }
  },
  get: async (id: string) => (await call<AdminReturn>(`/api/v1/admin/returns/${encodeURIComponent(id)}`)).data,
  transition: async (id: string, input: { to: ReturnStatus; note?: string; restock?: boolean }) =>
    (await call<AdminReturn>(`/api/v1/admin/returns/${encodeURIComponent(id)}/transition`, { method: 'POST', body: JSON.stringify(input) })).data,
}
