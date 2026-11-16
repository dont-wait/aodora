import { Link, NavLink, useNavigate } from 'react-router-dom'
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { NAV_ITEMS, ROUTES, SITE } from '@/data/site'
import Icon from '@/components/ui/Icon'
import BrandMark from './BrandMark'
import { useAuth } from '@/features/auth/useAuth'
import { useToast } from '@/hooks/useToast'
import { useWishlist } from '@/features/account/useWishlist'
import { useCart } from '@/features/cart/useCart'

const iconButton = 'relative text-on-surface-variant hover:text-primary transition-colors p-1'

export default function Header() {
  const navigate = useNavigate()
  const [showSearch, setShowSearch] = useState(false)
  const [search, setSearch] = useState('')
  const [accountOpen, setAccountOpen] = useState(false)
  const accountRef = useRef<HTMLDivElement>(null)
  const { user, logout } = useAuth()
  const { showToast } = useToast()
  const { items: wishlistItems } = useWishlist()
  const { itemCount: cartItemCount } = useCart()
  useEffect(() => {
    function closeAccount(event: PointerEvent) {
      if (!accountRef.current?.contains(event.target as Node)) setAccountOpen(false)
    }
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') setAccountOpen(false)
    }
    document.addEventListener('pointerdown', closeAccount)
    document.addEventListener('keydown', closeOnEscape)
    return () => {
      document.removeEventListener('pointerdown', closeAccount)
      document.removeEventListener('keydown', closeOnEscape)
    }
  }, [])
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
          <BrandMark />
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
          <button aria-label="Danh sách yêu thích" title="Danh sách yêu thích" className={`${iconButton} flex items-center gap-0.5`} type="button" onClick={() => navigate(user ? ROUTES.wishlist : `${ROUTES.login}?next=${encodeURIComponent(ROUTES.wishlist)}`)}><Icon name="favorite" size={20} />{user && wishlistItems.length > 0 && <span className="text-[10px] text-primary">{wishlistItems.length}</span>}</button>
          <button aria-label="Giỏ hàng" title="Giỏ hàng" className={`${iconButton} flex items-center gap-0.5`} type="button" onClick={() => navigate(ROUTES.cart)}><Icon name="shopping_bag" size={20} />{cartItemCount > 0 && <span className="text-[10px] text-primary">{cartItemCount}</span>}</button>
          {user ? <div ref={accountRef} className="relative flex items-center gap-2 ml-1">
            <span className="hidden lg:inline text-sm text-on-surface-variant max-w-28 truncate" title={user.full_name}>{user.full_name}</span>
            <button aria-label="Mở menu tài khoản" aria-expanded={accountOpen} aria-haspopup="menu" title="Tài khoản" className="flex items-center" type="button" onClick={() => setAccountOpen((value) => !value)}>
              <span className="w-8 h-8 rounded-full bg-primary flex items-center justify-center"><Icon name="person" size={18} className="text-on-primary" /></span>
            </button>
            {accountOpen && <div role="menu" aria-label="Menu tài khoản" className="absolute right-0 top-full mt-3 w-64 bg-surface border border-outline-variant shadow-lg p-2">
              <div className="border-b border-outline-variant px-3 py-3">
                <p className="font-semibold text-primary truncate">{user.full_name}</p>
                <p className="text-sm text-on-surface-variant truncate">{user.email}</p>
                <p className="text-xs uppercase tracking-widest text-secondary mt-1">{user.role === 'admin' ? 'Chủ cửa hàng' : 'Khách hàng'}</p>
              </div>
              <Link role="menuitem" to={ROUTES.account} className="flex items-center gap-2 px-3 py-3 text-on-surface-variant hover:bg-surface-container" onClick={() => setAccountOpen(false)}><Icon name="person" size={18} className="text-secondary" />Tài khoản của tôi</Link>
              <button role="menuitem" type="button" className="w-full text-left px-3 py-3 text-on-surface-variant hover:bg-surface-container" onClick={() => void logout().then(() => { setAccountOpen(false); navigate(ROUTES.login) }).catch((error: unknown) => showToast(error instanceof Error ? error.message : 'Không thể đổi tài khoản.'))}>Đổi tài khoản</button>
              <button role="menuitem" type="button" className="w-full text-left px-3 py-3 text-red-700 hover:bg-surface-container" onClick={() => void logout().then(() => setAccountOpen(false)).catch((error: unknown) => showToast(error instanceof Error ? error.message : 'Không thể đăng xuất.'))}>Đăng xuất</button>
            </div>}
          </div> : <button aria-label="Đăng nhập" title="Đăng nhập" className="flex items-center ml-1" type="button" onClick={() => navigate(ROUTES.login)}>
            <span className="w-8 h-8 rounded-full bg-primary flex items-center justify-center"><Icon name="person" size={18} className="text-on-primary" /></span>
          </button>}
        </div>
      </div>
      {showSearch && <form onSubmit={submitSearch} className="bg-surface-container-low px-margin-desktop py-3 flex gap-2" role="search">
        <input autoFocus aria-label="Từ khóa tìm sản phẩm" placeholder="Tìm sản phẩm theo từ khóa" value={search} onChange={(e) => setSearch(e.target.value)} className="flex-1 bg-surface p-2" />
        <button type="submit" className="bg-primary text-on-primary px-5">Tìm</button>
      </form>}
    </header>
  )
}
