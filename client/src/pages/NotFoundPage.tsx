import { Link } from 'react-router-dom'
import { ROUTES } from '@/data/site'

export default function NotFoundPage() {
  return (
    <main className="max-w-[720px] mx-auto px-margin py-24 text-center space-y-6">
      <p className="uppercase tracking-widest text-secondary text-sm">Lỗi 404</p>
      <h1 className="font-headline-lg text-headline-lg text-primary">Không tìm thấy trang</h1>
      <p className="text-on-surface-variant">Đường dẫn bạn mở không tồn tại hoặc đã được di chuyển.</p>
      <div className="flex flex-wrap justify-center gap-4">
        <Link to={ROUTES.home} className="bg-primary text-on-primary px-6 py-3">Về trang chủ</Link>
        <Link to={ROUTES.collection} className="bg-surface-container-high text-primary px-6 py-3">Xem bộ sưu tập</Link>
      </div>
    </main>
  )
}
