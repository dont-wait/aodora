import type { Order } from './types.ts'

const fulfillmentLabels: Record<string, string> = {
  pending: 'Chờ xác nhận',
  confirmed: 'Đã xác nhận',
  shipping: 'Đang giao áo',
  delivered: 'Đã nhận áo',
  cancelled: 'Đã hủy',
}
/** Fulfillment statuses in lifecycle order, for filters. */
export const orderFulfillmentStatuses = Object.keys(fulfillmentLabels)
export const orderFulfillmentLabel = (status: string) => fulfillmentLabels[status] ?? 'Chưa xác định'
/** Unknown or repeated `status` query values fall back to "all". */
export function readOrderStatusFilter(params: URLSearchParams): string {
  const values = params.getAll('status')
  return values.length === 1 && orderFulfillmentStatuses.includes(values[0]) ? values[0] : ''
}
/** Delivered and fully paid: the customer is invited to review each product. */
export const orderReviewable = (order: Pick<Order, 'status' | 'payment_status'>) => order.status === 'delivered' && order.payment_status === 'paid'
const paymentLabels: Record<string, string> = { unpaid: 'Chưa thu tiền', paid: 'Đã thu đủ tiền', refunded: 'Đã hoàn tiền' }

export function orderStatusLabels(order: Pick<Order, 'status' | 'payment_status'>) {
  return { fulfillment: fulfillmentLabels[order.status] ?? 'Chưa xác định', payment: paymentLabels[order.payment_status] ?? 'Chưa xác định' }
}
export function orderPaymentLabel(method: string): string {
  if (method === 'cod') return 'Giao tận nơi · Thanh toán khi nhận áo (COD)'
  if (method === 'cash_at_counter') return 'Nhận và thanh toán tại quầy'
  return 'Phương thức thanh toán đã lưu'
}
export function orderDeliveryAddress(order: Pick<Order, 'payment_method' | 'shipping_address'>): string | null {
  if (order.payment_method !== 'cod') return null
  return [order.shipping_address.line1, order.shipping_address.ward, order.shipping_address.district, order.shipping_address.city].filter(Boolean).join(', ')
}
export function formatOrderDate(value: string): string {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return 'Chưa có ngày tạo'
  return new Intl.DateTimeFormat('vi-VN', { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}
export function orderLoginPath(next: string): string {
  return `/dang-nhap?next=${encodeURIComponent(next)}`
}
