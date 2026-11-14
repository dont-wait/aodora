import StarRating from '@/components/ui/StarRating'
import type { CatalogProduct } from '@/features/catalog/types'
export default function ReviewsSection({ summary }: { summary: CatalogProduct['reviews'] }) {
  return <section id="danh-gia-khach-hang" className="w-full max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop py-16">
    <p className="uppercase tracking-widest text-secondary text-sm">Đánh giá khách hàng</p>
    <h2 className="font-headline-lg text-headline-lg text-primary mt-2">Đánh Giá Sản Phẩm</h2>
    <div className="flex items-center gap-4 mt-5">
      {summary.count > 0 ? <><StarRating value={Math.round(summary.average)} />
        <span>{summary.average.toFixed(1)}/5 từ {summary.count} đánh giá</span></> : <p className="text-on-surface-variant">Chưa có đánh giá cho mẫu áo này.</p>}
    </div>
    <p className="text-on-surface-variant text-sm mt-3">Nội dung nhận xét và chức năng gửi đánh giá sẽ được kết nối khi module đánh giá hoàn thiện.</p>
  </section>
}
