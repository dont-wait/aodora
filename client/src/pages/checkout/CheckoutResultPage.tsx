import { Link, useParams } from 'react-router-dom'
import { useOrderDetail } from '@/features/orders/useOrderRead'
import OrderReadNotice from '@/features/orders/OrderReadNotice'
import OrderSummary from './OrderSummary'

export default function CheckoutResultPage() {
  const { id = '' } = useParams()
  const { result, retry } = useOrderDetail(id)
  if (!result || result.kind !== 'ready')
    return <OrderReadNotice kind={result?.kind ?? 'loading'} next={`/dat-hang-thanh-cong/${encodeURIComponent(id)}`} retry={retry} />
  const { order, receipt, guest } = result
  return (
    <main className="max-w-3xl mx-auto px-5 py-10 lg:py-16 space-y-7">
      <header className="space-y-3">
        <p className="text-secondary tracking-widest uppercase text-sm">Cảm ơn bạn đã chọn Aodora</p>
        <h1 className="font-headline-lg text-4xl text-primary">Đơn hàng đã được ghi nhận</h1>
        <p>
          Mã đơn: <strong>{order.order_number}</strong>
        </p>
        <p className="text-on-surface-variant">Thông tin đơn được tải từ cửa hàng. Việc đặt đơn không đồng nghĩa với đã thanh toán.</p>
      </header>
      {receipt && receipt.cleanup !== 'done' && (
        <p role="alert" className="border border-secondary bg-secondary-fixed/20 p-4">
          Đơn đã được lưu, nhưng giỏ có thể chưa được dọn hết. Vui lòng kiểm tra giỏ trước lần mua tiếp theo; không cần đặt lại đơn này.
        </p>
      )}
      {guest && (
        <p className="border border-outline-variant p-4 text-sm">
          Quyền xem đơn được lưu tại trình duyệt này. Nếu xóa dữ liệu trình duyệt hoặc đổi thiết bị, bạn có thể mất quyền xem lại đơn.
        </p>
      )}
      <OrderSummary order={order} />
      <div className="flex flex-wrap gap-4">
        <Link to={`/don-hang/${encodeURIComponent(order.id)}`} className="bg-primary text-on-primary px-6 py-3">
          Xem chi tiết đơn
        </Link>
        <Link to="/bo-suu-tap" className="border border-primary text-primary px-6 py-3">
          Tiếp tục xem bộ sưu tập
        </Link>
        <Link to="/gio-hang" className="border border-primary text-primary px-6 py-3">
          Kiểm tra giỏ hàng
        </Link>
      </div>
    </main>
  )
}
