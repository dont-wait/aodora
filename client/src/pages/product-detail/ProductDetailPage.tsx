import { useState } from 'react'
import Breadcrumb from '@/components/ui/Breadcrumb'
import { productDetail } from '@/data/productDetail'
import { ROUTES } from '@/data/site'
import AccessoriesSection from './components/AccessoriesSection'
import InfoTabs from './components/InfoTabs'
import ProductGallery from './components/ProductGallery'
import PurchasePanel from './components/PurchasePanel'
import ReviewsSection from './components/ReviewsSection'
import SizeGuideModal from './components/SizeGuideModal'

/**
 * Hiện mới có một sản phẩm mẫu; mọi slug đều hiển thị dữ liệu này.
 * TODO: lấy dữ liệu theo `useParams().slug` khi có API.
 */
export default function ProductDetailPage() {
  const [sizeGuideOpen, setSizeGuideOpen] = useState(false)

  return (
    <div className="flex flex-col w-full">
      <div className="w-full bg-surface-container-low px-4 sm:px-8 lg:px-margin-desktop py-4">
        <Breadcrumb
          className="max-w-[1440px] mx-auto font-label-regular text-label-regular text-on-surface-variant"
          items={[
            { label: 'Trang chủ', to: ROUTES.home },
            { label: 'Bộ sưu tập', to: ROUTES.collection },
            { label: 'Cổ phục & Dân gian', to: `${ROUTES.collection}?loai=co-phuc` },
            { label: productDetail.shortName },
          ]}
        />
      </div>

      <section className="w-full max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop py-8 lg:py-12">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-gutter-desktop items-start">
          <ProductGallery />
          <PurchasePanel onOpenSizeGuide={() => setSizeGuideOpen(true)} />
        </div>
      </section>

      <InfoTabs />
      <ReviewsSection />
      <AccessoriesSection />
      <SizeGuideModal open={sizeGuideOpen} onClose={() => setSizeGuideOpen(false)} />
    </div>
  )
}
