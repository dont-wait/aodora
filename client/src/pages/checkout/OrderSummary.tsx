import { formatOrderMoney } from '@/features/orders/money'
import type { Order } from '@/features/orders/types'

const statuses: Record<string, string> = {
  pending: 'Chờ xác nhận',
  confirmed: 'Đã xác nhận',
  shipping: 'Đang giao áo',
  delivered: 'Đã nhận áo',
  cancelled: 'Đã hủy',
}
const payments: Record<string, string> = { unpaid: 'Chưa thu tiền', paid: 'Đã thu đủ tiền', refunded: 'Đã hoàn tiền' }

export default function OrderSummary({ order }: { order: Order }) {
  return (
    <div className="space-y-6">
      <dl className="grid sm:grid-cols-2 gap-4 border border-outline-variant p-5">
        <div>
          <dt className="text-sm text-on-surface-variant">Giao nhận</dt>
          <dd className="text-primary font-semibold">{statuses[order.status] || order.status}</dd>
        </div>
        <div>
          <dt className="text-sm text-on-surface-variant">Thanh toán</dt>
          <dd className="text-primary font-semibold">{payments[order.payment_status] || order.payment_status}</dd>
        </div>
      </dl>
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
        <p>{order.payment_method === 'cash_at_counter' ? 'Nhận và thanh toán tại quầy' : 'Giao tận nơi · Thanh toán khi nhận áo'}</p>
        {order.payment_method !== 'cash_at_counter' && (
          <p>
            {[order.shipping_address.line1, order.shipping_address.ward, order.shipping_address.district, order.shipping_address.city]
              .filter(Boolean)
              .join(', ')}
          </p>
        )}
        {order.note && <p>Ghi chú: {order.note}</p>}
      </section>
    </div>
  )
}
