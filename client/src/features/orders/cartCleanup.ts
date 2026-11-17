import type { CartData } from '../cart/api.ts'
import type { CartSelection } from '../catalog/selection.ts'
import { subtractPurchased } from './checkout.ts'
import type { PurchaseLine } from './types.ts'

type AccountCartPort = {
  get: () => Promise<CartData>
  update: (selection: CartSelection) => Promise<CartData>
  remove: (productId: string, sku: string) => Promise<CartData>
}

export async function consumeAccountPurchase(snapshot: PurchaseLine[], api: AccountCartPort, changed: (cart: CartData) => void, checkIdentity: () => void) {
  checkIdentity()
  const latest = await api.get()
  checkIdentity()
  // An item can move to issues after its last stock was bought. Include both.
  const lines = [...new Map([...(latest.items || []), ...(latest.issues || [])].map((line) => [JSON.stringify([line.product_id, line.sku]), line])).values()]
  const remaining = subtractPurchased(lines, snapshot)
  const keys = new Set(snapshot.map((line) => JSON.stringify([line.product_id, line.sku])))
  for (const line of lines) {
    if (!keys.has(JSON.stringify([line.product_id, line.sku]))) continue
    checkIdentity()
    const next = remaining.find((item) => item.product_id === line.product_id && item.sku === line.sku)
    const updated = next
      ? await api.update({
          product_id: next.product_id,
          product_slug: next.product_slug,
          sku: next.sku,
          quantity: next.quantity,
          color: next.color,
          size: next.size,
          fulfillment_mode: next.fulfillment_mode,
        })
      : await api.remove(line.product_id, line.sku)
    checkIdentity()
    changed(updated)
  }
}
