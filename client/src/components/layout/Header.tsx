import { Link, NavLink } from 'react-router-dom'
import { NAV_ITEMS, ROUTES, SITE } from '@/data/site'
import Icon from '@/components/ui/Icon'

const iconButton = 'relative text-on-surface-variant hover:text-primary transition-colors p-1'

export default function Header() {
  return (
    <header className="fixed top-0 left-0 right-0 z-50 bg-surface/90 backdrop-blur-md shadow-[0_1px_8px_rgba(0,0,0,0.04)]">
      <div className="bg-primary text-on-primary py-2 px-margin text-center font-label-uppercase text-label-uppercase tracking-widest">
        <span>{SITE.announcement}</span>
      </div>
      <div className="h-20 max-w-[1440px] mx-auto px-margin-desktop flex items-center justify-between gap-6">
        <Link to={ROUTES.home} className="flex items-center gap-4 flex-shrink-0">
          <img alt={`${SITE.name} - Logo`} className="h-8 w-auto object-contain" src={SITE.logo} />
          <div className="flex flex-col">
            <span className="font-title-editorial text-title-editorial tracking-wide uppercase text-primary font-bold">{SITE.name}</span>
            <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest">{SITE.tagline}</span>
          </div>
        </Link>

        <nav className="hidden xl:flex items-center gap-6" aria-label="Điều hướng chính">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.label}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                `px-3 py-1.5 rounded font-label-regular text-label-regular tracking-wide transition-colors ${
                  isActive && !item.to.includes('?') ? 'bg-primary-container text-on-primary font-semibold' : 'text-on-surface-variant hover:text-primary'
                }`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="flex items-center gap-4">
          <div className="hidden md:flex items-center gap-1.5 text-on-surface-variant font-label-uppercase text-label-uppercase bg-surface-container px-2.5 py-1">
            <span className="text-primary font-bold">VN</span>
            <span className="text-outline">|</span>
            <span>VND</span>
          </div>
          <button aria-label="Tìm kiếm" className={iconButton} type="button">
            <Icon name="search" size={20} />
          </button>
          <button aria-label="Danh sách yêu thích" className={iconButton} type="button">
            <Icon name="favorite" size={20} />
            <span className="absolute -top-1 -right-1 bg-primary text-on-primary text-[10px] w-4 h-4 flex items-center justify-center rounded-full font-label-regular">
              0
            </span>
          </button>
          <button aria-label="Giỏ hàng" className={iconButton} type="button">
            <Icon name="shopping_bag" size={20} />
            <span className="absolute -top-1 -right-1 bg-secondary text-on-secondary text-[10px] w-4 h-4 flex items-center justify-center rounded-full font-label-regular">
              2
            </span>
          </button>
          <button aria-label="Tài khoản" className="flex items-center ml-1" type="button">
            <span className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
              <Icon name="person" size={18} className="text-on-primary" />
            </span>
          </button>
        </div>
      </div>
    </header>
  )
}
