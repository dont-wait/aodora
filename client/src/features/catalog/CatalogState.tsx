import type { ReactNode } from 'react'
export default function CatalogState({ loading, error, onRetry, children }: { loading: boolean; error: string | null; onRetry: () => void; children: ReactNode }) {
  if (loading) return <div role="status" className="py-16 text-center text-on-surface-variant">Đang tải dữ liệu sản phẩm...</div>
  if (error) return <div role="alert" className="py-14 text-center space-y-4"><p className="text-on-surface-variant">{error}</p><button type="button" onClick={onRetry} className="bg-primary text-on-primary px-5 py-2">Thử lại</button></div>
  return <>{children}</>
}
