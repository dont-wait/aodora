import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Link, type LinkProps } from 'react-router-dom'
import { buttonClassFor, labelClass, toneClasses, type ButtonVariant, type Tone } from './uiStyles'



export function Button({ variant = 'primary', className = '', type = 'button', ...rest }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant }) {
  return <button type={type} className={buttonClassFor(variant, className)} {...rest} />
}
export function ButtonLink({ variant = 'secondary', className = '', ...rest }: LinkProps & { variant?: ButtonVariant }) {
  return <Link className={buttonClassFor(variant, className)} {...rest} />
}

export function Card({ title, description, actions, children, className = '' }: { title?: string; description?: string; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`bg-surface border border-outline-variant/60 p-5 space-y-4 min-w-0 ${className}`}>
      {(title || actions) && (
        <header className="flex flex-wrap items-start justify-between gap-3">
          <div>
            {title && <h2 className="text-lg font-bold text-primary">{title}</h2>}
            {description && <p className="text-sm text-on-surface-variant mt-1">{description}</p>}
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
      {hint && <span className="text-xs font-normal text-on-surface-variant">{hint}</span>}
    </label>
  )
}

export function StatusBadge({ tone = 'neutral', children, className = '' }: { tone?: Tone; children: ReactNode; className?: string }) {
  return <span className={`inline-block whitespace-nowrap px-2.5 py-0.5 text-xs font-semibold ${toneClasses[tone]} ${className}`}>{children}</span>
}


export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="bg-surface border border-dashed border-outline-variant p-8 text-center space-y-3">
      <p className="font-semibold text-primary">{title}</p>
      {children}
    </div>
  )
}

export function Alert({ kind = 'error', children }: { kind?: 'error' | 'success'; children: ReactNode }) {
  return kind === 'error'
    ? <p role="alert" className="border border-red-200 bg-red-50 px-4 py-3 text-red-800 text-sm">{children}</p>
    : <p role="status" className="border border-green-200 bg-green-50 px-4 py-3 text-green-800 text-sm">{children}</p>
}

export function PageHeader({ title, subtitle, actions }: { title: string; subtitle?: string; actions?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 className="text-2xl sm:text-3xl font-title-editorial text-primary">{title}</h1>
        {subtitle && <p className="text-sm text-on-surface-variant mt-1">{subtitle}</p>}
      </div>
      {actions}
    </div>
  )
}

/** Horizontally scrollable table wrapper with shared header/cell styling. */
export function DataTable({ children, label }: { children: ReactNode; label: string }) {
  return (
    <div className="overflow-x-auto bg-surface border border-outline-variant/60">
      <table aria-label={label} className="w-full text-sm text-left">{children}</table>
    </div>
  )
}
