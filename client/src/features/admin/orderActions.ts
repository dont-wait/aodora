import { adminOrders, AdminOrderApiError, type AdminOrder, type AdminOrderFilter, type AdminOrderPage } from './orders.ts'
import { formatOrderMoney } from '../orders/money.ts'

export type AdminOrderAction = 'confirm' | 'ship' | 'cancel' | 'restock' | 'mark_paid' | 'mark_refunded'
const actions: { id: AdminOrderAction; group: 'delivery' | 'payment'; label: string }[] = [
  { id: 'confirm', group: 'delivery', label: 'Xác nhận đơn' },
  { id: 'ship', group: 'delivery', label: 'Bắt đầu giao áo' },
  { id: 'cancel', group: 'delivery', label: 'Hủy đơn' },
  { id: 'restock', group: 'delivery', label: 'Xác nhận hàng đã về kho' },
  { id: 'mark_paid', group: 'payment', label: 'Ghi nhận đã thu đủ tiền' },
  { id: 'mark_refunded', group: 'payment', label: 'Ghi nhận đã hoàn tiền thủ công' },
]
export function adminOrderActionGroups(order: Pick<AdminOrder, 'allowed_actions'>) {
  const allowed = new Set(order.allowed_actions)
  return {
    delivery: actions.filter((action) => action.group === 'delivery' && allowed.has(action.id)),
    payment: actions.filter((action) => action.group === 'payment' && allowed.has(action.id)),
  }
}
/** Delivery is closed by the buyer ("Đã nhận được hàng"), which also records payment. */
export function awaitingCustomerReceipt(order: Pick<AdminOrder, 'status' | 'payment_method'>): boolean {
  return order.payment_method === 'cash_at_counter' ? order.status === 'confirmed' : order.status === 'shipping'
}
export function adminOrderConfirmation(order: AdminOrder, action: AdminOrderAction): string {
  switch (action) {
    case 'confirm':
      return 'Xác nhận đơn để chuẩn bị giao hoặc nhận tại quầy. Thao tác không ghi nhận thu tiền.'
    case 'ship':
      return 'Xác nhận bắt đầu giao áo. Trạng thái thanh toán giữ nguyên.'
    case 'cancel':
      return `${order.status === 'shipping' ? 'Hủy đơn đang giao: chưa cộng tồn vì hàng chưa về kho. Chỉ nhập lại khi xác nhận hàng thực tế đã về.' : 'Hủy đơn: hàng đủ điều kiện sẽ được hoàn tồn; hàng may đo hoặc đơn cũ không áp dụng hoàn tồn sẽ không được cộng tồn.'} Hủy không đổi trạng thái thanh toán và không tự hoàn tiền.`
    case 'restock':
      return 'Xác nhận hàng đã thực tế về kho. Hệ thống nhập lại tồn đúng một lần, không đổi trạng thái thanh toán.'
    case 'mark_paid':
      return `Bạn xác nhận đã thực tế thu đủ ${formatOrderMoney(order.total)}. Chỉ ghi nhận đã thu tiền, không đổi trạng thái giao nhận.`
    case 'mark_refunded':
      return `Bạn xác nhận đã hoàn đủ ${formatOrderMoney(order.total)} cho khách bằng cách thủ công. Thao tác chỉ ghi nhận việc đã hoàn tiền, không tự chuyển tiền và không đổi trạng thái giao nhận.`
  }
}
export function inventoryRestockLabel(value?: string): string {
  return (
    ({ restocked: 'Đã nhập lại', awaiting_return: 'Chờ hàng về', not_applicable: 'Không áp dụng' } as Record<string, string>)[value || ''] ||
    'Chưa có dấu hoàn tồn'
  )
}
export type AdminOrderFailure = { kind: 'unauthenticated' | 'forbidden' | 'not-found' | 'error'; message: string }
export type AdminOrderRead = { kind: 'ready'; order: AdminOrder } | AdminOrderFailure
export type AdminOrderListRead = { kind: 'ready'; page: AdminOrderPage } | AdminOrderFailure
export type AdminOrderActionResult = { kind: 'saved'; order: AdminOrder } | { kind: 'conflict'; order: AdminOrder; message: string } | AdminOrderFailure

function failure(error: unknown, message = 'Không thể tải đơn. Vui lòng thử lại.'): AdminOrderFailure {
  if (error instanceof AdminOrderApiError) {
    if (error.status === 401) return { kind: 'unauthenticated', message: 'Vui lòng đăng nhập lại.' }
    if (error.status === 403) return { kind: 'forbidden', message: 'Bạn không có quyền quản trị đơn hàng.' }
    if (error.status === 404) return { kind: 'not-found', message: 'Không tìm thấy đơn hàng.' }
  }
  return { kind: 'error', message }
}
export async function loadAdminOrder(id: string): Promise<AdminOrderRead> {
  try {
    return { kind: 'ready', order: await adminOrders.get(id) }
  } catch (error) {
    return failure(error)
  }
}
export async function loadAdminOrders(filter: AdminOrderFilter): Promise<AdminOrderListRead> {
  try {
    return { kind: 'ready', page: await adminOrders.list(filter) }
  } catch (error) {
    return failure(error)
  }
}
export async function performAdminOrderAction(order: AdminOrder, action: AdminOrderAction): Promise<AdminOrderActionResult> {
  try {
    if (!order.allowed_actions.includes(action)) throw new AdminOrderApiError(409, 'ORDER_STATE_CONFLICT', '', 'Thao tác không còn được phép.')
    let updated: AdminOrder
    switch (action) {
      case 'confirm':
        updated = await adminOrders.setStatus(order.id, 'confirmed')
        break
      case 'ship':
        updated = await adminOrders.setStatus(order.id, 'shipping')
        break
      case 'cancel':
        updated = await adminOrders.cancel(order.id)
        break
      case 'restock':
        updated = await adminOrders.restock(order.id)
        break
      case 'mark_paid':
        updated = await adminOrders.setPayment(order.id, 'paid')
        break
      case 'mark_refunded':
        updated = await adminOrders.setPayment(order.id, 'refunded')
        break
    }
    return { kind: 'saved', order: updated }
  } catch (error) {
    if (error instanceof AdminOrderApiError && error.status === 409) {
      const message =
        error.code === 'RESTOCK_UNAVAILABLE'
          ? 'Không thể hoàn tồn vì sản phẩm hoặc SKU không còn. Cần khôi phục sản phẩm/SKU trước khi thao tác lại.'
          : error.code === 'PAYMENT_STATE_CONFLICT'
            ? 'Trạng thái thanh toán đã thay đổi hoặc thao tác không còn hợp lệ.'
            : 'Trạng thái giao nhận đã thay đổi hoặc thao tác không còn hợp lệ.'
      const latest = await loadAdminOrder(order.id)
      if (latest.kind === 'ready') return { kind: 'conflict', order: latest.order, message: `${message} Đã tải lại thông tin mới nhất.` }
      if (latest.kind === 'error') return { kind: 'error', message: `${message} Chưa tải lại được đơn; vui lòng thử tải lại trước khi thao tác tiếp.` }
      return latest
    }
    return failure(error, 'Chưa xác định được kết quả thao tác. Vui lòng tải lại đơn trước khi thao tác tiếp.')
  }
}
