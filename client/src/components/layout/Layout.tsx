import { useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { ToastProvider } from '@/hooks/useToast'
import Footer from './Footer'
import Header from './Header'

export default function Layout() {
  const { pathname } = useLocation()
  useEffect(() => window.scrollTo(0, 0), [pathname])

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
