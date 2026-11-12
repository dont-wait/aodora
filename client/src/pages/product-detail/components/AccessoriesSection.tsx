import { formatVND } from '@/utils/format'
import type { CatalogProduct } from '@/features/catalog/types'
export default function AccessoriesSection({ accessories }: { accessories: CatalogProduct['accessories'] }) {
  if (!accessories.length) return null
  return <section className="w-full bg-surface-container py-16">
    <div className="max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop">
      <h2 className="font-headline-lg text-headline-lg text-primary mb-8">Phụ Kiện Đi Kèm</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
        {accessories.map((item, i) => <article key={`${item.id}-${i}`} className="p-5 bg-surface shadow-sm">
          {item.image?.src && <img src={item.image.src} alt={item.image.alt || item.name} loading="lazy" className="w-full aspect-square object-cover mb-4" />}
          <h3 className="font-headline-sm text-headline-sm">{item.name}</h3>
          <p className="text-on-surface-variant my-2">{item.description}</p>
          <strong className="text-primary">{formatVND(item.price)}</strong>
        </article>)}
      </div>
    </div>
  </section>
}
