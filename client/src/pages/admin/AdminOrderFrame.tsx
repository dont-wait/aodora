import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

export default function AdminOrderFrame({ title, children }: { title: string; children: ReactNode }) {
  return (
    <main className="min-h-screen bg-surface-container-low p-4 sm:p-8">
      <div className="max-w-[1440px] mx-auto space-y-7">
        <header className="flex flex-wrap justify-between items-start gap-5">
          <div>
            <p className="text-secondary uppercase tracking-wider text-sm">Aodora · Chủ cửa hàng</p>
            <h1 className="text-3xl font-title-editorial text-primary mt-2">{title}</h1>
          </div>
          <nav aria-label="Điều hướng quản trị" className="flex flex-wrap gap-5 text-primary underline">
            <Link to="/admin/catalog">Danh mục & sản phẩm</Link>
            <Link to="/admin/orders">Quản trị đơn hàng</Link>
            <Link to="/admin/tailoring">Yêu cầu may đo</Link>
            <Link to="/admin/reviews">Quản lý đánh giá</Link>
            <Link to="/">Trang khách</Link>
          </nav>
        </header>
        {children}
      </div>
    </main>
  )
}
