import Breadcrumb from '@/components/ui/Breadcrumb'
import { ROUTES } from '@/data/site'

const STATS = [
  { value: '100%', label: 'Tơ tằm dệt thủ công', accent: 'text-primary' },
  { value: '48 Mẫu', label: 'Độc bản & Tinh hoa', accent: 'text-primary' },
  { value: 'Bespoke 1:1', label: 'May đo chuẩn nhân trắc', accent: 'text-secondary' },
]

export default function CollectionHero() {
  return (
    <section className="relative w-full bg-surface-container-low px-margin md:px-margin-desktop py-12 md:py-20 overflow-hidden">
      <div className="absolute -top-32 -right-24 w-96 h-96 rounded-full bg-primary/5 blur-3xl pointer-events-none" />
      <div className="absolute -bottom-24 -left-20 w-80 h-80 rounded-full bg-secondary-container/20 blur-2xl pointer-events-none" />
      <div className="max-w-[1440px] mx-auto relative z-10 flex flex-col items-center text-center">
        <Breadcrumb
          className="mb-6 text-on-surface-variant font-label-regular text-label-regular tracking-wider"
          separatorClass="text-outline text-[10px]"
          items={[{ label: 'Trang chủ', to: ROUTES.home }, { label: 'Bộ sưu tập Áo Dài', to: ROUTES.collection }, { label: 'Tất cả sản phẩm' }]}
        />
        <span className="inline-block px-3 py-1 bg-surface-container text-secondary font-label-uppercase text-label-uppercase tracking-[0.2em] mb-4">
          Haute Couture & Di Sản Dệt May
        </span>
        <h1 className="font-headline-lg text-headline-lg md:text-[48px] md:leading-[56px] text-primary tracking-tight max-w-4xl mb-6">
          Bộ Sưu Tập Áo Dài Di Sản & Đương Đại
        </h1>
        <p className="font-body-lg text-body-lg text-on-surface-variant max-w-3xl leading-relaxed font-light">
          Thưởng lãm tinh hoa lụa tơ tằm nguyên bản, nơi sắc thái dịu lành của tơ sợi tự nhiên hòa cùng đường may đo đĩnh đạc, chắt lọc cốt cách truyền thống và
          nhịp sống thanh tân.
        </p>
        <div className="mt-10 pt-8 flex flex-wrap justify-center items-center gap-8 md:gap-16 text-on-surface">
          {STATS.map((s, i) => (
            <div key={s.label} className="flex items-center gap-8 md:gap-16">
              {i > 0 && <span className="w-1.5 h-1.5 rounded-full bg-outline-variant hidden sm:block" />}
              <div className="flex flex-col items-center">
                <span className={`font-title-editorial text-title-editorial font-bold ${s.accent}`}>{s.value}</span>
                <span className="font-label-uppercase text-label-uppercase text-on-surface-variant mt-1">{s.label}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
