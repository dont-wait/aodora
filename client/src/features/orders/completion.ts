import { clearAttempt, readReceipt, saveOrderToken, saveReceipt } from './checkout.ts'
import type { CheckoutAttempt, Order, PurchaseLine } from './types.ts'

// Persist confirmation before clearing an ambiguous attempt. Cart failures must
// never turn a confirmed order into a failed order or create another checkout.
export async function completeCheckout(order: Order, attempt: CheckoutAttempt, consume: (snapshot: PurchaseLine[], ownerId: string | null) => Promise<void>) {
  if (attempt.credential.kind === 'guest') saveOrderToken(order.id, attempt.credential.token)
  let receipt = readReceipt(order.id)
  if (!receipt) {
    receipt = {
      order_id: order.id,
      order_number: order.order_number,
      request_id: attempt.request_id,
      owner: attempt.credential.kind === 'account' ? attempt.credential.customerId : null,
      cleanup: 'pending',
    }
    saveReceipt(receipt)
  }
  if (receipt.cleanup === 'pending') {
    // At most one cleanup pass across recovery/remounts. An interrupted pass
    // leaves an explicit warning rather than subtracting purchased units twice.
    receipt = { ...receipt, cleanup: 'started' }
    saveReceipt(receipt)
    clearAttempt(attempt.request_id)
    try {
      await consume(attempt.snapshot, attempt.credential.kind === 'account' ? attempt.credential.customerId : null)
      receipt = { ...receipt, cleanup: 'done' }
    } catch {
      receipt = { ...receipt, cleanup: 'failed' }
    }
    try {
      saveReceipt(receipt)
    } catch {
      /* Receipt + guest credential were saved before cleanup. */
    }
  } else clearAttempt(attempt.request_id)
  return receipt
}
