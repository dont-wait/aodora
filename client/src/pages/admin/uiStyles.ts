export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

export const inputClass = 'w-full border border-outline-variant bg-surface p-2.5 text-on-surface focus:outline-primary disabled:opacity-60'

export const labelClass = 'flex flex-col gap-1.5 text-sm font-medium text-on-surface'

export const buttonVariants: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-on-primary hover:bg-primary-container border border-primary',
  secondary: 'bg-surface text-primary border border-primary hover:bg-surface-container',
  ghost: 'text-primary underline border border-transparent hover:bg-surface-container',
  danger: 'bg-surface text-red-700 border border-red-700 hover:bg-red-50',
}
const buttonBase = 'inline-flex items-center justify-center gap-2 px-4 py-2 text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed'

export const buttonClass = `${buttonBase} ${buttonVariants.primary}`

export const buttonClassFor = (variant: ButtonVariant = 'primary', extra = '') => `${buttonBase} ${buttonVariants[variant]} ${extra}`.trim()

const statusTones: Record<string, Tone> = {
  // orders
  pending: 'warning', confirmed: 'info', shipping: 'info', delivered: 'success', cancelled: 'danger',
  unpaid: 'warning', paid: 'success', refunded: 'neutral',
  // tailoring
  submitted: 'warning', contacted: 'info', consultation_scheduled: 'info', measurement_confirmed: 'info', in_progress: 'info', completed: 'success',
}
export const toneForStatus = (status: string): Tone => statusTones[status] ?? 'neutral'

export const thClass = 'px-3 py-2.5 font-semibold text-on-surface-variant bg-surface-container-low whitespace-nowrap'

export const tdClass = 'px-3 py-2.5 align-top border-t border-outline-variant/40'

export const toneClasses: Record<Tone, string> = {
  neutral: 'bg-surface-container-high text-on-surface',
  success: 'bg-green-100 text-green-800',
  warning: 'bg-amber-100 text-amber-900',
  danger: 'bg-red-100 text-red-800',
  info: 'bg-blue-100 text-blue-800',
}
