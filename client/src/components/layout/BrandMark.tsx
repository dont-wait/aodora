import { SITE } from '@/data/site'

// Emblem cropped from /public/logo/aodora.png (the full 2048px original is not loaded in the UI).
export default function BrandMark({ compact = false }: { compact?: boolean }) {
  return (
    <img src="/logo/aodora-mark.png" alt={`Biểu trưng ${SITE.name}`} width={256} height={256}
      className={`shrink-0 rounded-full object-cover ${compact ? 'w-10 h-10' : 'w-14 h-14'}`} />
  )
}
