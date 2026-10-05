export default function PreparationSection() {
  return (
    <section className="w-full py-16 px-margin-tablet lg:px-margin-desktop bg-surface">
      <div className="max-w-[1360px] mx-auto">
        <div className="p-8 lg:p-12 bg-surface-container rounded-none shadow-sm flex flex-col md:flex-row items-center justify-between gap-8">
          <div className="flex items-start gap-5">
            <div className="w-14 h-14 shrink-0 bg-primary text-on-primary flex items-center justify-center">
              <span className="material-symbols-outlined text-[28px]">inventory_2</span>
            </div>
            <div className="flex flex-col gap-2">
              <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest">Bước Khởi Đầu Quan Trọng</span>
              <h2 className="font-headline-sm text-headline-sm text-on-surface">Chuẩn bị trước khi tự lấy số đo tại nhà</h2>
              <p className="font-body-md text-body-md text-on-surface-variant max-w-2xl leading-relaxed">
                Hãy chuẩn bị 01 thước dây may đo sợi mềm (loại cm). Khi đo, nên mặc đồ lót vừa vặn, mềm mại hoặc chính chiếc áo nịt ngực quý khách dự kiến kết
                hợp cùng Áo Dài để thông số phom ngực và eo chuẩn xác nhất. Đứng thẳng tự nhiên, thở đều, không hóp bụng.
              </p>
            </div>
          </div>
          <div className="flex flex-col sm:flex-row gap-3 shrink-0">
            <div className="flex items-center gap-3 bg-surface-bright px-5 py-3 shadow-sm">
              <span className="material-symbols-outlined text-primary text-[22px]">check_circle</span>
              <span className="font-body-sm text-body-sm text-on-surface">Thước dây vải mềm</span>
            </div>
            <div className="flex items-center gap-3 bg-surface-bright px-5 py-3 shadow-sm">
              <span className="material-symbols-outlined text-primary text-[22px]">check_circle</span>
              <span className="font-body-sm text-body-sm text-on-surface">Nội y chuẩn phom</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
