// Admin image upload (POST /api/v1/admin/uploads/images) and client-side pre-checks.
import { AdminApiError, type Image } from './catalog.ts'

export const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024
export const MIN_SHORT_SIDE = 400
export const MAX_LONG_SIDE = 8000
export type UploadPurpose = 'product' | 'category' | 'content' | 'accessory'
export type UploadResult = { url: string; key: string; width: number; height: number; size: number; content_type: string }

const apiBase = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

/** First problem with the file itself, or "". The server re-checks everything by magic bytes. */
export function checkImageFile(file: { name: string; type: string; size: number }): string {
  const byExt = /\.(jpe?g|png|webp)$/i.test(file.name)
  const byType = (IMAGE_TYPES as readonly string[]).includes(file.type)
  if (/\.(svg|gif|heic|heif)$/i.test(file.name) || /^image\/(svg|gif|heic|heif)/.test(file.type)) return 'Chỉ nhận ảnh JPG, PNG hoặc WebP (không nhận SVG, GIF, HEIC).'
  if (!byType && !byExt) return 'Chỉ nhận ảnh JPG, PNG hoặc WebP.'
  if (file.size === 0) return 'File trống.'
  if (file.size > MAX_IMAGE_BYTES) return 'Ảnh quá lớn (' + (file.size / 1024 / 1024).toFixed(1).replace('.', ',') + ' MB), tối đa 10 MB.'
  return ''
}

export function checkImageDimensions(width: number, height: number): string {
  if (Math.min(width, height) < MIN_SHORT_SIDE) return 'Ảnh quá nhỏ (' + width + '×' + height + '). Cạnh ngắn cần ít nhất ' + MIN_SHORT_SIDE + ' px.'
  if (Math.max(width, height) > MAX_LONG_SIDE) return 'Ảnh quá lớn (' + width + '×' + height + '). Cạnh dài tối đa ' + MAX_LONG_SIDE + ' px.'
  return ''
}

/** Reads the pixel size in the browser (no upload). */
export function readImageSize(file: File): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => { URL.revokeObjectURL(url); resolve({ width: img.naturalWidth, height: img.naturalHeight }) }
    img.onerror = () => { URL.revokeObjectURL(url); reject(new Error('Không đọc được ảnh, file có thể bị hỏng.')) }
    img.src = url
  })
}

/** Default alt text: "<name> — ảnh 2" (or just the name for a single image). */
export function defaultAlt(name: string, position: number | null): string {
  const base = name.trim() || 'Ảnh sản phẩm'
  return position === null ? base : base + ' — ảnh ' + position
}

/** Moves list[from] to index to (immutable). Out-of-range indexes return the list unchanged. */
export function moveItem<T>(list: readonly T[], from: number, to: number): T[] {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) return [...list]
  const copy = [...list]
  const [item] = copy.splice(from, 1)
  copy.splice(to, 0, item)
  return copy
}

export function uploadedImage(result: UploadResult, alt: string): Image {
  return { src: result.url, alt }
}

/** Upload with progress and cancellation (XHR, because fetch has no upload progress). */
export function uploadImage(file: File, purpose: UploadPurpose, opts: { onProgress?: (fraction: number) => void; signal?: AbortSignal } = {}): Promise<UploadResult> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', apiBase + '/api/v1/admin/uploads/images')
    xhr.withCredentials = true
    xhr.setRequestHeader('Accept', 'application/json')
    xhr.upload.onprogress = (event) => { if (event.lengthComputable) opts.onProgress?.(event.loaded / event.total) }
    xhr.onload = () => {
      let body: { data?: UploadResult; error?: { message?: string; field?: string } } | null = null
      try { body = JSON.parse(xhr.responseText) } catch { /* not JSON */ }
      if (xhr.status >= 200 && xhr.status < 300 && body?.data?.url) resolve(body.data)
      else reject(new AdminApiError(xhr.status, body?.error?.field || '', uploadErrorMessage(xhr.status, body?.error?.message)))
    }
    xhr.onerror = () => reject(new Error('Không kết nối được máy chủ. Kiểm tra mạng rồi thử lại.'))
    xhr.onabort = () => reject(new DOMException('Đã hủy tải ảnh', 'AbortError'))
    opts.signal?.addEventListener('abort', () => xhr.abort(), { once: true })
    if (opts.signal?.aborted) { reject(new DOMException('Đã hủy tải ảnh', 'AbortError')); return }
    const form = new FormData()
    form.append('purpose', purpose)
    form.append('file', file)
    xhr.send(form)
  })
}

export function uploadErrorMessage(status: number, serverMessage?: string): string {
  if (serverMessage) return serverMessage
  if (status === 413) return 'Ảnh quá lớn, tối đa 10 MB.'
  if (status === 503) return 'Kho lưu ảnh chưa sẵn sàng. Vui lòng thử lại sau.'
  if (status === 401 || status === 403) return 'Bạn không có quyền tải ảnh lên.'
  return 'Không thể tải ảnh lên.'
}
