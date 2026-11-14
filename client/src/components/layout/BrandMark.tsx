import { SITE } from '@/data/site'

// A typographic brand mark avoids depending on a broken third-party image URL.
// Brand assets can be provided later without changing product catalog data.
export default function BrandMark({ compact = false }: { compact?: boolean }) {
  const initials = SITE.name.split(/\s+/).map((word) => word.charAt(0)).join('')
  return (
    <span aria-label={`Biểu trưng ${SITE.name}`} role="img"
      className={`inline-flex shrink-0 items-center justify-center border border-primary/60 bg-surface-container-low text-primary font-title-editorial font-semibold tracking-tight ${compact ? 'w-8 h-8 text-base' : 'w-10 h-10 text-lg'}`}>
      {initials}
    </span>
  )
}
