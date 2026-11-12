import { Link } from 'react-router-dom'
import Badge from '@/components/ui/Badge'
import { ROUTES } from '@/data/site'
import type { CatalogProductSummary } from '@/features/catalog/types'
import { badgeVariant } from '@/features/catalog/types'
import { formatVND } from '@/utils/format'

export default function ProductCard({ product }: { product: CatalogProductSummary }) {
  const link = ROUTES.product(product.slug)
  const colors = product.colors?.filter((c) => !!c.hex) || []
  return <article className="group flex flex-col bg-surface-container-low transition-all duration-300 hover:shadow-xl hover:-translate-y-1">
    <Link to={link} className="relative aspect-[3/4] bg-surface-container overflow-hidden block" aria-label={`Xem ${product.name}`}>
      {product.image?.src ? <img src={product.image.src} alt={product.image.alt || product.name} loading="lazy"
        className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-105" />
        : <div className="w-full h-full flex items-center justify-center text-on-surface-variant">Chưa có ảnh</div>}
      {product.badge?.label && <Badge variant={badgeVariant(product.badge.variant)} className="absolute top-3 left-3 px-2.5 py-1 shadow-sm">{product.badge.label}</Badge>}
      {!!colors.length && <div className="absolute bottom-3 left-3 p-2 bg-surface/90 flex gap-2">
        {colors.map((c, i) => <span key={`${c.name}-${i}`} title={c.name} className="w-4 h-4 rounded-full border border-outline-variant" style={{ backgroundColor: c.hex }} />)}
      </div>}
    </Link>
    <div className="p-5 flex flex-col flex-1 gap-2">
      {product.material && <span className="font-label-uppercase text-[10px] tracking-wider text-on-surface-variant">{product.material}</span>}
      <h3 className="font-headline-sm text-[20px] leading-tight group-hover:text-primary transition-colors"><Link to={link}>{product.name}</Link></h3>
      <p className="font-body-sm text-[13px] text-on-surface-variant line-clamp-2">{product.description}</p>
      <div className="flex gap-1.5 flex-wrap mt-2">{product.sizes?.map((size) => <span key={size} className="bg-surface px-2 py-1 text-xs">{size}</span>)}
        {product.bespoke && <span className="bg-secondary-fixed px-2 py-1 text-xs">May đo</span>}
      </div>
      <div className="mt-auto pt-3 flex items-center justify-between gap-3">
        <div><span className="font-title-editorial text-[20px] text-primary font-bold">{formatVND(product.price)}</span>
          {!product.ready_available && !product.bespoke && <span className="block text-sm text-on-surface-variant">Hết hàng</span>}</div>
        <Link to={link} className="px-3 py-2 bg-primary text-on-primary font-label-uppercase text-[10px] tracking-wider hover:bg-primary-container">Xem mẫu</Link>
      </div>
    </div>
  </article>
}
