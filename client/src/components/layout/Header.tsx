import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useState, type FormEvent } from 'react'
import { NAV_ITEMS, ROUTES, SITE } from '@/data/site'
import Icon from '@/components/ui/Icon'

const iconButton = 'relative text-on-surface-variant hover:text-primary transition-colors p-1'

export default function Header() {
  const navigate = useNavigate()
  const [showSearch, setShowSearch] = useState(false)
  const [search, setSearch] = useState('')
  function submitSearch(e: FormEvent) {
    e.preventDefault()
    navigate(`${ROUTES.collection}?search=${encodeURIComponent(search.trim())}`)
    setShowSearch(false)
  }
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
          <button aria-label="Tìm kiếm" aria-expanded={showSearch} onClick={() => setShowSearch((v) => !v)} className={iconButton} type="button">
            <Icon name="search" size={20} />
          </button>
          <button aria-label="Danh sách yêu thích (chưa tích hợp)" disabled title="Chức năng yêu thích đang phát triển" className={`${iconButton} opacity-40 cursor-not-allowed`} type="button"><Icon name="favorite" size={20} /></button>
          <button aria-label="Giỏ hàng (chưa tích hợp)" disabled title="Chức năng giỏ hàng đang phát triển" className={`${iconButton} opacity-40 cursor-not-allowed`} type="button"><Icon name="shopping_bag" size={20} /></button>
          <button aria-label="Tài khoản" className="flex items-center ml-1" type="button">
            <span className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
              <Icon name="person" size={18} className="text-on-primary" />
            </span>
          </button>
        </div>
      </div>
      {showSearch && <form onSubmit={submitSearch} className="bg-surface-container-low px-margin-desktop py-3 flex gap-2" role="search">
        <input autoFocus aria-label="Từ khóa tìm sản phẩm" placeholder="Tìm sản phẩm theo từ khóa" value={search} onChange={(e) => setSearch(e.target.value)} className="flex-1 bg-surface p-2" />
        <button type="submit" className="bg-primary text-on-primary px-5">Tìm</button>
      </form>}
    </header>
  )
}
