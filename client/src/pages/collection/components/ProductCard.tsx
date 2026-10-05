import { Link } from 'react-router-dom'
import Badge from '@/components/ui/Badge'
import Icon from '@/components/ui/Icon'
import { ROUTES } from '@/data/site'
import { useToast } from '@/hooks/useToast'
import type { Product } from '@/types'
import { formatVND } from '@/utils/format'

export default function ProductCard({ product }: { product: Product }) {
  const { showToast } = useToast()
  const to = ROUTES.product(product.slug)

  return (
    <article className="group flex flex-col bg-surface-container-low transition-all duration-300 hover:shadow-xl hover:-translate-y-1">
      <div className="relative aspect-[3/4] overflow-hidden bg-surface-container">
        <Link to={to}>
          <img
            src={product.image.src}
            alt={product.image.alt}
            loading="lazy"
            className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-105"
          />
        </Link>
        <div className="absolute top-3 left-3 flex flex-col gap-1.5">
          <Badge variant={product.badge.variant} className="px-2.5 py-1 text-[10px] tracking-widest shadow-sm">
            {product.badge.label}
          </Badge>
          {product.tag && (
            <span
              className={`px-2 py-0.5 font-label-uppercase text-[9px] tracking-wider ${
                product.tag.variant === 'secondaryContainer'
                  ? 'bg-secondary-container text-on-secondary-container'
                  : 'bg-surface/90 backdrop-blur text-secondary'
              }`}
            >
              {product.tag.label}
            </span>
          )}
        </div>
        <button
          type="button"
          aria-label="Yêu thích"
          onClick={() => showToast(`Đã lưu "${product.name}" vào danh sách yêu thích.`)}
          className="absolute top-3 right-3 w-8 h-8 rounded-full bg-surface/80 backdrop-blur text-on-surface hover:text-primary flex items-center justify-center transition-colors"
        >
          <Icon name="favorite" size={18} />
        </button>
        <div className="absolute bottom-3 inset-x-3 flex items-center justify-between p-2 bg-surface/90 backdrop-blur opacity-0 group-hover:opacity-100 transition-opacity duration-200">
          <div className="flex items-center gap-1.5">
            {product.colors.map((hex) => (
              <span key={hex} className="w-3.5 h-3.5 rounded-full shadow-inner" style={{ backgroundColor: hex }} />
            ))}
          </div>
          <span className="font-label-uppercase text-[10px] text-secondary">{product.colorNote}</span>
        </div>
      </div>

      <div className="p-5 flex flex-col flex-1">
        <span className="font-label-uppercase text-[10px] text-on-surface-variant tracking-wider uppercase mb-1">{product.material}</span>
        <h3 className="font-headline-sm text-[20px] leading-tight text-on-surface group-hover:text-primary transition-colors mb-2">
          <Link to={to}>{product.name}</Link>
        </h3>
        <p className="font-body-sm text-[13px] text-on-surface-variant line-clamp-1 mb-3">{product.description}</p>

        <div className="flex items-center gap-1.5 mb-4 min-h-[22px]">
          {product.sizes.map((s) => (
            <span key={s} className="px-1.5 py-0.5 bg-surface text-[11px] font-label-regular text-on-surface-variant">
              {s}
            </span>
          ))}
          {product.bespoke && (
            <span className="px-1.5 py-0.5 bg-secondary-fixed text-[11px] font-label-regular text-on-secondary-fixed font-semibold">
              {product.sizes.length ? 'May Đo' : 'May Đo Riêng 1:1'}
            </span>
          )}
        </div>

        <div className="mt-auto pt-3 flex items-center justify-between">
          <span className="font-title-editorial text-[20px] text-primary font-bold">{formatVND(product.price)}</span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              title="Thêm vào giỏ"
              onClick={() => showToast(`Đã thêm "${product.name}" vào giỏ hàng.`)}
              className="w-9 h-9 bg-surface text-on-surface hover:bg-surface-container flex items-center justify-center transition-colors"
            >
              <Icon name="shopping_bag" size={18} />
            </button>
            <Link
              to={to}
              className="px-3 py-2 bg-primary text-on-primary font-label-uppercase text-[10px] tracking-wider hover:bg-primary-container transition-colors"
            >
              {product.cta}
            </Link>
          </div>
        </div>
      </div>
    </article>
  )
}
