import Breadcrumb from '@/components/ui/Breadcrumb'
import { ROUTES } from '@/data/site'

export default function CollectionHero({ total }: { total?: number }) {
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
        {total !== undefined && <p className="mt-8 text-primary font-semibold">{total} sản phẩm phù hợp</p>}
      </div>
    </section>
  )
}
