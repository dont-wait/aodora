import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import Icon from '@/components/ui/Icon'

export type ToastKind = 'success' | 'error' | 'info'

const toastIcon: Record<ToastKind, { name: string; className: string }> = {
  success: { name: 'check_circle', className: 'text-secondary-fixed' },
  error: { name: 'error', className: 'text-red-300' },
  info: { name: 'info', className: 'text-secondary-fixed' },
}

interface ToastContextValue {
  showToast: (message: string, kind?: ToastKind) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<{ message: string; kind: ToastKind } | null>(null)
  const timer = useRef<number | undefined>(undefined)

  const showToast = useCallback((message: string, kind: ToastKind = 'success') => {
    setToast({ message, kind })
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(() => setToast(null), 3200)
  }, [])

  useEffect(() => () => window.clearTimeout(timer.current), [])

  const value = useMemo(() => ({ showToast }), [showToast])

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div
        role={toast?.kind === 'error' ? 'alert' : 'status'}
        aria-live={toast?.kind === 'error' ? 'assertive' : 'polite'}
        className={`fixed bottom-6 right-6 z-50 bg-inverse-surface text-inverse-on-surface px-5 py-3.5 shadow-xl flex items-center gap-3 transition-opacity duration-300 ${
          toast ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
      >
        <Icon name={toastIcon[toast?.kind ?? 'success'].name} size={20} className={toastIcon[toast?.kind ?? 'success'].className} />
        <span className="font-label-regular text-label-regular">{toast?.message}</span>
      </div>
    </ToastContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast phải được dùng bên trong <ToastProvider>')
  return ctx
}
