import { useState } from 'react'

interface Props {
  src?: string
  alt: string
  className?: string
  imageClassName?: string
  loading?: 'eager' | 'lazy'
}

export default function CatalogImage({ src, alt, className = '', imageClassName = '', loading = 'lazy' }: Props) {
  // The failed URL is keyed by its value so switching between different
  // product images automatically gives the new source a chance to load.
  const [failedSrc, setFailedSrc] = useState<string | null>(null)
  const showImage = !!src?.trim() && failedSrc !== src
  return (
    <div className={`relative overflow-hidden ${className}`}>
      {showImage ? (
        <img src={src} alt={alt} loading={loading} onError={() => setFailedSrc(src!)}
          className={`h-full w-full object-cover ${imageClassName}`} />
      ) : (
        <div className="flex h-full w-full flex-col items-center justify-center gap-3 bg-surface-container text-on-surface-variant px-4 text-center"
          role="img" aria-label={`${alt} — chưa có ảnh`}>
          <span className="text-3xl text-primary/40" aria-hidden="true">◇</span>
          <span className="text-xs sm:text-sm">Hình ảnh đang được cập nhật</span>
        </div>
      )}
    </div>
  )
}
