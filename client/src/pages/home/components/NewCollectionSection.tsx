import { Link } from 'react-router-dom'
import Badge from '@/components/ui/Badge'
import Icon from '@/components/ui/Icon'
import { featuredProducts } from '@/data/home'
import { ROUTES } from '@/data/site'
import { formatVND } from '@/utils/format'
import { useToast } from '@/hooks/useToast'

const QUICK_SIZES = ['S', 'M', 'L']

export default function NewCollectionSection() {
  const { showToast } = useToast()

  return (
    <section className="w-full py-16 lg:py-24 bg-surface" id="bo-suu-tap-moi">
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
        <div className="flex flex-col md:flex-row md:items-end justify-between mb-12 gap-6">
          <div>
            <div className="flex items-center gap-3 mb-2">
              <span className="h-px w-8 bg-secondary" />
              <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Ấn Phẩm Thu Đông 2025</span>
            </div>
            <h2 className="font-headline-lg text-headline-lg text-primary">Bộ Sưu Tập 'Nguyệt Dạ Vân Tơ'</h2>
            <p className="font-body-md text-body-md text-on-surface-variant mt-2">
              Dải lụa mềm soi bóng trăng khuya, lưu giữ đường nét ngọc ngà của người phụ nữ Việt.
            </p>
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 lg:gap-gutter">
          {featuredProducts.map((p) => (
            <article key={p.slug} className="group relative flex flex-col bg-surface shadow-sm hover:shadow-xl transition-all duration-300">
              <div className="relative aspect-[3/4] overflow-hidden bg-surface-container">
                <Link to={ROUTES.product(p.slug)}>
                  <img
                    src={p.image.src}
                    alt={p.image.alt}
                    loading="lazy"
                    className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-105"
                  />
                </Link>
                <Badge variant={p.variant} className="absolute top-3 left-3 px-2.5 py-0.5">
                  {p.badge}
                </Badge>
                <button
                  type="button"
                  aria-label="Thêm vào yêu thích"
                  onClick={() => showToast(`Đã lưu "${p.name}" vào danh sách yêu thích.`)}
                  className="absolute top-3 right-3 w-9 h-9 bg-surface/80 hover:bg-surface text-on-surface flex items-center justify-center transition-colors shadow-sm"
                >
                  <Icon name="favorite" size={18} />
                </button>

                <div className="absolute inset-x-0 bottom-0 p-4 bg-surface/95 backdrop-blur-md translate-y-full group-hover:translate-y-0 transition-transform duration-300 flex flex-col gap-2">
                  <div className="flex items-center justify-between text-body-sm font-label-regular text-on-surface-variant">
                    <span>Chọn số đo:</span>
                    <Link to={ROUTES.tailoring} className="text-primary hover:underline">
                      Bảng quy chuẩn
                    </Link>
                  </div>
                  <div className="flex gap-2">
                    {QUICK_SIZES.map((s) => (
                      <span key={s} className="flex-1 py-1 text-center bg-surface-container text-on-surface font-label-regular text-body-sm">
                        {s}
                      </span>
                    ))}
                    <span className="flex-1 py-1 text-center bg-secondary-container text-on-secondary-container font-label-regular text-body-sm">
                      May riêng
                    </span>
                  </div>
                  <Link
                    to={ROUTES.product(p.slug)}
                    className="w-full bg-primary text-on-primary py-2 mt-1 text-center font-label-uppercase text-label-uppercase hover:bg-primary-container transition-colors"
                  >
                    Chọn Áo Này
                  </Link>
                </div>
              </div>

              <div className="p-5 flex flex-col space-y-2">
                <span className="font-label-uppercase text-label-uppercase text-secondary tracking-wider">{p.material}</span>
                <h3 className="font-headline-sm text-headline-sm text-on-surface line-clamp-1 group-hover:text-primary transition-colors">{p.name}</h3>
                <p className="font-body-sm text-body-sm text-on-surface-variant line-clamp-1">{p.description}</p>
                <div className="pt-2 flex items-baseline justify-between">
                  <span className="font-title-editorial text-title-editorial text-primary font-bold">{formatVND(p.price)}</span>
                  {p.oldPrice ? (
                    <span className="font-body-sm text-body-sm text-outline line-through">{formatVND(p.oldPrice)}</span>
                  ) : (
                    <span className="font-label-regular text-body-sm text-secondary">{p.availability}</span>
                  )}
                </div>
              </div>
            </article>
          ))}
        </div>

        <div className="mt-12 text-center">
          <Link
            to={ROUTES.collection}
            className="inline-flex items-center gap-2 bg-surface-container hover:bg-surface-container-high text-primary px-8 py-3.5 font-label-uppercase text-label-uppercase transition-colors"
          >
            <span>Xem Toàn Bộ 48 Mẫu Áo Dài Di Sản</span>
            <Icon name="south_east" size={18} />
          </Link>
        </div>
      </div>
    </section>
  )
}
