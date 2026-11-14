import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import Breadcrumb from '@/components/ui/Breadcrumb'
import { ROUTES } from '@/data/site'
import { getProduct } from '@/features/catalog/api'
import { useCatalogRequest } from '@/features/catalog/useCatalogRequest'
import CatalogState from '@/features/catalog/CatalogState'
import ProductGallery from './components/ProductGallery'
import PurchasePanel from './components/PurchasePanel'
import InfoTabs from './components/InfoTabs'
import ReviewsSection from './components/ReviewsSection'
import AccessoriesSection from './components/AccessoriesSection'
import SizeGuideModal from './components/SizeGuideModal'

export default function ProductDetailPage() {
  const { slug = '' } = useParams<{ slug: string }>()
  const [sizeGuideOpen, setSizeGuideOpen] = useState(false)
  const product = useCatalogRequest((signal) => getProduct(slug, signal), `product:${slug}`)
  return <div className="flex flex-col w-full">
    <div className="w-full bg-surface-container-low px-4 sm:px-8 lg:px-margin-desktop py-4">
      <Breadcrumb className="max-w-[1440px] mx-auto font-label-regular text-label-regular text-on-surface-variant"
        items={[{ label: 'Trang chủ', to: ROUTES.home }, { label: 'Bộ sưu tập', to: ROUTES.collection }, { label: product.data?.name || 'Chi tiết sản phẩm' }]} />
    </div>
    <div className="max-w-[1440px] mx-auto w-full px-4 sm:px-8 lg:px-margin-desktop">
      {product.error ? <div role="alert" className="py-16 text-center space-y-4">
        <p>{product.error}</p>
        <Link to={ROUTES.collection} className="underline text-primary block">Quay lại bộ sưu tập</Link>
        <button type="button" onClick={product.retry} className="px-4 py-2 bg-primary text-on-primary">Thử tải lại</button>
      </div> : <CatalogState loading={product.loading} error={null} onRetry={product.retry}>
        {product.data && <>
          <section className="py-8 lg:py-12 grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-gutter-desktop items-start">
            <ProductGallery key={product.data.id} product={product.data} />
            <PurchasePanel key={product.data.id} product={product.data} onOpenSizeGuide={() => setSizeGuideOpen(true)} />
          </section>
        </>}
        {!product.data && !product.error && <div className="py-16 text-center">Không tìm thấy mẫu áo. <Link className="underline text-primary" to={ROUTES.collection}>Quay lại bộ sưu tập</Link></div>}
      </CatalogState>}
    </div>
    {product.data && <>
      <InfoTabs key={product.data.id} product={product.data} />
      <ReviewsSection summary={product.data.reviews} />
      <AccessoriesSection accessories={product.data.accessories || []} />
      <SizeGuideModal open={sizeGuideOpen} sizes={product.data.sizes || []} onClose={() => setSizeGuideOpen(false)} />
    </>}
  </div>
}
