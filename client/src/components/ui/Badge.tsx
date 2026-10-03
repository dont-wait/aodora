import type { ReactNode } from 'react'

export type BadgeVariant = 'primary' | 'secondary' | 'surface' | 'tertiary' | 'primaryContainer' | 'secondaryContainer' | 'secondaryFixed'

const VARIANTS: Record<BadgeVariant, string> = {
  primary: 'bg-primary text-on-primary',
  secondary: 'bg-secondary text-on-secondary',
  surface: 'bg-surface-container-highest text-on-surface',
  tertiary: 'bg-tertiary text-on-tertiary',
  primaryContainer: 'bg-primary-container text-on-primary',
  secondaryContainer: 'bg-secondary-container text-on-secondary-container',
  secondaryFixed: 'bg-secondary-fixed text-on-secondary-fixed',
}

interface BadgeProps {
  variant?: BadgeVariant
  className?: string
  children: ReactNode
}

export default function Badge({ variant = 'primary', className = '', children }: BadgeProps) {
  return <span className={`font-label-uppercase text-label-uppercase ${VARIANTS[variant]} ${className}`}>{children}</span>
}
