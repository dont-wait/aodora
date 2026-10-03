import EyebrowHeading from '@/components/ui/EyebrowHeading'
import Icon from '@/components/ui/Icon'
import { pillars } from '@/data/home'

export default function PillarsSection() {
  return (
    <section className="w-full py-16 lg:py-24 bg-surface-container">
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
        <EyebrowHeading
          align="center"
          className="max-w-2xl mb-16 space-y-3"
          eyebrow="Kỳ Công Từng Đường Kim"
          title="Triết Lý Nghệ Thuật & Chất Liệu"
          description="Mỗi tác phẩm Áo Dài tại AN SOIE là một kiệt tác dung hòa giữa hồn tơ truyền thống, sự khắt khe của tay nghề thủ công và phom dáng tôn vinh tuyệt đối cốt cách người mặc."
        />
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {pillars.map((p, i) => (
            <article key={p.title} className="bg-surface p-8 shadow-sm flex flex-col justify-between group hover:shadow-xl transition-all duration-300">
              <div className="space-y-6">
                <div className="w-14 h-14 bg-surface-container-low text-primary flex items-center justify-center rounded">
                  <Icon name={p.icon} size={32} />
                </div>
                <div className="aspect-[16/10] overflow-hidden bg-surface-container">
                  <img
                    src={p.image.src}
                    alt={p.image.alt}
                    loading="lazy"
                    className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
                  />
                </div>
                <div className="space-y-3">
                  <h3 className="font-headline-sm text-headline-sm text-on-surface">{p.title}</h3>
                  <p className="font-body-md text-body-md text-on-surface-variant leading-relaxed">{p.description}</p>
                </div>
              </div>
              <div className="mt-8 pt-4 flex items-center text-secondary font-label-uppercase text-label-uppercase gap-2">
                <span>{p.label}</span>
                <span className="h-px flex-1 bg-secondary/30" />
                <span>{String(i + 1).padStart(2, '0')}</span>
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}
