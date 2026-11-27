import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom'
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
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const { pathname, search: locationSearch } = useLocation()
  const accountRef = useRef<HTMLDivElement>(null)
  const { user, logout } = useAuth()
  const { showToast } = useToast()
  const { items: wishlistItems } = useWishlist()
  const { itemCount: cartItemCount } = useCart()
  const locationKey = pathname + locationSearch
  const [seenLocation, setSeenLocation] = useState(locationKey)
  if (seenLocation !== locationKey) {
    // Close menus when the route changes (state adjusted during render, not in an effect).
    setSeenLocation(locationKey)
    setMenuOpen(false)
    setAccountOpen(false)
  }
  useEffect(() => {
    function closeAccount(event: PointerEvent) {
      if (!accountRef.current?.contains(event.target as Node)) setAccountOpen(false)
      if (!menuRef.current?.contains(event.target as Node)) setMenuOpen(false)
    }
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') { setAccountOpen(false); setMenuOpen(false) }
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
      <div className="bg-primary text-on-primary py-2 px-margin md:px-margin-tablet lg:px-margin-desktop text-center font-label-uppercase text-label-uppercase tracking-widest">
        <span>{SITE.announcement}</span>
      </div>
      <div className="h-20 max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop flex items-center justify-between gap-3 sm:gap-6">
        <Link to={ROUTES.home} className="flex items-center gap-3 sm:gap-4 flex-shrink-0 min-w-0">
          <BrandMark responsive />
          <div className="flex flex-col">
            <span className="font-title-editorial text-title-editorial tracking-wide uppercase text-primary font-bold">{SITE.name}</span>
            <span className="hidden sm:block font-label-uppercase text-label-uppercase text-secondary tracking-widest">{SITE.tagline}</span>
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

        <div className="flex items-center gap-2 sm:gap-4">
          <div ref={menuRef} className="xl:hidden">
            <button aria-label="Mở menu điều hướng" aria-expanded={menuOpen} aria-controls="mobile-nav" className={iconButton} type="button" onClick={() => setMenuOpen((v) => !v)}>
              <Icon name={menuOpen ? 'close' : 'menu'} size={22} />
            </button>
            {menuOpen && <nav id="mobile-nav" aria-label="Menu điều hướng" className="absolute left-0 right-0 top-full bg-surface border-t border-outline-variant shadow-lg max-h-[calc(100vh-7rem)] overflow-y-auto">
              <ul className="px-margin md:px-margin-tablet py-2">
                {NAV_ITEMS.map((item) => <li key={item.label}>
                  <Link to={item.to} className="block py-3 font-label-regular text-label-regular text-on-surface hover:text-primary border-b border-outline-variant/40" onClick={() => setMenuOpen(false)}>{item.label}</Link>
                </li>)}
                <li><Link to={user ? ROUTES.account : ROUTES.login} className="block py-3 font-label-regular text-label-regular text-primary" onClick={() => setMenuOpen(false)}>{user ? 'Tài khoản của tôi' : 'Đăng nhập'}</Link></li>
              </ul>
            </nav>}
          </div>
          <button aria-label="Tìm kiếm" aria-expanded={showSearch} onClick={() => setShowSearch((v) => !v)} className={iconButton} type="button">
            <Icon name="search" size={20} />
          </button>
          <button aria-label="Danh sách yêu thích" title="Danh sách yêu thích" className={iconButton} type="button" onClick={() => navigate(user ? ROUTES.wishlist : `${ROUTES.login}?next=${encodeURIComponent(ROUTES.wishlist)}`)}><Icon name="favorite" size={20} />{user && wishlistItems.length > 0 && <span className="absolute -top-1 -right-1.5 min-w-4 h-4 px-1 rounded-full bg-primary text-on-primary text-[10px] leading-4 text-center" aria-label={`${wishlistItems.length} mục`}>{wishlistItems.length > 9 ? '9+' : wishlistItems.length}</span>}</button>
          <button aria-label="Giỏ hàng" title="Giỏ hàng" className={iconButton} type="button" onClick={() => navigate(ROUTES.cart)}><Icon name="shopping_bag" size={20} />{cartItemCount > 0 && <span className="absolute -top-1 -right-1.5 min-w-4 h-4 px-1 rounded-full bg-primary text-on-primary text-[10px] leading-4 text-center" aria-label={`${cartItemCount} mục`}>{cartItemCount > 9 ? '9+' : cartItemCount}</span>}</button>
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
              {user.role === 'admin' && <Link role="menuitem" to="/admin" className="flex items-center gap-2 px-3 py-3 text-on-surface-variant hover:bg-surface-container" onClick={() => setAccountOpen(false)}><Icon name="dashboard" size={18} className="text-secondary" />Trang quản trị</Link>}
              <button role="menuitem" type="button" className="w-full text-left px-3 py-3 text-red-700 hover:bg-surface-container" onClick={() => void logout().then(() => setAccountOpen(false)).catch((error: unknown) => showToast(error instanceof Error ? error.message : 'Không thể đăng xuất.', 'error'))}>Đăng xuất</button>
            </div>}
          </div> : <button aria-label="Đăng nhập" title="Đăng nhập" className="flex items-center ml-1" type="button" onClick={() => navigate(ROUTES.login)}>
            <span className="w-8 h-8 rounded-full bg-primary flex items-center justify-center"><Icon name="person" size={18} className="text-on-primary" /></span>
          </button>}
        </div>
      </div>
      {showSearch && <form onSubmit={submitSearch} className="bg-surface-container-low px-margin md:px-margin-tablet lg:px-margin-desktop py-3 flex gap-2" role="search">
        <input autoFocus aria-label="Từ khóa tìm sản phẩm" placeholder="Tìm sản phẩm theo từ khóa" value={search} onChange={(e) => setSearch(e.target.value)} className="flex-1 bg-surface p-2" />
        <button type="submit" className="bg-primary text-on-primary px-5">Tìm</button>
      </form>}
    </header>
  )
}
