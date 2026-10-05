import { guideImages } from '@/data/guide'

export default function HeritageSection() {
  return (
    <section className="w-full py-24 px-margin-tablet lg:px-margin-desktop bg-surface-container-high relative overflow-hidden">
      <div className="max-w-[1360px] mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 lg:gap-16 items-center">
          <div className="lg:col-span-6 grid grid-cols-2 gap-4 relative">
            <div className="space-y-4">
              <div className="aspect-[3/4] bg-surface overflow-hidden shadow-lg">
                <img src={guideImages.heritageA.src} alt={guideImages.heritageA.alt} className="w-full h-full object-cover" />
              </div>
              <div className="p-6 bg-surface-bright shadow-sm">
                <span className="font-headline-sm text-headline-sm text-primary font-semibold block">40+ Năm</span>
                <span className="font-body-sm text-body-sm text-on-surface-variant">Một đời bền bỉ giữ trọn đường kim tơ tằm nguyên bản</span>
              </div>
            </div>
            <div className="space-y-4 pt-10">
              <div className="p-6 bg-primary text-on-primary shadow-sm">
                <span className="font-label-uppercase text-label-uppercase tracking-widest text-secondary-fixed block mb-1">Làng Lụa Vạn Phúc</span>
                <p className="font-title-editorial text-title-editorial italic">Hà Đông nghìn năm dệt hồn văn hóa dân tộc</p>
              </div>
              <div className="aspect-[3/4] bg-surface overflow-hidden shadow-lg">
                <img src={guideImages.heritageB.src} alt={guideImages.heritageB.alt} className="w-full h-full object-cover" />
              </div>
            </div>
          </div>

          <div className="lg:col-span-6 flex flex-col gap-6">
            <div className="flex items-center gap-3">
              <span className="inline-block w-8 h-[1px] bg-secondary"></span>
              <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Hành trình di sản</span>
            </div>
            <h2 className="font-headline-lg text-headline-lg text-primary tracking-tight leading-tight">Về Lại Cái Nôi Lụa Vạn Phúc & Đất Thêu Quất Động</h2>
            <p className="font-body-md text-body-md text-on-surface-variant leading-relaxed">
              Mỗi thước vải tại AN SOIE không đơn thuần là phục trang may sẵn, mà là kết tinh từ đôi bàn tay gầy guộc của những nghệ nhân ưu tú đã gắn bó cả
              cuộc đời với khung cửi bên dòng sông Nhuệ và nghệ thuật thêu tay cung đình Quất Động. Tơ tằm được quay sợi tự nhiên, nhuộm bằng thảo mộc cổ truyền
              để giữ nguyên độ óng ánh dịu dàng và cảm giác mát rượi khi chạm vào làn da.
            </p>
            <div className="mt-4 p-6 bg-surface shadow-sm space-y-4">
              <div className="flex items-center gap-2 text-primary font-label-uppercase text-label-uppercase tracking-widest pb-2">
                <span className="material-symbols-outlined text-[20px] text-secondary">verified</span>
                <span>Bản Cam Kết '3 KHÔNG' Từ AN SOIE</span>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div className="bg-surface-container p-4 flex flex-col gap-2">
                  <span className="font-label-uppercase text-label-uppercase text-primary font-bold">1. KHÔNG PHA TẠP</span>
                  <p className="font-body-sm text-body-sm text-on-surface-variant">
                    Tuyệt đối không dùng sợi poly hay nilon công nghiệp. 100% tơ tằm dệt thủ công.
                  </p>
                </div>
                <div className="bg-surface-container p-4 flex flex-col gap-2">
                  <span className="font-label-uppercase text-label-uppercase text-primary font-bold">2. KHÔNG CÔNG NGHIỆP</span>
                  <p className="font-body-sm text-body-sm text-on-surface-variant">
                    Không may ẩu dây chuyền rập khuôn. Từng chiếc áo là một bản rập độc bản cho riêng một người.
                  </p>
                </div>
                <div className="bg-surface-container p-4 flex flex-col gap-2">
                  <span className="font-label-uppercase text-label-uppercase text-primary font-bold">3. KHÔNG MẤT CHẤT</span>
                  <p className="font-body-sm text-body-sm text-on-surface-variant">
                    Không lạm dụng hóa chất màu mè sặc sỡ làm biến chất độ mềm mộc và hương thơm tơ lụa.
                  </p>
                </div>
              </div>
            </div>
            <div className="flex items-center gap-4 pt-2">
              <div className="w-12 h-12 rounded-full bg-surface overflow-hidden shadow-sm shrink-0">
                <img src={guideImages.artisan.src} alt={guideImages.artisan.alt} className="w-full h-full object-cover" />
              </div>
              <div>
                <p className="font-title-editorial text-title-editorial font-bold text-on-surface m-0">Nghệ nhân Ưu tú Nguyễn Thị Thương</p>
                <span className="font-body-sm text-body-sm text-on-surface-variant">Cố vấn trưởng kỹ thuật cắt may tà áo truyền thống AN SOIE</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
