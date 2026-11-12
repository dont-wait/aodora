import { useState } from 'react'
import type { CatalogProduct, CatalogContent } from '@/features/catalog/types'
import CatalogImage from '@/features/catalog/CatalogImage'

type SectionName = 'story' | 'fabric' | 'craft'
const title: Record<SectionName, string> = { story: 'Câu chuyện tác phẩm', fabric: 'Chất liệu & bảo quản', craft: 'Quy trình chế tác' }
export default function InfoTabs({ product }: { product: CatalogProduct }) {
  const sections = (['story', 'fabric', 'craft'] as const).filter((key) => {
    const value = product[key]
    return !!(value && (value.title || value.description || value.paragraphs?.length || value.care?.length || value.steps?.length))
  })
  const [active, setActive] = useState<SectionName>(sections[0] || 'story')
  if (!sections.length) return null
  const current = sections.includes(active) ? active : sections[0]
  const content: CatalogContent | undefined = product[current]
  return <section className="w-full bg-surface-container-low py-16">
    <div className="max-w-[1440px] mx-auto px-4 sm:px-8 lg:px-margin-desktop">
      <div role="tablist" aria-label="Thông tin sản phẩm" className="flex flex-wrap justify-center gap-3 mb-8">
        {sections.map((key) => <button key={key} role="tab" type="button" aria-selected={current === key} onClick={() => setActive(key)}
          className={`px-5 py-3 font-title-editorial ${current === key ? 'bg-primary text-on-primary' : 'bg-surface text-on-surface'}`}>{title[key]}</button>)}
      </div>
      {content && <div role="tabpanel" className="grid grid-cols-1 lg:grid-cols-12 gap-8 bg-surface-container p-6 sm:p-10">
        <div className={content.image?.src ? 'lg:col-span-7 space-y-4' : 'lg:col-span-12 space-y-4'}>
          {content.eyebrow && <p className="text-secondary uppercase tracking-widest text-sm">{content.eyebrow}</p>}
          {content.title && <h2 className="font-headline-md text-headline-md text-primary">{content.title}</h2>}
          {content.description && <p className="leading-relaxed text-on-surface-variant">{content.description}</p>}
          {content.paragraphs?.map((paragraph, i) => <p key={i} className="leading-relaxed text-on-surface-variant">{paragraph}</p>)}
          {(content.care || content.steps || []).map((item, i) => <div key={`${item.title}-${i}`} className="bg-surface p-4">
            <h3 className="font-semibold text-primary">{item.title}</h3><p className="mt-1 text-on-surface-variant">{item.description}</p>
          </div>)}
        </div>
        {content.image?.src && <div className="lg:col-span-5"><CatalogImage src={content.image.src} alt={content.image.alt || title[current]} className="w-full aspect-[4/5]" /></div>}
      </div>}
    </div>
  </section>
}
