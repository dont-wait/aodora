import Icon from '@/components/ui/Icon'
import StarRating from '@/components/ui/StarRating'
import { heroTestimonial, testimonials, wideTestimonial } from '@/data/home'

export default function TestimonialsSection() {
  return (
    <section className="w-full py-16 lg:py-24 bg-surface overflow-hidden">
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
          <div className="lg:col-span-5 space-y-6">
            <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Tiếng Nói Người Yêu Lụa</span>
            <h2 className="font-headline-lg text-headline-lg text-primary leading-tight">"Áo Dài Không Chỉ Là Trang Phục, Đó Là Căn Cước Văn Hóa."</h2>
            <figure className="relative bg-surface-container-low p-8 shadow-sm">
              <Icon name="format_quote" size={48} className="text-secondary/30 absolute -top-4 -left-2" />
              <blockquote className="font-body-lg text-body-lg text-on-surface italic relative z-10 leading-relaxed">{heroTestimonial.quote}</blockquote>
              <figcaption className="mt-6 flex items-center gap-4">
                <img
                  src={heroTestimonial.avatar.src}
                  alt={heroTestimonial.avatar.alt}
                  className="w-12 h-12 rounded-full object-cover bg-surface-container-high"
                />
                <div>
                  <span className="block font-headline-sm text-headline-sm text-primary">{heroTestimonial.name}</span>
                  <span className="block font-body-sm text-body-sm text-on-surface-variant">{heroTestimonial.role}</span>
                </div>
              </figcaption>
            </figure>
          </div>

          <div className="lg:col-span-7 grid grid-cols-1 md:grid-cols-2 gap-6">
            {testimonials.map((t) => (
              <article key={t.name} className="bg-surface-container p-6 flex flex-col justify-between shadow-sm">
                <div className="space-y-4">
                  <StarRating />
                  <p className="font-body-md text-body-md text-on-surface leading-relaxed">"{t.content}"</p>
                </div>
                <div className="mt-6 pt-4 flex items-center justify-between">
                  <div>
                    <strong className="block font-label-regular text-label-regular text-on-surface">{t.name}</strong>
                    <span className="font-body-sm text-body-sm text-on-surface-variant">{t.role}</span>
                  </div>
                  <span className="font-label-uppercase text-label-uppercase text-secondary bg-surface px-2 py-1">{t.tag}</span>
                </div>
              </article>
            ))}

            <article className="bg-surface-container p-6 flex flex-col justify-between shadow-sm md:col-span-2">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div className="flex items-center gap-4">
                  <img
                    src={wideTestimonial.avatar.src}
                    alt={wideTestimonial.avatar.alt}
                    className="w-12 h-12 rounded-full object-cover bg-surface-container-high flex-shrink-0"
                  />
                  <div>
                    <strong className="block font-label-regular text-label-regular text-on-surface">{wideTestimonial.name}</strong>
                    <span className="font-body-sm text-body-sm text-on-surface-variant">{wideTestimonial.role}</span>
                  </div>
                </div>
                <StarRating />
              </div>
              <p className="font-body-md text-body-md text-on-surface mt-4 leading-relaxed">"{wideTestimonial.content}"</p>
            </article>
          </div>
        </div>
      </div>
    </section>
  )
}
