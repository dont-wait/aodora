import type { ReactNode } from 'react'

export function ProductGridSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6 md:gap-8" aria-hidden="true">
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="animate-pulse space-y-3">
          <div className="aspect-[3/4] bg-surface-container" />
          <div className="h-4 w-3/4 bg-surface-container" />
          <div className="h-4 w-1/3 bg-surface-container" />
        </div>
      ))}
    </div>
  )
}

export function ProductDetailSkeleton() {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 animate-pulse" aria-hidden="true">
      <div className="lg:col-span-7 aspect-[4/5] bg-surface-container" />
      <div className="lg:col-span-5 space-y-4">
        <div className="h-4 w-1/3 bg-surface-container" />
        <div className="h-9 w-4/5 bg-surface-container" />
        <div className="h-20 bg-surface-container" />
        <div className="h-12 bg-surface-container" />
        <div className="h-12 bg-surface-container" />
      </div>
    </div>
  )
}

interface Props {
  loading: boolean
  error: string | null
  onRetry: () => void
  children: ReactNode
  /** Shown instead of the plain loading text while data loads. */
  skeleton?: ReactNode
}

export default function CatalogState({ loading, error, onRetry, children, skeleton }: Props) {
  if (loading) {
    return (
      <div role="status" aria-label="Đang tải dữ liệu sản phẩm">
        {skeleton ?? <div className="py-16 text-center text-on-surface-variant">Đang tải dữ liệu sản phẩm...</div>}
      </div>
    )
  }
  if (error) return <div role="alert" className="py-14 text-center space-y-4"><p className="text-on-surface-variant">{error}</p><button type="button" onClick={onRetry} className="bg-primary text-on-primary px-5 py-2">Thử lại</button></div>
  return <>{children}</>
}
