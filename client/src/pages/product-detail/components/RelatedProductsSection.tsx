import { Link } from 'react-router-dom'
import { ROUTES } from '@/data/site'
import { getCategories, listProducts } from '@/features/catalog/api'
import { useCatalogRequest } from '@/features/catalog/useCatalogRequest'
import type { CatalogProduct } from '@/features/catalog/types'
import ProductCard from '@/pages/collection/components/ProductCard'

const SHOWN = 4

/** Other public products from the same category, newest first. */
export default function RelatedProductsSection({ product }: { product: CatalogProduct }) {
  // Ask for one extra so the current product can be dropped and still fill the row.
  const related = useCatalogRequest(
    (signal) => listProducts({ category: product.category_id, sort: 'newest', page: 1, page_size: SHOWN + 1 }, signal),
    `related:${product.category_id}:${product.id}`,
  )
  const categories = useCatalogRequest((signal) => getCategories(signal), 'categories')
  const category = categories.data?.find((item) => item.id === product.category_id)
  const items = (related.data?.data || []).filter((item) => item.id !== product.id).slice(0, SHOWN)
  // A failed or empty suggestion list never blocks the product page.
  if (related.loading || related.error || !items.length) return null
  return (
    <section className="w-full bg-surface py-16" aria-labelledby="goi-y-cung-loai">
      <div className="max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop">
        <div className="flex flex-wrap items-end justify-between gap-4 mb-8">
          <div>
            <p className="uppercase tracking-widest text-secondary text-sm">Có thể bạn cũng thích</p>
            <h2 id="goi-y-cung-loai" className="font-headline-lg text-headline-lg text-primary mt-2">Mẫu Áo Cùng Loại</h2>
          </div>
          <Link to={`${ROUTES.collection}?category=${encodeURIComponent(category?.slug || product.category_id)}`} className="text-primary underline">
            Xem thêm {category?.name ? `“${category.name}”` : 'cùng loại'}
          </Link>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {items.map((item) => <ProductCard key={item.id} product={item} />)}
        </div>
      </div>
    </section>
  )
}
