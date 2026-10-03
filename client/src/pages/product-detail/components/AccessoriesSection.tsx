import Badge from '@/components/ui/Badge'
import Icon from '@/components/ui/Icon'
import { accessories } from '@/data/productDetail'
import { useToast } from '@/hooks/useToast'
import { formatVND } from '@/utils/format'

export default function AccessoriesSection() {
  const { showToast } = useToast()
  return (
    <section className="w-full bg-surface-container py-16">
      <div className="max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop">
        <div className="flex flex-col items-center text-center mb-10">
          <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Gợi Ý Phối Đồ Chuẩn Cốt Cách</span>
          <h2 className="font-headline-lg text-headline-lg text-primary font-normal mt-1">Hoàn Thiện Bộ Cổ Phục Ngũ Thân</h2>
          <p className="font-body-md text-body-md text-on-surface-variant max-w-xl mt-2">
            Những món phụ kiện truyền thống tinh tế được chế tác đồng điệu với chất liệu lụa tơ tằm nguyên bản.
          </p>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
          {accessories.map((a) => (
            <article key={a.id} className="bg-surface p-5 flex flex-col justify-between shadow-sm group">
              <div>
                <div className="relative aspect-square w-full bg-surface-container-low overflow-hidden mb-4">
                  <img
                    src={a.image.src}
                    alt={a.image.alt}
                    loading="lazy"
                    className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                  />
                  <Badge variant={a.tagVariant === 'secondaryFixed' ? 'secondaryFixed' : 'surface'} className="absolute top-3 left-3 px-2 py-0.5 font-bold">
                    {a.tag}
                  </Badge>
                </div>
                <h3 className="font-headline-sm text-headline-sm text-on-surface group-hover:text-primary transition-colors">{a.name}</h3>
                <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">{a.description}</p>
              </div>
              <div className="flex items-center justify-between pt-4 mt-4">
                <span className="font-title-editorial text-title-editorial text-primary font-bold">{formatVND(a.price).replace('đ', '₫')}</span>
                <button
                  type="button"
                  onClick={() => showToast(`Đã thêm ${a.name} (${formatVND(a.price).replace('đ', '₫')}) vào đơn may.`)}
                  className="bg-surface-container hover:bg-primary hover:text-on-primary text-primary px-3 py-1.5 font-label-uppercase text-label-uppercase tracking-wider transition-colors flex items-center gap-1"
                >
                  <Icon name="keyboard_double_arrow_left" size={16} />
                  Thêm Phụ Kiện
                </button>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}
