import { useState } from 'react'
import type { CatalogProduct } from '@/features/catalog/types'
import { badgeVariant } from '@/features/catalog/types'
import Badge from '@/components/ui/Badge'
import CatalogImage from '@/features/catalog/CatalogImage'
import { productGallery } from '@/features/catalog/gallery'
export default function ProductGallery({ product }: { product: CatalogProduct }) {
  const [active, setActive] = useState(0)
  const images = productGallery(product)
  const current = images[active] || images[0]
  return <div className="lg:col-span-7 flex flex-col gap-5">
    <div className="relative aspect-[3/4] w-full overflow-hidden bg-surface-container shadow-sm">
      {product.badge?.label && <Badge variant={badgeVariant(product.badge.variant)} className="absolute z-10 top-4 left-4 px-3 py-1">{product.badge.label}</Badge>}
      <CatalogImage src={current?.src} alt={current?.alt || product.name} className="w-full h-full" imageClassName="object-top" loading="eager" />
    </div>
    {images.length > 1 && <div className="grid grid-cols-4 gap-3 sm:gap-4">
      {images.map((image, index) => <button key={`${image.src}-${index}`} type="button" aria-label={`Xem ảnh ${index + 1}`} aria-pressed={index === active}
        onClick={() => setActive(index)} className={`aspect-square overflow-hidden p-1 bg-surface-container ${active === index ? 'ring-2 ring-primary' : 'opacity-75'}`}>
        <CatalogImage src={image.src} alt={image.alt || product.name} className="w-full h-full" />
      </button>)}
    </div>}
    {!!product.trust?.length && <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 bg-surface-container-low p-5">
      {product.trust.map((item, i) => <div key={`${item.title}-${i}`}><strong className="text-primary">{item.title}</strong><p className="text-sm text-on-surface-variant mt-1">{item.description}</p></div>)}
    </div>}
  </div>
}
