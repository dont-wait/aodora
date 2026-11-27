import { ordersApi, OrderApiError } from './api.ts'
import { formatOrderMoney } from './money.ts'
import { authApi, AuthApiError } from '../auth/api.ts'
import type { CheckoutAttempt, CheckoutBody, OrderCredential, OrderReceipt, PurchaseLine, Quote } from './types.ts'

type StorageAccess = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>
const attemptKey = 'aodora.orders.attempt.v1'
const receiptKey = (id: string) => `aodora.orders.receipt.v1.${id}`
const tokenKey = (id: string) => `aodora.orders.guest.v1.${id}`

export function createGuestToken(random: Pick<Crypto, 'getRandomValues'> = crypto): string {
  const bytes = random.getRandomValues(new Uint8Array(32))
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}
export function matchesIdentity(credential: OrderCredential, customerId: string | null): boolean {
  return credential.kind === 'guest' ? customerId === null : credential.customerId === customerId
}
export function makeAttempt(body: CheckoutBody, snapshot: PurchaseLine[], customerId: string | null): CheckoutAttempt {
  const credential: OrderCredential = customerId ? { kind: 'account', customerId } : { kind: 'guest', token: createGuestToken() }
  return { version: 1, request_id: crypto.randomUUID(), credential, body: JSON.stringify(body), snapshot: snapshot.map((line) => ({ ...line })) }
}
export function saveAttempt(attempt: CheckoutAttempt, storage: StorageAccess = localStorage): void {
  if (storage.getItem(attemptKey)) throw new Error('Lần đặt trước chưa có kết quả. Vui lòng tra cứu lại trước khi đặt mới.')
  storage.setItem(attemptKey, JSON.stringify(attempt))
}
export function readAttempt(storage: StorageAccess = localStorage): CheckoutAttempt | null {
  const raw = storage.getItem(attemptKey)
  if (!raw) return null
  try {
    const value = JSON.parse(raw) as CheckoutAttempt
    const body = JSON.parse(value.body) as CheckoutBody
    if (
      value.version !== 1 ||
      !/^[0-9a-f-]{36}$/.test(value.request_id) ||
      !Array.isArray(value.snapshot) ||
      !value.snapshot.length ||
      !Array.isArray(body.items) ||
      !body.items.length ||
      !body.quote_token ||
      !value.snapshot.every(
        (line) => typeof line.product_id === 'string' && typeof line.sku === 'string' && Number.isSafeInteger(line.quantity) && line.quantity > 0,
      ) ||
      !(value.credential?.kind === 'account' ? !!value.credential.customerId : value.credential?.kind === 'guest' && /^[\w-]{43}$/.test(value.credential.token))
    )
      throw new Error()
    return value
  } catch {
    throw new Error('Không đọc được đơn bạn vừa gửi. Vui lòng giữ nguyên dữ liệu trình duyệt và liên hệ cửa hàng trước khi đặt lại.')
  }
}
export function clearAttempt(requestId: string, storage: StorageAccess = localStorage): void {
  if (readAttempt(storage)?.request_id === requestId) storage.removeItem(attemptKey)
}
export function saveOrderToken(id: string, token: string, storage: StorageAccess = localStorage): void {
  storage.setItem(tokenKey(id), token)
}
export function saveReceipt(receipt: OrderReceipt, storage: StorageAccess = localStorage): void {
  // Explicit allowlist: never persist an Order, contact fields or credentials,
  // even when the caller passes extra properties at runtime.
  const safe: OrderReceipt = {
    order_id: receipt.order_id,
    order_number: receipt.order_number,
    request_id: receipt.request_id,
    owner: receipt.owner,
    cleanup: receipt.cleanup,
  }
  storage.setItem(receiptKey(safe.order_id), JSON.stringify(safe))
}
export function readReceipt(id: string, storage: StorageAccess = localStorage): OrderReceipt | null {
  const raw = storage.getItem(receiptKey(id))
  if (!raw) return null
  const value = JSON.parse(raw) as OrderReceipt & {
    order?: { id?: string; order_number?: string }
    credential?: OrderCredential
  }
  const legacy = value.order !== undefined
  const orderId = legacy ? value.order?.id : value.order_id
  const orderNumber = legacy ? value.order?.order_number : value.order_number
  const owner = legacy ? (value.credential?.kind === 'account' ? value.credential.customerId : null) : value.owner
  if (
    orderId !== id ||
    typeof orderNumber !== 'string' ||
    typeof value.request_id !== 'string' ||
    !(owner === null || typeof owner === 'string') ||
    !['pending', 'started', 'done', 'failed'].includes(value.cleanup)
  ) {
    if (legacy) storage.removeItem(receiptKey(id))
    return null
  }
  const safe: OrderReceipt = { order_id: id, order_number: orderNumber, request_id: value.request_id, owner, cleanup: value.cleanup }
  if (legacy && value.credential?.kind === 'guest' && !storage.getItem(tokenKey(id))) {
    saveOrderToken(id, value.credential.token, storage)
  }
  // Migrate old receipts in place, removing their PII and embedded credential.
  // Also strip unexpected properties from metadata-only receipts.
  if (raw !== JSON.stringify(safe)) saveReceipt(safe, storage)
  return safe
}
export function readOrderToken(id: string, storage: StorageAccess = localStorage): string | null {
  return storage.getItem(tokenKey(id))
}

