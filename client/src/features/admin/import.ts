// Client side of POST /api/v1/admin/import/{categories|products} and GET /api/v1/admin/export/{kind}.
import { AdminApiError } from './catalog.ts'

export type ImportKind = 'categories' | 'products'
export type ImportMode = 'dry_run' | 'commit'
export type ImportRowError = { row: number; field: string; message: string }
export type ImportItem = { row: number; slug: string; action: 'create' | 'update' | 'skip'; detail?: string }
export type ImportReport = {
  mode: ImportMode
  created: number
  updated: number
  skipped: number
  ignored_stock: number
  rows: number
  committed: boolean
  errors: ImportRowError[]
  notes: string[]
  items: ImportItem[]
}

export const IMPORT_MAX_BYTES = 5 * 1024 * 1024

const apiBase = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

const num = (v: unknown) => (typeof v === 'number' && Number.isFinite(v) ? v : 0)
const BOM = String.fromCharCode(0xfeff)

/** Normalises the server payload so the UI never sees null/undefined arrays. */
export function parseImportReport(payload: unknown): ImportReport {
  const d = (payload && typeof payload === 'object' ? payload : {}) as Record<string, unknown>
  const arr = <T,>(v: unknown, map: (x: Record<string, unknown>) => T): T[] =>
    Array.isArray(v) ? v.filter((x): x is Record<string, unknown> => !!x && typeof x === 'object').map(map) : []
  return {
    mode: d.mode === 'commit' ? 'commit' : 'dry_run',
    created: num(d.created), updated: num(d.updated), skipped: num(d.skipped), ignored_stock: num(d.ignored_stock), rows: num(d.rows),
    committed: d.committed === true,
    errors: arr(d.errors, (e) => ({ row: num(e.row), field: String(e.field ?? ''), message: String(e.message ?? '') })),
    notes: Array.isArray(d.notes) ? d.notes.map(String) : [],
    items: arr(d.items, (i) => ({ row: num(i.row), slug: String(i.slug ?? ''), action: i.action === 'update' ? 'update' : i.action === 'skip' ? 'skip' : 'create', detail: i.detail ? String(i.detail) : undefined })),
  }
}

/** A dry run with no errors and something to write can be committed. */
export function canCommit(report: ImportReport | null): boolean {
  return !!report && report.mode === 'dry_run' && report.errors.length === 0 && report.created + report.updated > 0
}

/** Rows that have at least one error, for highlighting. */
export function errorRows(report: ImportReport): Set<number> {
  return new Set(report.errors.map((e) => e.row))
}

export function commitLabel(report: ImportReport): string {
  return 'Nhập ' + report.rows + ' dòng'
}

export function fileProblem(file: { name: string; size: number }): string {
  if (!/\.(csv|json)$/i.test(file.name)) return 'Chỉ nhận file .csv (UTF-8) hoặc .json'
  if (file.size > IMPORT_MAX_BYTES) return 'File quá lớn (tối đa 5 MB)'
  if (file.size === 0) return 'File trống'
  return ''
}

export async function runImport(kind: ImportKind, file: File, mode: ImportMode): Promise<ImportReport> {
  const form = new FormData()
  form.append('file', file)
  const response = await fetch(apiBase + '/api/v1/admin/import/' + kind + '?mode=' + mode, { method: 'POST', credentials: 'include', headers: { Accept: 'application/json' }, body: form })
  const body = (await response.json().catch(() => null)) as { data?: unknown; error?: { message?: string; field?: string } } | null
  if (!response.ok) throw new AdminApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể nhập dữ liệu')
  if (!body?.data) throw new AdminApiError(response.status, '', 'Phản hồi không hợp lệ')
  return parseImportReport(body.data)
}

export async function downloadExport(kind: ImportKind): Promise<void> {
  const response = await fetch(apiBase + '/api/v1/admin/export/' + kind, { credentials: 'include' })
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { error?: { message?: string } } | null
    throw new AdminApiError(response.status, '', body?.error?.message || 'Không thể xuất dữ liệu')
  }
  saveBlob(await response.blob(), 'aodora-' + kind + '.csv')
}

export function saveBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

export const CATEGORY_COLUMNS = ['name', 'slug', 'description', 'parent_slug', 'sort_order', 'is_active', 'image_url', 'image_alt'] as const
export const PRODUCT_COLUMNS = ['product_slug', 'name', 'category_slug', 'description', 'material', 'price', 'compare_at_price', 'is_active', 'image_url', 'gallery_urls', 'color_name', 'color_hex', 'size', 'sku', 'stock'] as const

// Sample image URLs live under the bucket's public URL: the server only accepts its own images.
const bucketUrl = ((import.meta.env?.VITE_RUSTFS_PUBLIC_URL as string | undefined) ?? 'http://localhost:9100/aodora-images').replace(/\/$/, '')
const sampleImage = (name: string) => bucketUrl + '/uploads/product/2026/01/' + name

const SAMPLE_CATEGORIES = [
  ['Áo dài', 'ao-dai', 'Áo dài truyền thống', '', '1', 'true', sampleImage('ao-dai.jpg'), 'Áo dài'],
  ['Áo dài cưới', 'ao-dai-cuoi', '', 'ao-dai', '2', 'true', '', ''],
]
const SAMPLE_PRODUCTS = [
  ['ao-dai-lua-do', 'Áo dài lụa đỏ', 'ao-dai', 'Áo dài lụa tơ tằm', 'Lụa tơ tằm', '1500000', '1800000', 'true', sampleImage('a.jpg'), sampleImage('a1.jpg') + '|' + sampleImage('a2.jpg'), 'Đỏ đô', '#8b0000', 'S', 'AO-DAI-LUA-DO-S', '10'],
  ['ao-dai-lua-do', '', '', '', '', '', '', '', '', '', 'Đỏ đô', '#8b0000', 'M', 'AO-DAI-LUA-DO-M', '8'],
  ['ao-dai-lua-do', '', '', '', '', '', '', '', '', '', 'Trắng', '#ffffff', 'Freesize', 'AO-DAI-LUA-TRANG-F', '5'],
]

const csvCell = (v: string) => (/[",\n\r]/.test(v) ? '"' + v.replaceAll('"', '""') + '"' : v)
export const toCsv = (rows: string[][]) => rows.map((r) => r.map(csvCell).join(',')).join('\r\n') + '\r\n'

/** CSV template with a UTF-8 BOM (opens correctly in Excel). */
export function templateCsv(kind: ImportKind): string {
  const header = kind === 'categories' ? [...CATEGORY_COLUMNS] : [...PRODUCT_COLUMNS]
  return BOM + toCsv([header, ...(kind === 'categories' ? SAMPLE_CATEGORIES : SAMPLE_PRODUCTS)])
}

export function templateJson(kind: ImportKind): string {
  const columns: readonly string[] = kind === 'categories' ? CATEGORY_COLUMNS : PRODUCT_COLUMNS
  const rows = kind === 'categories' ? SAMPLE_CATEGORIES : SAMPLE_PRODUCTS
  return JSON.stringify(rows.map((row) => Object.fromEntries(columns.map((c, i) => [c, row[i]]))), null, 2) + '\n'
}
