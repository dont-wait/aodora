import StarRating from '@/components/ui/StarRating'
import { productDetail, reviews } from '@/data/productDetail'

export default function ReviewsSection() {
  return (
    <section id="danh-gia-khach-hang" className="w-full max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop py-16">
      <div className="flex flex-col md:flex-row md:items-end justify-between mb-10 gap-4">
        <div>
          <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Khách Hàng Đã Trải Nghiệm</span>
          <h2 className="font-headline-lg text-headline-lg text-primary font-normal mt-1">Vẻ Đẹp Thực Tế & Lời Cảm Nhận</h2>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex flex-col text-right">
            <span className="font-headline-sm text-headline-sm text-primary font-bold">{productDetail.reviewCount} Lời Nhận Xét</span>
            <span className="font-body-sm text-body-sm text-on-surface-variant">100% Khách hàng hài lòng về chất vải</span>
          </div>
          <button
            type="button"
            className="bg-surface-container hover:bg-surface-container-high text-primary px-4 py-2.5 font-label-uppercase text-label-uppercase tracking-wider transition-colors shadow-sm"
          >
            Viết Đánh Giá
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {reviews.map((r) => (
          <article key={r.id} className="bg-surface-container-low p-6 flex flex-col justify-between shadow-sm">
            <div className="flex flex-col gap-4">
              <StarRating size={16} />
              <p className="font-body-md text-body-md text-on-surface leading-relaxed italic">"{r.content}"</p>
              <div className="relative aspect-square w-full bg-surface-container overflow-hidden mt-1 shadow-sm">
                <img src={r.image.src} alt={r.image.alt} loading="lazy" className="w-full h-full object-cover" />
              </div>
            </div>
            <div className="flex items-center gap-3 pt-4 mt-4">
              <div className="w-10 h-10 rounded-full bg-surface-container-highest flex items-center justify-center font-bold text-primary">{r.initials}</div>
              <div className="flex flex-col">
                <span className="font-label-regular text-label-regular font-bold text-on-surface">{r.author}</span>
                <span className="font-body-sm text-body-sm text-secondary">{r.meta}</span>
              </div>
            </div>
          </article>
        ))}
      </div>
    </section>
  )
}
