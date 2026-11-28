import type { ReactNode } from 'react'

/**
 * Customer-side loading placeholders. Each block is decorative
 * (aria-hidden); the wrapper announces the loading state once via
 * role="status" + aria-label so screen readers hear one message.
 */
export function SkeletonBlock({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`animate-pulse bg-surface-container ${className}`} />
}

export function SkeletonStatus({ label, className = '', children }: { label: string; className?: string; children: ReactNode }) {
  return (
    <div role="status" aria-label={label} aria-busy="true" className={className}>
      {children}
    </div>
  )
}

/** Card grid used by home categories, collection, related products and wishlist. */
export function CardGridSkeleton({ count = 4, className = 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6', label = 'Đang tải danh sách' }: { count?: number; className?: string; label?: string }) {
  return (
    <SkeletonStatus label={label} className={className}>
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="space-y-3">
          <SkeletonBlock className="aspect-[3/4]" />
          <SkeletonBlock className="h-4 w-3/4" />
          <SkeletonBlock className="h-4 w-1/3" />
        </div>
      ))}
    </SkeletonStatus>
  )
}

/** Horizontal line items: cart lines, order rows. */
export function ListSkeleton({ rows = 3, label, media = true }: { rows?: number; label: string; media?: boolean }) {
  return (
    <SkeletonStatus label={label} className="space-y-4">
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className={`bg-surface-container-low p-4 sm:p-5 grid gap-4 ${media ? 'grid-cols-[5rem_1fr] sm:grid-cols-[7rem_1fr]' : ''}`}>
          {media && <SkeletonBlock className="aspect-[3/4]" />}
          <div className="space-y-3">
            <SkeletonBlock className="h-3 w-1/4" />
            <SkeletonBlock className="h-6 w-2/3" />
            <SkeletonBlock className="h-4 w-1/3" />
            <SkeletonBlock className="h-10 w-40" />
          </div>
        </div>
      ))}
    </SkeletonStatus>
  )
}

/** Generic page: title area + stacked content blocks. */
export function PageSkeleton({ label, blocks = 3, narrow = false }: { label: string; blocks?: number; narrow?: boolean }) {
  return (
    <main className={`${narrow ? 'max-w-3xl' : 'max-w-[1180px]'} mx-auto px-5 py-8 lg:py-14`}>
      <SkeletonStatus label={label} className="space-y-7">
        <SkeletonBlock className="h-3 w-40" />
        <div className="space-y-3">
          <SkeletonBlock className="h-3 w-28" />
          <SkeletonBlock className="h-10 w-2/3 max-w-md" />
          <SkeletonBlock className="h-4 w-1/2 max-w-sm" />
        </div>
        {Array.from({ length: blocks }, (_, i) => (
          <div key={i} className="border border-outline-variant/50 p-5 sm:p-7 space-y-4">
            <SkeletonBlock className="h-6 w-1/3" />
            <SkeletonBlock className="h-4 w-full" />
            <SkeletonBlock className="h-4 w-5/6" />
            <SkeletonBlock className="h-10 w-36" />
          </div>
        ))}
      </SkeletonStatus>
    </main>
  )
}

/** Review cards on the product page. */
export function ReviewCardsSkeleton({ count = 3 }: { count?: number }) {
  return (
    <SkeletonStatus label="Đang tải đánh giá" className="grid grid-cols-1 md:grid-cols-3 gap-5">
      {Array.from({ length: count }, (_, i) => (
        <div key={i} className="bg-surface p-5 border border-outline-variant/50 space-y-3">
          <div className="flex justify-between"><SkeletonBlock className="h-4 w-24" /><SkeletonBlock className="h-3 w-16" /></div>
          <SkeletonBlock className="h-4 w-1/2" />
          <SkeletonBlock className="h-3 w-full" />
          <SkeletonBlock className="h-3 w-4/5" />
        </div>
      ))}
    </SkeletonStatus>
  )
}
