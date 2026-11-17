import { Link } from 'react-router-dom'
import { orderLoginPath } from './presentation'

export default function OrderReadNotice({
  kind,
  next,
  retry,
  backTo = '/thanh-toan',
  backLabel = 'Trở lại checkout',
}: {
  kind: 'loading' | 'unauthenticated' | 'not-found' | 'error'
  next: string
  retry: () => void
  backTo?: string
  backLabel?: string
}) {
  if (kind === 'loading')
    return (
      <main className="max-w-3xl mx-auto px-5 py-12" role="status">
        Đang tải và kiểm tra quyền xem đơn...
      </main>
    )
  return (
    <main className="max-w-3xl mx-auto px-5 py-12 space-y-5">
      <h1 className="font-headline-lg text-primary text-3xl">
        {kind === 'unauthenticated'
          ? 'Vui lòng đăng nhập để xem đơn'
          : kind === 'not-found'
            ? 'Không tìm thấy đơn hoặc bạn không có quyền xem'
            : 'Chưa tải được đơn hàng'}
      </h1>
      {kind === 'unauthenticated' ? (
        <Link className="text-primary underline" to={orderLoginPath(next)}>
          Đăng nhập
        </Link>
      ) : kind === 'error' ? (
        <>
          <p role="alert">Không thể kết nối hoặc dịch vụ tạm không khả dụng. Vui lòng thử lại.</p>
          <button type="button" className="bg-primary text-on-primary px-6 py-3" onClick={retry}>
            Thử lại
          </button>
        </>
      ) : (
        <p>Vui lòng sử dụng đúng tài khoản hoặc quyền khách đã lưu trên trình duyệt này.</p>
      )}
      <Link className="text-primary underline block" to={backTo}>
        {backLabel}
      </Link>
    </main>
  )
}
