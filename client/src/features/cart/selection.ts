import type { CartLine } from './api.ts'

type StorageAccess = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

// Stores the lines the customer UNticked, so new cart lines are selected by
// default and removed lines simply drop out. Holds only product_id/sku keys.
const UNSELECTED_KEY = 'aodora.cart.unselected.v1'

export const cartLineKey = (line: Pick<CartLine, 'product_id' | 'sku'>) => JSON.stringify([line.product_id, line.sku])

export function readUnselected(storage: StorageAccess = localStorage): Set<string> {
  try {
    const raw = JSON.parse(storage.getItem(UNSELECTED_KEY) || '[]') as unknown
    return new Set(Array.isArray(raw) ? raw.filter((key): key is string => typeof key === 'string') : [])
  } catch {
    return new Set()
  }
}

export function writeUnselected(keys: Set<string>, storage: StorageAccess = localStorage): void {
  if (keys.size) storage.setItem(UNSELECTED_KEY, JSON.stringify([...keys]))
  else storage.removeItem(UNSELECTED_KEY)
}

export function selectedLines<T extends Pick<CartLine, 'product_id' | 'sku'>>(items: T[], unselected: Set<string>): T[] {
  return items.filter((item) => !unselected.has(cartLineKey(item)))
}

export function selectedSubtotal(items: Pick<CartLine, 'line_total'>[]): number {
  return items.reduce((sum, item) => sum + item.line_total, 0)
}
