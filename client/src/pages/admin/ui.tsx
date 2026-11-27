import { useLayoutEffect, useRef, type ButtonHTMLAttributes, type ReactNode, type TextareaHTMLAttributes } from 'react'
import { Link, type LinkProps } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import { buttonClassFor, hintClass, labelClass, textareaClass, toneClasses, type ButtonVariant, type Tone } from './uiStyles'

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant; loading?: boolean }
export function Button({ variant = 'primary', className = '', type = 'button', loading = false, disabled, children, ...rest }: ButtonProps) {
  return (
    <button type={type} className={buttonClassFor(variant, className)} disabled={disabled || loading} aria-busy={loading || undefined} {...rest}>
      {loading && <Icon name="progress_activity" size={16} className="animate-spin" />}
      {children}
    </button>
  )
}
export function ButtonLink({ variant = 'secondary', className = '', ...rest }: LinkProps & { variant?: ButtonVariant }) {
  return <Link className={buttonClassFor(variant, className)} {...rest} />
}

export function Card({ title, description, actions, children, className = '' }: { title?: string; description?: string; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`min-w-0 space-y-4 rounded-lg border border-outline-variant/50 bg-surface-container-lowest p-5 shadow-[0_1px_2px_rgba(31,27,24,0.04)] sm:p-6 ${className}`}>
      {(title || actions) && (
        <header className="flex flex-wrap items-start justify-between gap-3">
          <div>
            {title && <h2 className="text-base font-semibold text-primary">{title}</h2>}
            {description && <p className="mt-1 text-[13px] text-on-surface-variant">{description}</p>}
          </div>
          {actions}
        </header>
      )}
      {children}
    </section>
  )
}

export function Field({ label, hint, children, className = '' }: { label: string; hint?: string; children: ReactNode; className?: string }) {
  return (
    <label className={`${labelClass} ${className}`}>
      <span>{label}</span>
      {children}
      {hint && <span className={hintClass}>{hint}</span>}
    </label>
  )
}

/** Textarea that grows with its content (no inner scrollbar). */
export function AutoTextarea({ className = '', value, minRows = 3, ...rest }: TextareaHTMLAttributes<HTMLTextAreaElement> & { minRows?: number }) {
  const ref = useRef<HTMLTextAreaElement>(null)
  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = el.scrollHeight + 2 + 'px'
  }, [value])
  return <textarea ref={ref} rows={minRows} value={value} className={`${textareaClass} resize-none overflow-hidden ${className}`} {...rest} />
}

export function StatusBadge({ tone = 'neutral', children, className = '' }: { tone?: Tone; children: ReactNode; className?: string }) {
  return <span className={`inline-block whitespace-nowrap rounded-md px-2.5 py-0.5 text-xs font-semibold ${toneClasses[tone]} ${className}`}>{children}</span>
}

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="space-y-3 rounded-lg border border-dashed border-outline-variant bg-surface-container-lowest p-8 text-center">
      <p className="font-semibold text-primary">{title}</p>
      {children}
    </div>
  )
}

export function Alert({ kind = 'error', children, action }: { kind?: 'error' | 'success'; children: ReactNode; action?: ReactNode }) {
  const body = (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <span>{children}</span>
      {action}
    </div>
  )
  return kind === 'error'
    ? <div role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">{body}</div>
    : <div role="status" className="rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800">{body}</div>
}

export function PageHeader({ title, subtitle, actions }: { title: string; subtitle?: string; actions?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="text-primary">{title}</h1>
        {subtitle && <p className="mt-1 text-[13px] text-on-surface-variant">{subtitle}</p>}
      </div>
      {actions}
    </div>
  )
}

/** Scrollable table wrapper: sticky header, zebra rows, hover highlight. */
export function DataTable({ children, label }: { children: ReactNode; label: string }) {
  return (
    <div className="max-h-[75vh] overflow-auto rounded-lg border border-outline-variant/50 bg-surface-container-lowest">
      <table aria-label={label} className="w-full text-left text-sm [&_tbody_tr]:transition-colors [&_tbody_tr:hover]:bg-surface-container-low [&_tbody_tr:nth-child(even)]:bg-surface-container-low/40">
        {children}
      </table>
    </div>
  )
}

export function Skeleton({ className = '' }: { className?: string }) {
  return <div aria-hidden="true" className={`animate-pulse rounded-md bg-surface-container-high ${className}`} />
}
export function TableSkeleton({ rows = 6, label = 'Đang tải dữ liệu' }: { rows?: number; label?: string }) {
  return (
    <div role="status" aria-label={label} className="space-y-3 rounded-lg border border-outline-variant/50 bg-surface-container-lowest p-4">
      <Skeleton className="h-8 w-full" />
      {Array.from({ length: rows }, (_, i) => <Skeleton key={i} className="h-10 w-full" />)}
    </div>
  )
}
export function CardSkeleton({ lines = 4, label = 'Đang tải dữ liệu' }: { lines?: number; label?: string }) {
  return (
    <div role="status" aria-label={label} className="space-y-3 rounded-lg border border-outline-variant/50 bg-surface-container-lowest p-5">
      <Skeleton className="h-5 w-1/3" />
      {Array.from({ length: lines }, (_, i) => <Skeleton key={i} className={`h-4 ${i % 2 ? 'w-3/4' : 'w-full'}`} />)}
    </div>
  )
}
