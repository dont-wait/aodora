import { formatAddress } from '../address/divisions.ts'
import type { Order } from './types.ts'

const fulfillmentLabels: Record<string, string> = {
  pending: 'Chờ xác nhận',
  confirmed: 'Đã xác nhận',
  shipping: 'Đang giao áo',
  delivered: 'Đã nhận áo',
  cancelled: 'Đã hủy',
}
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
  return formatAddress(order.shipping_address)
}
export function formatOrderDate(value: string): string {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return 'Chưa có ngày tạo'
  return new Intl.DateTimeFormat('vi-VN', { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}
export function orderLoginPath(next: string): string {
  return `/dang-nhap?next=${encodeURIComponent(next)}`
}
