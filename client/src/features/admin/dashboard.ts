// Types mirror GET /api/v1/admin/dashboard (server/internal/application/dashboard).
import { AdminApiError } from './catalog.ts'

export type Granularity = 'day' | 'week' | 'month'
export type Metric = { value: number; previous: number; change_pct: number | null }
export type Count = { key: string; count: number; revenue: number }
export type ProductStat = { product_id: string; name: string; slug?: string; image?: string; revenue: number; quantity: number }
export type CategoryStat = { category_id: string; name: string; revenue: number; quantity: number }
export type SizeStat = { size: string; revenue: number; quantity: number }
export type LowStockItem = { product_id: string; name: string; slug: string; sku: string; color: string; size: string; stock: number }
export type DashboardReport = {
  range: { from: string; to: string; prev_from: string; prev_to: string; granularity: Granularity; timezone: string }
  kpis: { revenue: Metric; orders: Metric; avg_order_value: Metric; new_customers: Metric; cancel_rate: Metric }
  series: { date: string; revenue: number; orders: number }[]
  orders_by_status: Count[]
  orders_by_payment_status: Count[]
  orders_by_payment_method: Count[]
  top_products_revenue: ProductStat[]
  top_products_quantity: ProductStat[]
  revenue_by_category: CategoryStat[]
  revenue_by_size: SizeStat[]
  inventory: {
    low_stock_threshold: number; active_products: number; hidden_products: number; stock_value: number
    out_of_stock_skus: number; low_stock_skus: number; low_stock: LowStockItem[]
  }
  tailoring_by_status: Count[]
}

const apiBase = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

export async function fetchDashboard(query: { from: string; to: string; granularity: Granularity }, signal?: AbortSignal): Promise<DashboardReport> {
  const params = new URLSearchParams(query)
  const response = await fetch(apiBase + '/api/v1/admin/dashboard?' + params.toString(), { credentials: 'include', headers: { Accept: 'application/json' }, signal })
  const body = (await response.json().catch(() => null)) as { data?: DashboardReport; error?: { message?: string; field?: string } } | null
  if (!response.ok) throw new AdminApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể tải số liệu thống kê')
  if (!body?.data || !Array.isArray(body.data.series) || !body.data.kpis) throw new AdminApiError(response.status, '', 'Dữ liệu thống kê không hợp lệ')
  return body.data
}

// ---- date presets (Vietnam calendar dates, no time zone maths needed: server resolves the day boundaries) ----
export type Preset = '7d' | '30d' | '90d' | 'ytd' | 'custom'

const pad = (n: number) => String(n).padStart(2, '0')
export const isoDate = (d: Date) => d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate())

/** Today's date in Vietnam (UTC+7), independent of the browser time zone. */
export function vietnamToday(now: Date = new Date()): Date {
  const shifted = new Date(now.getTime() + 7 * 3600 * 1000)
  return new Date(shifted.getUTCFullYear(), shifted.getUTCMonth(), shifted.getUTCDate())
}

export function presetRange(preset: Exclude<Preset, 'custom'>, now: Date = new Date()): { from: string; to: string } {
  const today = vietnamToday(now)
  const back = (days: number) => new Date(today.getFullYear(), today.getMonth(), today.getDate() - (days - 1))
  switch (preset) {
    case '7d': return { from: isoDate(back(7)), to: isoDate(today) }
    case '90d': return { from: isoDate(back(90)), to: isoDate(today) }
    case 'ytd': return { from: today.getFullYear() + '-01-01', to: isoDate(today) }
    default: return { from: isoDate(back(30)), to: isoDate(today) }
  }
}

export function defaultGranularity(preset: Preset): Granularity {
  return preset === '90d' ? 'week' : preset === 'ytd' ? 'month' : 'day'
}

/** Whole days in [from, to] inclusive; NaN for invalid dates. */
export function rangeDays(from: string, to: string): number {
  const a = Date.parse(from + 'T00:00:00Z')
  const b = Date.parse(to + 'T00:00:00Z')
  return Math.round((b - a) / 86400000) + 1
}

