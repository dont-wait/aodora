export type Address = {
  id: string
  recipient: string
  phone: string
  line1: string
  ward?: string
  /** chỉ đọc dữ liệu trước 07/2025 */
  district?: string
  city: string
  is_default: boolean
}

export type AddressInput = Omit<Address, 'id'>

export type Measurements = {
  body?: string
  neck_cm?: number
  bust_cm?: number
  waist_cm?: number
  hips_cm?: number
  shoulder_cm?: number
  sleeve_cm?: number
  garment_length_cm?: number
}

export type Profile = {
  id: string
  full_name: string
  email: string
  phone?: string
  addresses: Address[]
  measurement_profile?: Measurements
  measurement_notes?: string
}

export type WishlistProduct = {
  id: string
  slug: string
  name: string
  description?: string
  material?: string
  price: number
  image?: { src: string; alt: string }
  stock: number
  ready_available: boolean
  bespoke: boolean
  unavailable?: boolean
}

export type WishlistItem = { id: string; product?: WishlistProduct }

export class AccountApiError extends Error {
  status: number
  field: string
  constructor(status: number, field: string, message: string) {
    super(message)
    this.name = 'AccountApiError'
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
  if (!response.ok) throw new AccountApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể xử lý dữ liệu tài khoản')
  return body?.data as T
}

export const accountApi = {
  profile: () => request<Profile>('/api/v1/me/profile'),
  updateProfile: (input: { full_name: string; phone: string }) => request<Profile>('/api/v1/me/profile', 'PATCH', input),
  createAddress: (input: AddressInput) => request<Profile>('/api/v1/me/addresses', 'POST', input),
  updateAddress: (id: string, input: AddressInput) => request<Profile>(`/api/v1/me/addresses/${encodeURIComponent(id)}`, 'PUT', input),
  deleteAddress: (id: string) => request<Profile>(`/api/v1/me/addresses/${encodeURIComponent(id)}`, 'DELETE'),
  measurements: () => request<{ measurement_profile?: Measurements; measurement_notes?: string }>('/api/v1/me/measurements'),
  updateMeasurements: (input: { measurement_profile: Measurements | null; measurement_notes: string }) => request<{ measurement_profile?: Measurements; measurement_notes?: string }>('/api/v1/me/measurements', 'PUT', input),
  wishlist: () => request<{ items: WishlistItem[] }>('/api/v1/me/wishlist'),
  addWishlist: (productId: string) => request<{ saved: boolean }>(`/api/v1/me/wishlist/${encodeURIComponent(productId)}`, 'PUT'),
  removeWishlist: (productId: string) => request<{ saved: boolean }>(`/api/v1/me/wishlist/${encodeURIComponent(productId)}`, 'DELETE'),
}
