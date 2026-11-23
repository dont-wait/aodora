import { Link, useParams } from 'react-router-dom'
import Breadcrumb from '@/components/ui/Breadcrumb'
import OrderReadNotice from '@/features/orders/OrderReadNotice'
import { useOrderDetail } from '@/features/orders/useOrderRead'
import OrderSummary from '@/pages/checkout/OrderSummary'

export default function OrderDetailPage() {
  const { id = '' } = useParams()
  const { result, retry, isAccount } = useOrderDetail(id)
  const backTo = isAccount ? '/tai-khoan/don-hang' : '/bo-suu-tap'
  const backLabel = isAccount ? 'Đơn hàng của tôi' : 'Về bộ sưu tập'
  if (!result || result.kind !== 'ready')
    return <OrderReadNotice kind={result?.kind ?? 'loading'} next={`/don-hang/${encodeURIComponent(id)}`} retry={retry} backTo={backTo} backLabel={backLabel} />
  const { order } = result
  return (
    <main className="max-w-3xl mx-auto px-5 py-8 lg:py-14 space-y-7">
      <Breadcrumb
        items={[
          { label: 'Trang chủ', to: '/' },
          ...(isAccount ? [{ label: 'Đơn hàng của tôi', to: '/tai-khoan/don-hang' }] : []),
          { label: order.order_number },
        ]}
      />
      <header className="space-y-3">
        <p className="text-secondary text-sm uppercase tracking-widest">Thông tin đơn hàng</p>
        <h1 className="font-headline-lg text-4xl text-primary">Đơn {order.order_number}</h1>
        <p className="text-on-surface-variant">Thông tin được tải từ cửa hàng. Trạng thái nhận áo và thanh toán được cập nhật độc lập.</p>
      </header>
      <OrderSummary order={order} />
      {result.guest && (
        <p className="border border-outline-variant p-4 text-sm">
          Quyền xem đơn nằm trên trình duyệt này. Xóa dữ liệu trình duyệt hoặc đổi thiết bị có thể khiến bạn không xem lại được đơn.
        </p>
      )}
      <Link to={backTo} className="inline-block border border-primary text-primary px-6 py-3">
        {backLabel}
      </Link>
    </main>
  )
}
