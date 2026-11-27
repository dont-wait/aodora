export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger'
export type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

/** One corner radius (rounded-lg) for inputs, buttons and cards; spacing in multiples of 4px. */
export const inputClass = 'w-full h-10 rounded-lg border border-outline-variant bg-surface-container-lowest px-3 text-sm text-on-surface placeholder:text-outline focus:border-primary disabled:opacity-60'
export const textareaClass = 'w-full rounded-lg border border-outline-variant bg-surface-container-lowest px-3 py-2 text-sm leading-6 text-on-surface placeholder:text-outline focus:border-primary disabled:opacity-60'
export const labelClass = 'flex flex-col gap-1.5 text-[13px] font-medium text-on-surface-variant'
export const hintClass = 'text-xs font-normal text-on-surface-variant'

export const buttonVariants: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-on-primary hover:bg-primary-container border border-primary',
  secondary: 'bg-surface-container-lowest text-primary border border-primary hover:bg-surface-container',
  ghost: 'text-primary border border-transparent hover:bg-surface-container',
  danger: 'bg-surface-container-lowest text-red-700 border border-red-700 hover:bg-red-50',
}
const buttonBase = 'inline-flex h-10 items-center justify-center gap-2 rounded-lg px-4 text-sm font-medium whitespace-nowrap transition-colors disabled:opacity-50 disabled:cursor-not-allowed'
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

/** Table: 13px uppercase header, numbers right-aligned with tabular figures. */
export const thClass = 'sticky top-0 z-10 px-3 py-2.5 text-[12px] font-semibold uppercase tracking-wide text-on-surface-variant bg-surface-container-low whitespace-nowrap text-left'
export const tdClass = 'px-3 py-2.5 align-top border-t border-outline-variant/40'
export const thNumClass = `${thClass} text-right`
export const tdNumClass = `${tdClass} text-right tabular-nums whitespace-nowrap`

export const toneClasses: Record<Tone, string> = {
  neutral: 'bg-surface-container-high text-on-surface',
  success: 'bg-green-100 text-green-800',
  warning: 'bg-amber-100 text-amber-900',
  danger: 'bg-red-100 text-red-800',
  info: 'bg-blue-100 text-blue-800',
}
