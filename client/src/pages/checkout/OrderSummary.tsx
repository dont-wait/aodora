import { formatOrderMoney } from '@/features/orders/money'
import OrderStatusLabels from '@/features/orders/OrderStatusLabels'
import { formatOrderDate, orderDeliveryAddress, orderPaymentLabel } from '@/features/orders/presentation'
import type { Order } from '@/features/orders/types'

export default function OrderSummary({ order }: { order: Order }) {
  return (
    <div className="space-y-6">
      <p className="text-on-surface-variant">
        Ngày đặt: <time dateTime={order.created_at}>{formatOrderDate(order.created_at)}</time>
      </p>
      <div className="border border-outline-variant p-5">
        <OrderStatusLabels order={order} />
      </div>
      <div>
        <h2 className="font-headline-md text-2xl text-primary">Áo đã đặt</h2>
        <ul className="divide-y divide-outline-variant mt-3">
          {order.items.map((item) => (
            <li key={`${item.product_id}:${item.sku}`} className="py-4">
              <p className="font-semibold">{item.product_name}</p>
              <p className="text-sm text-on-surface-variant">{[item.color, item.size, item.sku].filter(Boolean).join(' · ')}</p>
              <p className="mt-1">
                {item.quantity} áo · Đơn giá {formatOrderMoney(item.unit_price)}
              </p>
            </li>
          ))}
        </ul>
      </div>
      <dl className="space-y-3 border-t border-outline-variant pt-5">
        <div className="flex justify-between gap-4">
          <dt>Tiền hàng</dt>
          <dd>{formatOrderMoney(order.subtotal)}</dd>
        </div>
        <div className="flex justify-between gap-4">
          <dt>Phí giao hàng</dt>
          <dd>{formatOrderMoney(order.shipping_fee)}</dd>
        </div>
        <div className="flex justify-between gap-4 font-semibold text-xl text-primary">
          <dt>Tổng tiền</dt>
          <dd>{formatOrderMoney(order.total)}</dd>
        </div>
      </dl>
      <section className="bg-surface-container-low p-5 space-y-2">
        <h2 className="font-headline-sm text-primary">Thông tin nhận áo</h2>
        <p>
          {order.shipping_address.recipient} · {order.shipping_address.phone}
        </p>
        <p>{orderPaymentLabel(order.payment_method)}</p>
        {orderDeliveryAddress(order) && <p>{orderDeliveryAddress(order)}</p>}
        {order.note && <p>Ghi chú: {order.note}</p>}
      </section>
    </div>
  )
}
