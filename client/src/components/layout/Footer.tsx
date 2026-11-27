import { FOOTER, SITE, ROUTES } from '@/data/site'
import { Link } from 'react-router-dom'
import { getCategories } from '@/features/catalog/api'
import { useCatalogRequest } from '@/features/catalog/useCatalogRequest'
import BrandMark from './BrandMark'

const linkItem = 'font-body-sm text-body-sm text-on-surface-variant hover:text-primary transition-colors cursor-pointer'
const colTitle = 'font-label-uppercase text-label-uppercase text-primary mb-4 tracking-widest'

export default function Footer() {
  const categories = useCatalogRequest(getCategories, 'footer:categories')

  return (
    <footer className="w-full bg-surface-container-high text-on-surface mt-20">
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop py-16">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-8 lg:gap-gutter-desktop">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <BrandMark compact />
              <span className="font-title-editorial text-title-editorial uppercase text-primary font-bold">{SITE.name}</span>
            </div>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-6 leading-relaxed">{FOOTER.about}</p>
            <span className="inline-block px-2.5 py-1 bg-surface text-secondary font-label-uppercase text-label-uppercase">{FOOTER.badge}</span>
          </div>

          <div>
            <h4 className={colTitle}>Danh mục</h4>
            <ul className="space-y-2.5">
              {categories.data?.map((category) => (
                <li key={category.id} className={linkItem}>
                  <Link to={`${ROUTES.collection}?category=${encodeURIComponent(category.slug || category.id)}`}>{category.name}</Link>
                </li>
              ))}
            </ul>
          </div>

          <div>
            <h4 className={colTitle}>Hỗ trợ Khách hàng</h4>
            <ul className="space-y-2.5">
              {FOOTER.support.map((c) => (
                <li key={c.label} className={linkItem}>
                  <Link to={c.to}>{c.label}</Link>
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
            <h4 className={colTitle}>Liên hệ</h4>
            <div className="space-y-3 font-body-sm text-body-sm text-on-surface-variant">
              <p>
                Hotline tư vấn: <a className="text-primary font-semibold" href={`tel:${SITE.hotline.replace(/\s/g, '')}`}>{SITE.hotline}</a>
              </p>
              <Link
                to={`${ROUTES.tailoring}#bang-so-do`}
                className="inline-block bg-primary hover:bg-primary-container text-on-primary font-label-uppercase text-label-uppercase px-4 py-2.5 transition-colors"
              >
                Đặt lịch may đo
              </Link>
            </div>
          </div>
        </div>

        <div className="mt-12 pt-8 flex flex-col md:flex-row items-center justify-between gap-4 font-body-sm text-body-sm text-on-surface-variant">
          <p>{FOOTER.copyright}</p>
          <div className="flex items-center gap-6 font-label-uppercase text-label-uppercase tracking-wider">
            {FOOTER.socials.filter((s) => s.href).map((s) => (
              <a key={s.label} className="hover:text-primary transition-colors" href={s.href} target="_blank" rel="noreferrer">
                {s.label}
              </a>
            ))}
          </div>
        </div>
      </div>
    </footer>
  )
}
