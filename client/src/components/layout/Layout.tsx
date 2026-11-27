import { useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { ToastProvider } from '@/hooks/useToast'
import Footer from './Footer'
import Header from './Header'

export default function Layout() {
  const { pathname, hash } = useLocation()
  useEffect(() => {
    const target = hash ? document.getElementById(decodeURIComponent(hash.slice(1))) : null
    if (target) target.scrollIntoView()
    else window.scrollTo(0, 0)
  }, [pathname, hash])

  return (
    <ToastProvider>
      <Header />
      <main className="w-full pt-28 bg-background min-h-screen">
        <Outlet />
      </main>
      <Footer />
    </ToastProvider>
  )
}
