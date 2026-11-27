import { useCallback, useEffect, useRef, useState, type MouseEvent } from 'react'
import { Link, NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import { useAuth } from '@/features/auth/useAuth'
import { ToastProvider, useToast } from '@/hooks/useToast'
import { Button } from './ui'
import { UNSAVED_MESSAGE, UnsavedContext } from './unsaved'

const NAV = [
  { to: '/admin', label: 'Tổng quan', icon: 'dashboard', end: true },
  { to: '/admin/catalog', label: 'Sản phẩm', icon: 'checkroom' },
  { to: '/admin/categories', label: 'Danh mục', icon: 'category' },
  { to: '/admin/orders', label: 'Đơn hàng', icon: 'receipt_long' },
  { to: '/admin/tailoring', label: 'May đo', icon: 'straighten' },
  { to: '/admin/reviews', label: 'Đánh giá', icon: 'rate_review' },
  { to: '/admin/import', label: 'Nhập dữ liệu', icon: 'upload_file' },
] as const

const linkClass = ({ isActive }: { isActive: boolean }) =>
  `flex items-center gap-3 px-4 py-3 text-sm transition-colors ${isActive ? 'bg-primary text-on-primary font-semibold' : 'text-on-surface-variant hover:bg-surface-container'}`

function NavList({ onNavigate, guard }: { onNavigate?: () => void; guard: (event: MouseEvent) => void }) {
  return (
    <nav aria-label="Điều hướng quản trị" className="flex flex-col">
      {NAV.map((item) => (
        <NavLink key={item.to} to={item.to} end={'end' in item} className={linkClass} onClick={(event) => { guard(event); if (!event.defaultPrevented) onNavigate?.() }}>
          <Icon name={item.icon} size={20} />
          {item.label}
        </NavLink>
      ))}
      <Link to="/" onClick={guard} className="flex items-center gap-3 px-4 py-3 text-sm text-on-surface-variant hover:bg-surface-container border-t border-outline-variant/40 mt-2">
        <Icon name="storefront" size={20} />
        Về trang khách
      </Link>
    </nav>
  )
}

function Shell() {
  const { user, logout } = useAuth()
  const { showToast } = useToast()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const [open, setOpen] = useState(false)
  const dirtyRef = useRef(false)
  const [dirty, setDirtyState] = useState(false)
  const setDirty = useCallback((value: boolean) => { dirtyRef.current = value; setDirtyState(value) }, [])
  const guard = (event: MouseEvent) => { if (dirtyRef.current && !window.confirm(UNSAVED_MESSAGE)) event.preventDefault() }
  useEffect(() => {
    if (!dirty) return
    const warn = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = '' }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])
  const [seenPath, setSeenPath] = useState(pathname)
  if (seenPath !== pathname) {
    // Close the mobile menu on navigation (adjusted during render, not in an effect).
    setSeenPath(pathname)
    setOpen(false)
  }
  const signOut = () => { if (dirtyRef.current && !window.confirm(UNSAVED_MESSAGE)) return; void logout().then(() => navigate('/dang-nhap')).catch((error: unknown) => showToast(error instanceof Error ? error.message : 'Không thể đăng xuất.', 'error')) }

  return (
    <div className="admin-ui min-h-screen bg-surface-container-low lg:flex">
      <aside className="hidden lg:flex lg:flex-col w-60 shrink-0 bg-surface border-r border-outline-variant/60 sticky top-0 h-screen overflow-y-auto">
        <div className="px-5 py-6 border-b border-outline-variant/40">
          <p className="font-title-editorial text-xl uppercase text-primary font-bold">Aodora</p>
          <p className="text-xs uppercase tracking-wider text-secondary mt-1">Chủ cửa hàng</p>
        </div>
        <div className="py-3"><NavList guard={guard} /></div>
      </aside>
      <div className="flex-1 min-w-0 flex flex-col">
        <header className="sticky top-0 z-40 bg-surface border-b border-outline-variant/60">
          <div className="flex items-center justify-between gap-3 px-4 sm:px-6 h-14">
            <div className="flex items-center gap-2">
              <button
                type="button"
                className="lg:hidden p-1 text-primary"
                aria-label={open ? 'Đóng menu quản trị' : 'Mở menu quản trị'}
                aria-expanded={open}
                aria-controls="admin-mobile-nav"
                onClick={() => setOpen((value) => !value)}
              >
                <Icon name={open ? 'close' : 'menu'} size={24} />
              </button>
              <span className="lg:hidden font-title-editorial uppercase font-bold text-primary">Aodora</span>
            </div>
            <div className="flex items-center gap-3 text-sm">
              <span className="hidden sm:inline text-on-surface-variant truncate max-w-48" title={user?.email}>{user?.full_name}</span>
              <Button variant="secondary" onClick={signOut}>Đăng xuất</Button>
            </div>
          </div>
          {open && <div id="admin-mobile-nav" className="lg:hidden border-t border-outline-variant/40 bg-surface"><NavList guard={guard} onNavigate={() => setOpen(false)} /></div>}
        </header>
        <main className="flex-1 p-4 sm:p-6 lg:p-8">
          <div className="max-w-[1440px] mx-auto space-y-6">
            <UnsavedContext.Provider value={{ setDirty }}>
              <Outlet />
            </UnsavedContext.Provider>
          </div>
        </main>
      </div>
    </div>
  )
}

export default function AdminLayout() {
  return (
    <ToastProvider>
      <Shell />
    </ToastProvider>
  )
}
