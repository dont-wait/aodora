import { ordersApi, OrderApiError } from './api.ts'
import { readOrderToken, readReceipt } from './checkout.ts'
import type { Order, OrderCredential, OrderReceipt } from './types.ts'

export type CheckoutResult = { kind: 'ready'; order: Order; receipt: OrderReceipt | null; guest: boolean } | { kind: 'unauthenticated' | 'not-found' | 'error' }

// Receipt metadata only controls the cleanup warning. Permission and every
// displayed order field ALWAYS come from GET /orders/{id}, including on reload.
export async function loadOrderDetail(id: string, customerId: string | null): Promise<CheckoutResult> {
  let receipt: OrderReceipt | null = null
  try {
    receipt = readReceipt(id)
  } catch {
    /* Local metadata cannot grant access or prevent an account GET. */
  }
  let credential: OrderCredential
  if (customerId) credential = { kind: 'account', customerId }
  else {
    let token: string | null
    try {
      token = readOrderToken(id)
    } catch {
      return { kind: 'error' }
    }
    if (!token) return { kind: 'not-found' }
    credential = { kind: 'guest', token }
  }
  try {
    const order = await ordersApi.get(id, credential)
    return { kind: 'ready', order, receipt, guest: credential.kind === 'guest' }
  } catch (error) {
    if (error instanceof OrderApiError && error.status === 401) return { kind: 'unauthenticated' }
    if (error instanceof OrderApiError && error.status === 404) return { kind: 'not-found' }
    return { kind: 'error' }
  }
}

// Retain the checkout caller contract while sharing all detail reads.
export const loadCheckoutResult = loadOrderDetail

/** Buyer's "Đã nhận được hàng": marks the order delivered and paid. Throws a user-facing Error. */
export async function confirmOrderReceived(id: string, customerId: string | null): Promise<void> {
  let credential: OrderCredential
  if (customerId) credential = { kind: 'account', customerId }
  else {
    const token = readOrderToken(id)
    if (!token) throw new Error('Trình duyệt này không còn quyền với đơn. Vui lòng liên hệ cửa hàng.')
    credential = { kind: 'guest', token }
  }
  try {
    await ordersApi.confirmReceived(id, credential)
  } catch (error) {
    if (error instanceof OrderApiError && error.status === 409) throw new Error('Đơn chưa ở bước nhận hàng hoặc vừa được cập nhật. Hãy tải lại trang.')
    throw new Error(error instanceof Error ? error.message : 'Không xác nhận được. Vui lòng thử lại.')
  }
}