// Subtract only purchased quantities from the latest cart, preserving all other
// fields, newly added lines and any quantity added while checkout was in flight.
export function subtractPurchased<T extends PurchaseLine>(current: T[], snapshot: PurchaseLine[]): T[] {
  const bought = new Map<string, number>()
  for (const line of snapshot) {
    const key = JSON.stringify([line.product_id, line.sku])
    bought.set(key, (bought.get(key) || 0) + line.quantity)
  }
  return current.flatMap((line) => {
    const remaining = line.quantity - (bought.get(JSON.stringify([line.product_id, line.sku])) || 0)
    return remaining > 0 ? [{ ...line, quantity: remaining }] : []
  })
}
export function isDefinitiveOrderError(error: unknown): boolean {
  return (
    error instanceof OrderApiError &&
    error.field !== 'X-Order-Access-Token' &&
    error.status >= 400 &&
    error.status < 500 &&
    ['INVALID_BODY', 'INVALID_FIELD', 'OUT_OF_STOCK', 'QUOTE_CHANGED', 'QUOTE_EXPIRED', 'IDEMPOTENCY_CONFLICT'].includes(error.code)
  )
}
export class CheckoutIdentityError extends Error {}

export async function verifyCheckoutIdentity(credential: OrderCredential, customerId: string | null) {
  if (!matchesIdentity(credential, customerId)) throw new CheckoutIdentityError('Vui lòng dùng đúng tài khoản đã gửi đơn trước đó (hoặc đăng xuất nếu bạn đã đặt khi chưa đăng nhập).')
  let actualId: string | null
  try {
    actualId = (await authApi.me()).id
  } catch (error) {
    if (error instanceof AuthApiError && error.status === 401) actualId = null
    else throw error
  }
  if (!matchesIdentity(credential, actualId))
    throw new CheckoutIdentityError('Trạng thái đăng nhập đã thay đổi. Vui lòng tải lại trang và dùng đúng tài khoản đã gửi đơn.')
}
export async function sendAttempt(attempt: CheckoutAttempt, customerId: string | null) {
  await verifyCheckoutIdentity(attempt.credential, customerId)
  return ordersApi.create(attempt)
}

export async function lookupAttempt(attempt: CheckoutAttempt, customerId: string | null) {
  if (!matchesIdentity(attempt.credential, customerId)) throw new Error('Vui lòng dùng đúng tài khoản đã gửi đơn trước đó (hoặc đăng xuất nếu bạn đã đặt khi chưa đăng nhập) để kiểm tra đơn.')
  await verifyCheckoutIdentity(attempt.credential, customerId)
  try {
    return await ordersApi.attempt(attempt)
  } catch (error) {
    if (error instanceof OrderApiError && error.status === 404) return null
    throw error
  }
}
export function quoteDifferences(previous: Quote | null, next: Quote): string[] {
  if (!previous) return []
  const changes: string[] = []
  if (previous.subtotal !== next.subtotal) changes.push(`Tiền hàng: ${formatOrderMoney(previous.subtotal)} → ${formatOrderMoney(next.subtotal)}.`)
  if (previous.shipping_fee !== next.shipping_fee)
    changes.push(`Phí giao hàng: ${formatOrderMoney(previous.shipping_fee)} → ${formatOrderMoney(next.shipping_fee)}.`)
  if (previous.total !== next.total) changes.push(`Tổng tiền: ${formatOrderMoney(previous.total)} → ${formatOrderMoney(next.total)}.`)
  for (const item of next.items) {
    const old = previous.items.find((line) => line.product_id === item.product_id && line.sku === item.sku)
    if (!old) changes.push(`Thêm dòng ${item.sku}: ${item.quantity} áo, đơn giá ${formatOrderMoney(item.unit_price)}.`)
    else {
      if (old.quantity !== item.quantity) changes.push(`${item.sku}: số lượng ${old.quantity} → ${item.quantity}.`)
      if (old.unit_price !== item.unit_price) changes.push(`${item.sku}: đơn giá ${formatOrderMoney(old.unit_price)} → ${formatOrderMoney(item.unit_price)}.`)
    }
  }
  if (previous.items.length !== next.items.length) changes.push('Danh sách áo đã thay đổi.')
  return changes
}
