import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useAuth } from '@/features/auth/useAuth'
import { loadCheckoutResult, type CheckoutResult } from '@/features/orders/result'
import OrderSummary from './OrderSummary'

export default function CheckoutResultPage() {
  const { id = '' } = useParams()
  const { user, loading } = useAuth()
  const [retry, setRetry] = useState(0)
  const [loaded, setLoaded] = useState<{ key: string; result: CheckoutResult } | null>(null)
  const customerId = user?.id ?? null
  const key = JSON.stringify([id, customerId, retry])
  useEffect(() => {
    if (loading) return
    let active = true
    loadCheckoutResult(id, customerId).then((result) => {
      if (active) setLoaded({ key, result })
    })
    return () => {
      active = false
    }
  }, [id, customerId, key, loading])

  // Hide the previous response immediately on id/identity changes or retry.
  // A late response from the prior request must never reveal the old order.
  if (loading || loaded?.key !== key)
    return (
      <main className="max-w-3xl mx-auto px-5 py-12" role="status">
        Đang tải và kiểm tra quyền xem đơn...
      </main>
    )
  const result = loaded.result
  if (result.kind !== 'ready')
    return (
      <main className="max-w-3xl mx-auto px-5 py-12 space-y-5">
        <h1 className="font-headline-lg text-primary text-3xl">
          {result.kind === 'unauthenticated'
            ? 'Vui lòng đăng nhập để xem đơn'
            : result.kind === 'not-found'
              ? 'Không tìm thấy đơn hoặc bạn không có quyền xem'
              : 'Chưa tải được đơn hàng'}
        </h1>
        {result.kind === 'unauthenticated' ? (
          <Link className="text-primary underline" to={`/dang-nhap?next=${encodeURIComponent(`/dat-hang-thanh-cong/${id}`)}`}>
            Đăng nhập
          </Link>
        ) : result.kind === 'error' ? (
          <>
            <p role="alert">Không thể kết nối hoặc dịch vụ tạm không khả dụng. Vui lòng thử lại.</p>
            <button type="button" className="bg-primary text-on-primary px-6 py-3" onClick={() => setRetry((value) => value + 1)}>
              Thử lại
            </button>
          </>
        ) : (
          <p>Vui lòng sử dụng đúng tài khoản hoặc quyền khách đã lưu trên trình duyệt này.</p>
        )}
        <Link className="text-primary underline block" to="/thanh-toan">
          Trở lại checkout
        </Link>
      </main>
    )
  const { order, receipt, guest } = result
  return (
    <main className="max-w-3xl mx-auto px-5 py-10 lg:py-16 space-y-7">
      <header className="space-y-3">
        <p className="text-secondary tracking-widest uppercase text-sm">Cảm ơn bạn đã chọn An Soie</p>
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
        <Link to="/bo-suu-tap" className="bg-primary text-on-primary px-6 py-3">
          Tiếp tục xem bộ sưu tập
        </Link>
        <Link to="/gio-hang" className="border border-primary text-primary px-6 py-3">
          Kiểm tra giỏ hàng
        </Link>
      </div>
    </main>
  )
}
