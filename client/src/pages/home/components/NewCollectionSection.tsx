import { Link } from 'react-router-dom'
import { ROUTES } from '@/data/site'
import { listProducts } from '@/features/catalog/api'
import { useCatalogRequest } from '@/features/catalog/useCatalogRequest'
import CatalogState from '@/features/catalog/CatalogState'
import ProductCard from '@/pages/collection/components/ProductCard'

export default function NewCollectionSection() {
  const products = useCatalogRequest((signal) => listProducts({ sort: 'newest', page: 1, page_size: 4 }, signal), 'home:featured')
  return <section className="w-full py-16 lg:py-24 bg-surface" id="bo-suu-tap-moi">
    <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
      <div className="mb-10"><p className="uppercase tracking-widest text-secondary text-sm">Sản phẩm cập nhật từ cửa hàng</p>
        <h2 className="font-headline-lg text-headline-lg text-primary mt-2">Những Mẫu Áo Mới Nhất</h2></div>
      <CatalogState loading={products.loading} error={products.error} onRetry={products.retry}>
        {products.data?.data.length ? <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {products.data.data.map((p) => <ProductCard key={p.id} product={p} />)}
        </div> : <p className="py-10 text-on-surface-variant">Cửa hàng chưa có sản phẩm đang bán.</p>}
      </CatalogState>
      <div className="mt-12 text-center"><Link to={ROUTES.collection} className="inline-flex bg-primary text-on-primary px-8 py-3">Xem toàn bộ sản phẩm</Link></div>
    </div>
  </section>
}
