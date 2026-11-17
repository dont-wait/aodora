import { ordersApi, OrderApiError } from './api.ts'
import { readOrderToken, readReceipt } from './checkout.ts'
import type { Order, OrderCredential, OrderReceipt } from './types.ts'

export type CheckoutResult = { kind: 'ready'; order: Order; receipt: OrderReceipt | null; guest: boolean } | { kind: 'unauthenticated' | 'not-found' | 'error' }

// Receipt metadata only controls the cleanup warning. Permission and every
// displayed order field ALWAYS come from GET /orders/{id}, including on reload.
export async function loadCheckoutResult(id: string, customerId: string | null): Promise<CheckoutResult> {
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
