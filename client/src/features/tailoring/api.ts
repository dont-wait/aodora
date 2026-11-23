import type { TailoringReceipt, TailoringRequestBody } from './types.ts'

export class TailoringApiError extends Error {
  status: number
  code: string
  field: string
  constructor(status: number, code: string, field: string, message: string) {
    super(message)
    this.name = 'TailoringApiError'
    this.status = status
    this.code = code
    this.field = field
  }
}

const baseUrl = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

export const tailoringApi = {
  /**
   * Sends a consultation request. Guests and signed-in customers use the same
   * call; credentials: 'include' lets the server link a valid session cookie.
   * Resolves only for a well-formed 201; everything else throws.
   */
  submit: async (body: TailoringRequestBody): Promise<TailoringReceipt> => {
    const response = await fetch(`${baseUrl}/api/v1/tailoring-requests`, {
      method: 'POST',
      credentials: 'include',
      headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    let payload: { data?: { id?: unknown; status?: unknown }; error?: { code?: string; field?: string; message?: string } } | null = null
    try {
      payload = JSON.parse(await response.text())
    } catch {
      /* Treated as ambiguous below. */
    }
    if (!response.ok) {
      const error = payload?.error
      throw new TailoringApiError(response.status, error?.code || '', error?.field || '', error?.message || 'Không thể gửi yêu cầu. Vui lòng thử lại.')
    }
    const data = payload?.data
    if (response.status !== 201 || typeof data?.id !== 'string' || !data.id || typeof data.status !== 'string') {
      throw new TailoringApiError(response.status, 'INVALID_RESPONSE', '', 'Chưa nhận được xác nhận rõ ràng từ cửa hàng. Vui lòng liên hệ cửa hàng trước khi gửi lại.')
    }
    return { id: data.id, status: data.status }
  },
}
