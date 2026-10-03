import { useState, type FormEvent } from 'react'
import { FOOTER, SITE } from '@/data/site'
import { useToast } from '@/hooks/useToast'

const linkItem = 'font-body-sm text-body-sm text-on-surface-variant hover:text-primary transition-colors cursor-pointer'
const colTitle = 'font-label-uppercase text-label-uppercase text-primary mb-4 tracking-widest'

export default function Footer() {
  const { showToast } = useToast()
  const [email, setEmail] = useState('')

  const subscribe = (e: FormEvent) => {
    e.preventDefault()
    // TODO: gọi API đăng ký bản tin
    showToast('Cảm ơn quý khách đã đăng ký nhận tin!')
    setEmail('')
  }

  return (
    <footer className="w-full bg-surface-container-high text-on-surface mt-20">
      <div className="max-w-[1440px] mx-auto px-margin-desktop py-16">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-8 lg:gap-gutter-desktop">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <img alt={SITE.name} className="h-7 w-auto object-contain" src={SITE.logo} />
              <span className="font-title-editorial text-title-editorial uppercase text-primary font-bold">{SITE.name}</span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-6 leading-relaxed">{FOOTER.about}</p>
            <span className="inline-block px-2.5 py-1 bg-surface text-secondary font-label-uppercase text-label-uppercase">{FOOTER.badge}</span>
          </div>

          <div>
            <h4 className={colTitle}>Danh mục</h4>
            <ul className="space-y-2.5">
              {FOOTER.categories.map((c) => (
                <li key={c} className={linkItem}>
                  {c}
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className={colTitle}>Hỗ trợ Khách hàng</h4>
            <ul className="space-y-2.5">
              {FOOTER.support.map((c) => (
                <li key={c} className={linkItem}>
                  {c}
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className={colTitle}>Xưởng & Showroom</h4>
            <div className="space-y-3 font-body-sm text-body-sm text-on-surface-variant">
              {FOOTER.showrooms.map((s) => (
                <p key={s.label}>
                  <strong className="text-on-surface font-label-regular">{s.label}:</strong> {s.value}
                </p>
              ))}
            </div>
          </div>

          <div>
            <h4 className={colTitle}>Bản tin Thưởng lãm</h4>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-4">
              Đăng ký nhận thông tin các bộ sưu tập giới hạn và nhận ngay voucher may đo 500.000đ.
            </p>
            <form className="space-y-2" onSubmit={subscribe}>
              <input
                required
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full bg-surface text-on-surface px-3 py-2 text-body-sm font-body-sm focus:outline-none focus:bg-surface-bright"
                placeholder="Địa chỉ email của quý khách..."
              />
              <button
                className="w-full bg-primary hover:bg-primary-container text-on-primary font-label-uppercase text-label-uppercase py-2.5 transition-colors"
                type="submit"
              >
                Đăng ký nhận tin
              </button>
            </form>
          </div>
        </div>

        <div className="mt-12 pt-8 flex flex-col md:flex-row items-center justify-between gap-4 font-body-sm text-body-sm text-on-surface-variant">
          <p>{FOOTER.copyright}</p>
          <div className="flex items-center gap-6 font-label-uppercase text-label-uppercase tracking-wider">
            {FOOTER.socials.map((s) => (
              <a key={s} className="hover:text-primary transition-colors" href="#">
                {s}
              </a>
            ))}
          </div>
        </div>
      </div>
    </footer>
  )
}
