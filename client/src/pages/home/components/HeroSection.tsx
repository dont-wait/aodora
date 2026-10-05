import { heroImages, heroStats } from '@/data/home'

export default function HeroSection() {
  return (
    <section className="relative w-full overflow-hidden bg-surface py-12 md:py-20 lg:py-24">
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-gutter-desktop items-center">
          <div className="lg:col-span-6 flex flex-col justify-center space-y-6 lg:space-y-8 z-10">
            <div className="inline-flex items-center gap-2 self-start bg-surface-container px-3 py-1.5 rounded-full">
              <span className="w-2 h-2 rounded-full bg-primary animate-pulse"></span>
              <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Bộ Sưu Tập Mùa Thu 2025</span>
            </div>
            <div className="space-y-4">
              <h1 className="font-headline-lg lg:font-display-hero text-headline-lg lg:text-display-hero text-primary tracking-tight leading-none">
                Dáng Việt Trong Lụa
                <br />
                <span className="font-display-hero italic font-normal text-on-surface">Tinh Hoa Ngàn Năm</span>
              </h1>
              <p className="font-body-lg text-body-lg text-on-surface-variant max-w-xl font-light leading-relaxed">
                Tôn vinh vẻ đẹp thuần hậu và khí chất phụ nữ Việt qua từng đường kim, thớ lụa dệt thủ công làng nghề truyền thống. Di sản dệt may hòa quyện
                trong hơi thở thời đại đương nhịp.
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-4 pt-2">
              <a
                className="inline-flex items-center justify-center bg-primary hover:bg-primary-container text-on-primary font-label-uppercase text-label-uppercase px-8 py-4 transition-all duration-300 shadow-md hover:shadow-lg"
                href="#bo-suu-tap-moi"
              >
                <span>Khám Phá Bộ Sưu Tập</span>
                <span className="material-symbols-outlined ml-2 text-[18px]">arrow_forward</span>
              </a>
              <a
                className="inline-flex items-center justify-center bg-surface-container-low hover:bg-surface-container text-on-surface font-label-uppercase text-label-uppercase px-8 py-4 transition-all duration-300 shadow-sm"
                href="#dat-may-do"
              >
                <span className="material-symbols-outlined mr-2 text-[18px] text-secondary">straighten</span>
                <span>Đặt Lịch May Đo Riêng</span>
              </a>
            </div>

            <div className="grid grid-cols-3 gap-4 pt-6 bg-surface-container-low p-6 rounded-lg shadow-sm">
              {heroStats.map((stat) => (
                <div key={stat.label} className="space-y-1">
                  <span className="block font-headline-md text-headline-md text-primary font-semibold">{stat.value}</span>
                  <span className="block font-body-sm text-body-sm text-on-surface-variant leading-snug">{stat.label}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="lg:col-span-6 relative flex justify-center items-center">
            <div className="relative w-full aspect-[4/5] max-w-lg lg:max-w-none shadow-xl overflow-hidden bg-surface-container-highest">
              <img
                src={heroImages.main.src}
                alt={heroImages.main.alt}
                className="w-full h-full object-cover transition-transform duration-700 hover:scale-105"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-primary/70 via-transparent to-transparent"></div>

              <div className="absolute bottom-6 left-6 right-6 p-5 bg-surface/90 backdrop-blur-md shadow-lg flex items-center justify-between">
                <div>
                  <p className="font-label-uppercase text-label-uppercase text-secondary">Tuyệt Tác Thêu Độc Bản</p>
                  <h3 className="font-headline-sm text-headline-sm text-on-surface">Nguyệt Dạ Vân Tơ '25</h3>
                </div>
                <span className="font-title-editorial text-title-editorial text-primary font-bold">Kén tằm tự nhiên</span>
              </div>
            </div>

            <div className="hidden sm:block absolute -bottom-8 -left-8 w-44 h-56 shadow-2xl overflow-hidden bg-surface-container z-20">
              <img src={heroImages.detail.src} alt={heroImages.detail.alt} className="w-full h-full object-cover" />
              <div className="absolute inset-0 bg-secondary/10"></div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
