export type AuthUser = {
  id: string
  full_name: string
  email: string
  phone?: string
  role: 'customer' | 'admin'
}

export class AuthApiError extends Error {
  status: number
  field: string
  constructor(status: number, field: string, message: string) {
    super(message)
    this.name = 'AuthApiError'
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
  if (response.status === 204) return undefined as T
  const body = await response.json().catch(() => null) as { data?: T; error?: { field?: string; message?: string } } | null
  if (!response.ok) throw new AuthApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể xử lý yêu cầu')
  return body?.data as T
}

export const authApi = {
  me: () => request<AuthUser>('/api/v1/auth/me'),
  register: (input: { full_name: string; email: string; password: string; phone?: string }) => request<AuthUser>('/api/v1/auth/register', 'POST', input),
  login: (input: { email: string; password: string }) => request<AuthUser>('/api/v1/auth/login', 'POST', input),
  logout: () => request<void>('/api/v1/auth/logout', 'POST'),
}
