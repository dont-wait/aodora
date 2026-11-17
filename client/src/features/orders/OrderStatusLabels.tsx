import Badge from '@/components/ui/Badge'
import { orderStatusLabels } from './presentation'
import type { Order } from './types'

export default function OrderStatusLabels({ order }: { order: Pick<Order, 'status' | 'payment_status'> }) {
  const labels = orderStatusLabels(order)
  return (
    <dl className="grid sm:grid-cols-2 gap-4">
      <div>
        <dt className="text-sm text-on-surface-variant mb-2">Nhận áo</dt>
        <dd>
          <Badge variant="surface" className="inline-block px-3 py-1">
            {labels.fulfillment}
          </Badge>
        </dd>
      </div>
      <div>
        <dt className="text-sm text-on-surface-variant mb-2">Thanh toán</dt>
        <dd>
          <Badge variant="secondaryFixed" className="inline-block px-3 py-1">
            {labels.payment}
          </Badge>
        </dd>
      </div>
    </dl>
  )
}