// ---- number formatting ----
const trimZero = (n: number) => n.toFixed(1).replace(/\.0$/, '').replace('.', ',')

/** 1.234.567.890 -> "1,2 tỷ", 2.500.000 -> "2,5 tr", 850.000 -> "850k", 950 -> "950". */
export function formatVNDShort(value: number): string {
  const sign = value < 0 ? '-' : ''
  const n = Math.abs(value)
  if (n >= 1e9) return sign + trimZero(n / 1e9) + ' tỷ'
  if (n >= 1e6) return sign + trimZero(n / 1e6) + ' tr'
  if (n >= 1e3) return sign + trimZero(n / 1e3) + 'k'
  return sign + String(Math.round(n))
}

/** Full VND with dot thousands separators: 1500000 -> "1.500.000 ₫". */
export function formatVND(value: number): string {
  return Math.round(value).toString().replace(/\B(?=(\d{3})+(?!\d))/g, '.') + ' ₫'
}

/** "+12,3%", "-4,0%", or "—" when there is no baseline. */
export function formatChange(pct: number | null | undefined): string {
  if (pct === null || pct === undefined || !Number.isFinite(pct)) return '—'
  const rounded = Math.round(pct * 10) / 10
  return (rounded > 0 ? '+' : '') + rounded.toFixed(1).replace('.', ',') + '%'
}

export function formatPercent(rate: number): string {
  return (Math.round(rate * 1000) / 10).toFixed(1).replace('.', ',') + '%'
}

// ---- chart scales ----
/** Rounds a maximum up to a "nice" axis maximum and returns evenly spaced ticks from 0. */
export function niceScale(max: number, tickCount = 4): { max: number; ticks: number[] } {
  if (!Number.isFinite(max) || max <= 0) return { max: tickCount, ticks: Array.from({ length: tickCount + 1 }, (_, i) => i) }
  const rough = max / tickCount
  const magnitude = Math.pow(10, Math.floor(Math.log10(rough)))
  const residual = rough / magnitude
  const nice = residual <= 1 ? 1 : residual <= 2 ? 2 : residual <= 2.5 ? 2.5 : residual <= 5 ? 5 : 10
  const step = nice * magnitude
  const niceMax = step * Math.ceil(max / step - 1e-9)
  const ticks: number[] = []
  for (let v = 0; v <= niceMax + step / 2; v += step) ticks.push(Math.round(v * 1e6) / 1e6)
  return { max: niceMax, ticks }
}

/** Linear map from [d0,d1] to [r0,r1]. */
export function scaleLinear(d0: number, d1: number, r0: number, r1: number): (v: number) => number {
  const span = d1 - d0 || 1
  return (v) => r0 + ((v - d0) / span) * (r1 - r0)
}

/** "2026-03-02" -> "02/03" (day/week buckets) or "03/2026" (month buckets). */
export function formatBucketLabel(date: string, granularity: Granularity): string {
  const [y, m, d] = date.split('-')
  return granularity === 'month' ? m + '/' + y : d + '/' + m
}

const STATUS_LABELS: Record<string, string> = {
  pending: 'Chờ xác nhận', confirmed: 'Đã xác nhận', shipping: 'Đang giao', delivered: 'Đã giao', cancelled: 'Đã hủy',
  unpaid: 'Chưa thanh toán', paid: 'Đã thanh toán', refunded: 'Đã hoàn tiền',
  submitted: 'Mới gửi', contacted: 'Đã liên hệ', consultation_scheduled: 'Đã hẹn tư vấn', measurement_confirmed: 'Đã chốt số đo', in_progress: 'Đang may', completed: 'Hoàn thành',
  cod: 'Giao tận nơi (COD)', cash_at_counter: 'Nhận tại quầy',
}
export const statusLabel = (key: string) => STATUS_LABELS[key] ?? (key || 'Khác')

/** Colour palette (max 6 categorical colours) consistent with the site theme. */
export const CHART_COLORS = ['#5b0215', '#775a19', '#2f6f73', '#4a5d8a', '#7a8a3a', '#c0577a'] as const
export const chartColor = (index: number) => CHART_COLORS[index % CHART_COLORS.length]
