import { Children, cloneElement, isValidElement, useLayoutEffect, useRef, useState, type ButtonHTMLAttributes, type ReactElement, type ReactNode, type TextareaHTMLAttributes } from 'react'
import { Link, type LinkProps } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import { fieldId, formatThousands, parseThousands } from '@/features/admin/validation'
import { buttonClassFor, hintClass, inputClass, labelClass, textareaClass, toneClasses, type ButtonVariant, type Tone } from './uiStyles'

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

/**
 * Label + control + inline error. With `path`, the control gets id/aria-invalid/aria-describedby
 * (id = fieldId(path)) so the page can focus the first invalid control and screen readers read the error.
 */
export function Field({ label, hint, children, className = '', path, error, required }: { label: string; hint?: string; children: ReactNode; className?: string; path?: string; error?: string; required?: boolean }) {
  const id = path ? fieldId(path) : undefined
  const child = Children.count(children) === 1 && isValidElement(children) && id
    ? cloneElement(children as ReactElement<Record<string, unknown>>, { id, 'aria-invalid': error ? true : undefined, 'aria-describedby': [error ? id + '-error' : '', hint ? id + '-hint' : ''].filter(Boolean).join(' ') || undefined })
    : children
  return (
    <div className={`${labelClass} ${className}`}>
      <label htmlFor={id}>{label}{required ? <span aria-hidden="true" className="text-red-700"> *</span> : null}</label>
      {child}
      {hint && <span id={id ? id + '-hint' : undefined} className={hintClass}>{hint}</span>}
      {error && <p id={id ? id + '-error' : undefined} role="alert" className="text-xs font-normal text-red-700">{error}</p>}
    </div>
  )
}

/** Whole-VND input shown with thousands separators (650.000); onChange gets an integer or null when empty. */
export function MoneyInput({ value, onChange, className = '', ...rest }: { value: number | null; onChange: (value: number | null) => void; className?: string } & Omit<React.InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'>) {
  const [draft, setDraft] = useState<string | null>(null)
  return (
    <div className="relative">
      <input
        {...rest}
        type="text"
        inputMode="numeric"
        autoComplete="off"
        className={`${inputClass} pr-10 tabular-nums ${className}`}
        value={draft ?? formatThousands(value)}
        onChange={(event) => {
          const parsed = parseThousands(event.target.value)
          setDraft(event.target.value === '' || Number.isNaN(parsed) ? event.target.value.replace(/[^\d.]/g, '') : formatThousands(parsed))
          if (!Number.isNaN(parsed)) onChange(parsed)
        }}
        onBlur={() => setDraft(null)}
      />
      <span aria-hidden="true" className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs text-on-surface-variant">₫</span>
    </div>
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

/** "Có N lỗi cần sửa" with a link to each invalid field. */
export function ErrorSummary({ errors, labels = {} }: { errors: Record<string, string>; labels?: Record<string, string> }) {
  const entries = Object.entries(errors).filter(([, message]) => message)
  if (entries.length === 0) return null
  const jump = (path: string) => {
    const el = document.getElementById(fieldId(path))
    if (el) { el.scrollIntoView({ block: 'center', behavior: 'smooth' }); (el as HTMLElement).focus({ preventScroll: true }) }
  }
  return (
    <div role="alert" className="space-y-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
      <p className="font-semibold">Có {entries.length} lỗi cần sửa trước khi lưu</p>
      <ul className="list-disc space-y-0.5 pl-5">
        {entries.slice(0, 8).map(([path, message]) => (
          <li key={path}><button type="button" className="text-left underline" onClick={() => jump(path)}>{labels[path] ? labels[path] + ': ' : ''}{message}</button></li>
        ))}
        {entries.length > 8 && <li>… và {entries.length - 8} lỗi khác</li>}
      </ul>
    </div>
  )
}
