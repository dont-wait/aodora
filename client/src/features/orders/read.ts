import { ordersApi, OrderApiError } from './api.ts'
import type { OrderPage } from './types.ts'

export type OrderListState = { kind: 'ready'; page: OrderPage } | { kind: 'unauthenticated' | 'error' }

export async function loadMyOrders(page: number, pageSize: number): Promise<OrderListState> {
  try {
    return { kind: 'ready', page: await ordersApi.list(page, pageSize) }
  } catch (error) {
    if (error instanceof OrderApiError && error.status === 401) return { kind: 'unauthenticated' }
    return { kind: 'error' }
  }
}

// Shared subscription for detail/list reads. Cleanup invalidates the prior
// request so an id, account or page change cannot display a late response.
export function startOrderRead<T>(load: () => Promise<T>, receive: (result: T) => void): () => void {
  let active = true
  void load().then((result) => {
    if (active) receive(result)
  })
  return () => {
    active = false
  }
}

export function readOrdersPage(value: string | null): number {
  if (!value || !/^\d+$/.test(value)) return 1
  const page = Number(value)
  return Number.isSafeInteger(page) && page >= 1 && page <= 100000 ? page : 1
}
export function orderPageCount(page: Pick<OrderPage, 'total' | 'page_size'>): number {
  return Math.max(1, Math.ceil(page.total / page.page_size))
}
