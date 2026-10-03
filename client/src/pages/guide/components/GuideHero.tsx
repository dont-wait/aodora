import { guideImages } from '@/data/guide'

export default function GuideHero() {
  return (
    <section className="relative w-full overflow-hidden bg-surface-container-low px-margin-tablet lg:px-margin-desktop py-16 lg:py-24">
      <div className="absolute inset-0 pointer-events-none opacity-40 bg-[radial-gradient(circle_at_20%_30%,rgba(122,28,41,0.08)_0%,transparent_70%)]"></div>
      <div className="absolute -right-24 -top-24 w-96 h-96 rounded-full bg-secondary-container/20 blur-3xl pointer-events-none"></div>
      <div className="max-w-[1360px] mx-auto relative z-10 grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
        <div className="lg:col-span-7 flex flex-col gap-6">
          <div className="flex items-center gap-3">
            <span className="inline-block w-8 h-[1px] bg-secondary"></span>
            <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Atelier Bespoke & Heritage Craft</span>
          </div>
          <h1 className="font-display-hero text-headline-lg lg:text-display-hero text-primary tracking-tight font-normal leading-tight">
            Cốt Cách Vừa Vặn — <br className="hidden sm:inline" />
            <span className="italic font-light text-on-surface">Nghệ Thuật May Đo</span> Áo Dài Riêng Biệt
          </h1>
          <p className="font-body-lg text-body-lg text-on-surface-variant max-w-2xl leading-relaxed">
            Chiếc áo dài đẹp nhất là chiếc áo ôm vừa vặn từng đường cong tự nhiên, nâng đỡ phom dáng nhẹ bẫng và giúp người mặc luôn tự tin, thanh thoát trong
            từng bước chân qua năm tháng.
          </p>
          <div className="flex flex-wrap items-center gap-4 pt-4">
            <a
              className="inline-flex items-center justify-center px-8 py-3.5 bg-primary text-on-primary font-label-uppercase text-label-uppercase tracking-widest shadow-md hover:bg-primary-container transition-colors"
              href="#bang-so-do"
            >
              Khám phá 8 số đo vàng
            </a>
            <a
              className="inline-flex items-center justify-center px-7 py-3.5 bg-surface text-on-surface font-label-uppercase text-label-uppercase tracking-widest hover:bg-surface-container transition-colors shadow-sm"
              href="#dat-lich-tan-noi"
            >
              <span className="material-symbols-outlined text-[18px] mr-2 text-secondary">concierge</span>
              Hẹn đo tại tư gia
            </a>
          </div>
        </div>
        <div className="lg:col-span-5 relative">
          <div className="relative mx-auto w-full aspect-[4/5] overflow-hidden bg-surface-container shadow-xl">
            <img src={guideImages.hero.src} alt={guideImages.hero.alt} className="w-full h-full object-cover" />
            <div className="absolute inset-0 bg-gradient-to-t from-primary/60 via-transparent to-transparent"></div>
            <div className="absolute bottom-6 left-6 right-6 text-on-primary">
              <span className="font-label-uppercase text-label-uppercase tracking-widest text-secondary-fixed opacity-90 block mb-1">
                Quy chuẩn thủ công Hà Đông
              </span>
              <p className="font-title-editorial text-title-editorial italic">"Từng tấc vải tơ tằm là một nhịp thở, chỉ đo chuẩn mới tạo nên tà áo bay."</p>
            </div>
          </div>
          <div className="hidden sm:flex absolute -bottom-6 -left-8 bg-surface-bright p-5 shadow-lg max-w-[220px] flex-col gap-1">
            <div className="flex items-center gap-2 text-secondary">
              <span className="material-symbols-outlined text-[20px]">straighten</span>
              <span className="font-label-uppercase text-label-uppercase tracking-widest">Độ Chuẩn Xác</span>
            </div>
            <p className="font-headline-sm text-headline-sm text-primary font-semibold m-0">
              1.0 <span className="text-xs text-on-surface-variant font-normal">milimet</span>
            </p>
            <span className="font-body-sm text-body-sm text-on-surface-variant">Tối ưu cho từng cử động thở và bước đi</span>
          </div>
        </div>
      </div>
    </section>
  )
}
