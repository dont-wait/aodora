import { useState } from 'react'
import Icon from '@/components/ui/Icon'
import { craftSteps, fabricCare, infoTabs, productDetail, type InfoTab } from '@/data/productDetail'

const label = 'font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase'
const title = 'font-headline-md text-headline-md text-primary font-normal'
const body = 'font-body-md text-body-md text-on-surface-variant leading-relaxed'

export default function InfoTabs() {
  const [active, setActive] = useState<InfoTab['id']>('story')

  return (
    <section className="w-full bg-surface-container-low py-16">
      <div className="max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop">
        <div role="tablist" className="flex items-center justify-center gap-2 sm:gap-6 pb-6 flex-wrap">
          {infoTabs.map((t) => (
            <button
              key={t.id}
              type="button"
              role="tab"
              aria-selected={active === t.id}
              aria-controls={`tab-pane-${t.id}`}
              onClick={() => setActive(t.id)}
              className={`py-2.5 px-4 sm:px-6 font-title-editorial text-title-editorial transition-all ${
                active === t.id ? 'bg-surface-container-highest text-primary font-bold shadow-sm' : 'text-on-surface-variant hover:text-on-surface'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        <div id={`tab-pane-${active}`} role="tabpanel" className="bg-surface-container p-6 sm:p-10 shadow-sm mt-2">
          {active === 'story' && (
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
              <div className="lg:col-span-7 flex flex-col gap-4">
                <span className={label}>Cốt Cách Áo Dài Di Sản</span>
                <h2 className={title}>Biểu Tượng Đạo Lý Ngũ Thường Trong Tà Áo Dài Ngũ Thân</h2>
                <p className={body}>
                  Áo dài ngũ thân ra đời dưới thời chúa Nguyễn Phúc Khoát và được hoàn thiện chuẩn mực vào triều vua Minh Mạng (1837). Năm thân áo tượng trưng
                  cho tứ thân phụ mẫu (cha mẹ đẻ và cha mẹ chồng) che chở cho thân thứ năm ở trong lòng – biểu tượng của người mặc.
                </p>
                <p className={body}>
                  Năm chiếc cúc ngọc cài khép kín cổ lập lĩnh là hiện thân của năm đức hạnh cao quý của người Việt: <em>Nhân, Lễ, Nghĩa, Trí, Tín</em>. Sắc sen
                  trầm mang lại nét thanh cao, đằm thắm của người phụ nữ Tràng An xưa, kín đáo nhưng toát lên vẻ quyền quý, tự tại.
                </p>
                <div className="flex items-center gap-6 pt-2 flex-wrap">
                  {[
                    ['1837', 'Chuẩn hoá sắc phục triều đình'],
                    ['5 Tà Thân', 'Hiếu kính đạo phụ mẫu'],
                    ['5 Cúc Ngọc', 'Ngũ thường vẹn toàn'],
                  ].map(([v, d], i) => (
                    <div key={v} className="flex items-center gap-6">
                      {i > 0 && <div className="h-8 w-px bg-outline-variant" />}
                      <div>
                        <span className="block font-headline-sm text-headline-sm text-primary font-bold">{v}</span>
                        <span className="font-body-sm text-body-sm text-on-surface-variant">{d}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
              <PaneImage colSpan="lg:col-span-5" ratio="aspect-[4/5]" image={productDetail.storyImage} />
            </div>
          )}

          {active === 'fabric' && (
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
              <div className="lg:col-span-6 flex flex-col gap-4">
                <span className={label}>Sợi Tơ Tự Nhiên Làng Vạn Phúc</span>
                <h2 className={title}>100% Lụa Tơ Tằm Dệt Tay – Thở Cùng Làn Da</h2>
                <p className={body}>
                  Tấm lụa Sắc Sen Trầm được ươm tơ và dệt theo kỹ thuật hoa văn chìm truyền thống tại làng lụa Vạn Phúc (Hà Đông). Với định lượng 16 Momme cao
                  cấp, bề mặt lụa có độ óng ả tự nhiên, thoáng mát vào mùa hạ và giữ ấm nhẹ nhàng khi tiết trời thu đông.
                </p>
                <div className="space-y-3 pt-2">
                  {fabricCare.map((c) => (
                    <div key={c.title} className="flex items-start gap-3 bg-surface-container-low p-3">
                      <Icon name={c.icon} size={22} className="text-secondary" />
                      <div>
                        <h3 className="font-label-regular text-label-regular font-bold text-on-surface">{c.title}</h3>
                        <p className="font-body-sm text-body-sm text-on-surface-variant">{c.description}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
              <PaneImage colSpan="lg:col-span-6" ratio="aspect-video sm:aspect-[4/3]" image={productDetail.fabricImage} />
            </div>
          )}

          {active === 'craft' && (
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
              <div className="lg:col-span-7 flex flex-col gap-4">
                <span className={label}>Nghệ Thuật Cắt May Cổ Điển</span>
                <h2 className={title}>Kỹ Thuật Cắt Tà Xéo & Giấu Chỉ Độc Bản</h2>
                <p className={body}>
                  Mỗi bộ Áo Dài Ngũ Thân tại xưởng may AN SOIE đòi hỏi hơn 18 giờ làm việc thủ công từ người nghệ nhân lâu năm. Từng đường ráp tà theo canh vải
                  xéo giúp tà áo có độ rủ tự nhiên hình lưỡi đao mà không bao giờ bị hếch khi di chuyển.
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
                  {craftSteps.map((s) => (
                    <div key={s.title} className="bg-surface-container-low p-4">
                      <span className="font-label-uppercase text-label-uppercase text-primary font-bold">{s.title}</span>
                      <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">{s.description}</p>
                    </div>
                  ))}
                </div>
              </div>
              <PaneImage colSpan="lg:col-span-5" ratio="aspect-[4/5]" image={productDetail.craftImage} />
            </div>
          )}
        </div>
      </div>
    </section>
  )
}

function PaneImage({ image, colSpan, ratio }: { image: { src: string; alt: string }; colSpan: string; ratio: string }) {
  return (
    <div className={colSpan}>
      <div className={`relative ${ratio} bg-surface-container-high overflow-hidden shadow-sm`}>
        <img src={image.src} alt={image.alt} loading="lazy" className="w-full h-full object-cover" />
      </div>
    </div>
  )
}
