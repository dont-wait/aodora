import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { orderReviewable } from '@/features/orders/presentation'
import type { Order } from '@/features/orders/types'
import { reviewApi, type ProductReviewStatus } from './api'

export type ReviewStatusMap = Map<string, ProductReviewStatus>

/**
 * Loads review status once for every product in the delivered and paid
 * orders of the current page. Returns null while loading or on failure,
 * so buttons stay disabled rather than linking somewhere wrong.
 */
export function useOrderReviewStatuses(orders: Order[]): ReviewStatusMap | null {
  const productIds = useMemo(() => [...new Set(orders.filter(orderReviewable).flatMap((order) => order.items.map((item) => item.product_id)))].sort(), [orders])
  const key = productIds.join(',')
  const [loaded, setLoaded] = useState<{ key: string; map: ReviewStatusMap } | null>(null)
  useEffect(() => {
    if (!key) return
    let active = true
    reviewApi.statuses(key.split(','))
      .then((items) => { if (active) setLoaded({ key, map: new Map(items.map((item) => [item.product_id, item])) }) })
      .catch(() => { /* Buttons stay disabled; the order list itself still works. */ })
    return () => { active = false }
  }, [key])
  if (!key) return new Map()
  return loaded?.key === key ? loaded.map : null
}

const baseClass = 'inline-block px-5 py-2 border'

/** One button per product of a delivered and paid order; rendered inline
 * so the caller's row (next to "Xem chi tiết đơn") controls the layout. */
export default function OrderReviewButtons({ order, statuses }: { order: Order; statuses: ReviewStatusMap | null }) {
  if (!orderReviewable(order)) return null
  const seen = new Set<string>()
  const lines = order.items.filter((item) => !seen.has(item.product_id) && seen.add(item.product_id))
  return (
    <>
      {lines.map((item) => {
        const status = statuses?.get(item.product_id)
        if (status?.slug && !status.reviewed)
          return (
            <Link key={item.product_id} to={`/san-pham/${encodeURIComponent(status.slug)}#danh-gia-khach-hang`} className={`${baseClass} bg-primary text-on-primary border-primary`}>
              Đánh giá {item.product_name}
            </Link>
          )
        const label = !status ? `Đánh giá ${item.product_name}` : status.reviewed ? `Đã đánh giá ${item.product_name}` : `${item.product_name} không còn hiển thị`
        return (
          <button key={item.product_id} type="button" disabled className={`${baseClass} border-outline-variant text-on-surface-variant opacity-60 cursor-not-allowed`}>
            {label}
          </button>
        )
      })}
    </>
  )
}
