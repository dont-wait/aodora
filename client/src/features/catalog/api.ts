import type { CatalogCategory, CatalogFilters, CatalogPage, CatalogProduct } from './types'

// During development Vite proxies /api to the Go backend. In production
// the API can share the web origin or use a configured public base URL.
const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
export class CatalogApiError extends Error {
  status: number
  code?: string
  constructor(message: string, status: number, code?: string) { super(message); this.name = 'CatalogApiError'; this.status = status; this.code = code }
}

async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, { signal, headers: { Accept: 'application/json' } })
  let payload: unknown
  try { payload = await response.json() } catch { throw new CatalogApiError('Máy chủ trả về dữ liệu không hợp lệ.', response.status) }
  if (!response.ok) {
    const body = payload as { error?: { code?: string; message?: string } }
    throw new CatalogApiError(body?.error?.message || 'Không thể tải dữ liệu từ máy chủ.', response.status, body?.error?.code)
  }
  return payload as T
}

export async function getCategories(signal?: AbortSignal): Promise<CatalogCategory[]> {
  const result = await getJSON<{ data: CatalogCategory[] }>('/api/v1/categories', signal)
  return result.data
}

export function listProducts(filters: CatalogFilters = {}, signal?: AbortSignal): Promise<CatalogPage> {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  const query = params.toString()
  return getJSON<CatalogPage>(`/api/v1/products${query ? `?${query}` : ''}`, signal)
}

export async function getProduct(slug: string, signal?: AbortSignal): Promise<CatalogProduct> {
  const result = await getJSON<{ data: CatalogProduct }>(`/api/v1/products/${encodeURIComponent(slug)}`, signal)
  return result.data
}
