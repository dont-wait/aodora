import { useState } from 'react'
import Icon from '@/components/ui/Icon'
import RichText from '@/components/ui/RichText'
import { faqs } from '@/data/guide'

export default function FaqSection() {
  const [openId, setOpenId] = useState<number | null>(null)

  return (
    <section className="w-full py-20 px-margin-tablet lg:px-margin-desktop bg-surface-container-lowest">
      <div className="max-w-[1000px] mx-auto">
        <div className="text-center mb-14">
          <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Giải đáp thắc mắc</span>
          <h2 className="font-headline-lg text-headline-lg text-primary mt-2">Những Câu Hỏi Thường Gặp Về May Đo Áo Dài</h2>
        </div>
        <div className="space-y-4">
          {faqs.map((f) => {
            const open = openId === f.id
            return (
              <div key={f.id} className="bg-surface-container shadow-sm">
                <button
                  type="button"
                  aria-expanded={open}
                  aria-controls={`faq-${f.id}`}
                  onClick={() => setOpenId(open ? null : f.id)}
                  className="w-full p-6 flex items-center justify-between gap-4 text-left"
                >
                  <h3 className="font-headline-sm text-title-editorial text-on-surface font-medium">{f.question}</h3>
                  <Icon name={open ? 'keyboard_arrow_up' : 'keyboard_arrow_down'} className="text-primary" />
                </button>
                {open && (
                  <div id={`faq-${f.id}`} className="px-6 pb-6 font-body-md text-body-md text-on-surface-variant leading-relaxed">
                    <RichText text={f.answer} />
                    {f.bullets && (
                      <ul className="list-disc pl-5 mt-2 space-y-1">
                        {f.bullets.map((b) => (
                          <li key={b}>
                            <RichText text={b} />
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}
