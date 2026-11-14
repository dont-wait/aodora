import { Link } from 'react-router-dom'

export default function ForbiddenPage() {
  return <main className="min-h-screen bg-background text-on-surface flex items-center justify-center px-5 py-20">
    <div className="max-w-xl w-full text-center space-y-6 bg-surface p-8 sm:p-12 border border-outline/30">
      <p className="text-secondary uppercase tracking-widest text-sm">Aodora · Truy cập bị từ chối</p>
      <h1 className="text-7xl font-title-editorial text-primary">403</h1>
      <h2 className="text-2xl font-title-editorial text-primary">Bạn không có quyền ở đây</h2>
      <p className="text-on-surface-variant">Khu vực này chỉ dành cho chủ cửa hàng. Bạn có thể quay về trang khách để tiếp tục xem các mẫu áo.</p>
      <Link to="/" className="inline-flex bg-primary text-on-primary px-6 py-3 rounded-sm">Về trang chủ</Link>
    </div>
  </main>
}
