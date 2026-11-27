import { STANDARD_SIZES } from '../catalog/sizes.ts'
import type { ProductContentInput, ProductContentRecord } from './content'
export type Image = { src: string; alt: string }
export type CategoryInput = {
  name: string; slug: string; description: string; parent_id: string
  is_active: boolean; sort_order: number; image: Image | null; expected_version?: number
}
export type Category = CategoryInput & { id: string; version?: number }
export type Variant = { sku: string; color: string; size: string; stock: number }
export type ProductInput = {
  name: string; slug: string; category_id: string; description: string; material: string
  price: number; compare_at_price: number | null; is_active: boolean
  images: string[]; image: Image | null; gallery: Image[]
  colors: { name: string; hex?: string }[]
  accessories: { id: string; name: string; description: string; price: number; image?: Image | null; tag?: string; tag_variant?: string }[]
  expected_version?: number
}
export type Product = ProductInput & { id: string; stock: number; variants: Variant[]; old_price?: number | null; version?: number }
export type ProductPage = { items: Product[]; total: number; page: number; page_size: number; total_pages: number }

const apiBase = (import.meta.env?.VITE_API_BASE_URL ?? '').replace(/\/$/, '')
export class AdminApiError extends Error {
  status: number
  field: string
  constructor(status: number, field: string, message: string) { super(message); this.status = status; this.field = field }
}
async function request<T>(path: string, method = 'GET', input?: unknown): Promise<T> {
  const response = await fetch(`${apiBase}/api/v1/admin${path}`, {
    method, credentials: 'include',
    headers: { Accept: 'application/json', ...(input === undefined ? {} : { 'Content-Type': 'application/json' }) },
    body: input === undefined ? undefined : JSON.stringify(input),
  })
  const body = await response.json().catch(() => null) as { data?: T; error?: { message?: string; field?: string } } | null
  if (!response.ok) throw new AdminApiError(response.status, body?.error?.field || '', body?.error?.message || 'Không thể xử lý yêu cầu')
  if (body?.data === undefined) throw new AdminApiError(response.status, '', 'Dữ liệu trả về không hợp lệ')
  return body.data
}
export const adminCatalog = {
  categories: () => request<Category[]>('/categories'),
  products: (page = 1) => request<ProductPage>(`/products?page=${page}&page_size=50`),
  createCategory: (v: CategoryInput) => request<Category>('/categories', 'POST', v),
  updateCategory: (id: string, v: CategoryInput) => request<Category>(`/categories/${encodeURIComponent(id)}`, 'PUT', v),
  content: (id: string) => request<ProductContentRecord>(`/products/${encodeURIComponent(id)}/content`),
  updateContent: (id: string, v: ProductContentInput) => request<ProductContentRecord>(`/products/${encodeURIComponent(id)}/content`, 'PUT', v),
  createProduct: (v: ProductInput & { variants: Variant[] }) => request<Product>('/products', 'POST', v),
  updateProduct: (id: string, v: ProductInput) => request<Product>(`/products/${encodeURIComponent(id)}`, 'PUT', v),
  addVariant: (id: string, v: Variant) => request<Product>(`/products/${encodeURIComponent(id)}/variants`, 'POST', v),
  updateVariant: (id: string, sku: string, v: Pick<Variant, 'color' | 'size'>) => request<Product>(`/products/${encodeURIComponent(id)}/variants/${encodeURIComponent(sku)}`, 'PATCH', v),
  adjustStock: (id: string, sku: string, delta: number, requestId: string) => request<Product>(`/products/${encodeURIComponent(id)}/inventory/${encodeURIComponent(sku)}`, 'POST', { delta, request_id: requestId }),
}

export function parseImageLines(input: string): Image[] {
  return input.split('\n').map((line) => line.trim()).filter(Boolean).map((line) => {
    const separator = line.indexOf('|')
    if (separator < 1 || !line.slice(separator + 1).trim()) throw new Error('Ảnh phụ phải theo dạng URL | mô tả ảnh')
    return { src: line.slice(0, separator).trim(), alt: line.slice(separator + 1).trim() }
  })
}
export function parseVariantLines(input: string): Variant[] {
  return input.split('\n').map((line) => line.trim()).filter(Boolean).map((line) => {
    const parts = line.split('|').map((part) => part.trim())
    if (parts.length !== 4 || parts.slice(0, 3).some((part) => !part)) throw new Error('Mỗi SKU cần SKU | màu | cỡ | tồn kho')
    if (!parts[3]) throw new Error('Phải nhập tồn kho SKU, kể cả khi là 0')
    const stock = Number(parts[3])
    if (!Number.isSafeInteger(stock) || stock < 0) throw new Error('Tồn kho SKU phải là số nguyên không âm')
    return { sku: parts[0], color: parts[1], size: parts[2], stock }
  })
}

// Legacy products may store URLs only in images[], without gallery entries.
// Make every URL visible in the editor; deleting a line explicitly removes
// that image rather than silently preserving an invisible hidden copy.
export function editableGallery(product: Pick<Product, 'name' | 'image' | 'images' | 'gallery'>): Image[] {
  const primary = product.image?.src || product.images?.[0] || ''
  const seen = new Set(primary ? [primary] : [])
  const items: Image[] = []
  for (const item of product.gallery || []) {
    if (item.src && !seen.has(item.src)) {
      items.push(item)
      seen.add(item.src)
    }
  }
  for (const src of product.images || []) {
    if (src && !seen.has(src)) {
      items.push({ src, alt: product.name })
      seen.add(src)
    }
  }
  return items
}

/** Upper-case ASCII fragment for SKUs: "Đỏ đô" -> "DO-DO". */
export function skuPart(value: string): string {
  return value
    .normalize('NFD')
    .replace(/[\u{300}-\u{36f}]/gu, '')
    .replace(/đ/gi, 'd')
    .toUpperCase()
    .replace(/[^A-Z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

/** Suggested SKU: SLUG-COLOR-SIZE; empty parts are skipped. */
export function suggestSku(slug: string, color: string, size: string): string {
  return [skuPart(slug), skuPart(color), skuPart(size)].filter(Boolean).join('-')
}

/** Splits comma/newline separated colours into unique trimmed names (case-insensitive), keeping order. */
export function parseColorNames(input: string): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const part of input.split(/[,\n]/)) {
    const name = part.trim()
    const key = name.toLowerCase()
    if (name && !seen.has(key)) { seen.add(key); out.push(name) }
  }
  return out
}

export type VariantDraft = { key: string; color: string; size: string; sku: string; stock: string }
export type VariantEdits = Record<string, { sku?: string; stock?: string }>
export const variantKey = (color: string, size: string) => color + '||' + size

/**
 * Rows for the create-product builder: every colour x every chosen size (no sizes => one row per colour
 * with an empty size). Hand edits to sku/stock survive regenerating; untouched SKUs follow the slug.
 */
export function buildVariantDrafts(slug: string, colors: string[], sizes: string[], edits: VariantEdits = {}): VariantDraft[] {
  const ordered = STANDARD_SIZES.filter((size) => sizes.includes(size))
  const sizeList: string[] = ordered.length ? [...ordered] : ['']
  return colors.flatMap((color) =>
    sizeList.map((size) => {
      const key = variantKey(color, size)
      return { key, color, size, sku: edits[key]?.sku ?? suggestSku(slug, color, size), stock: edits[key]?.stock ?? '0' }
    }),
  )
}

/** Validates builder rows and converts them to API variants. */
export function draftsToVariants(rows: VariantDraft[]): Variant[] {
  if (rows.length === 0) throw new Error('Thêm ít nhất một màu để tạo SKU')
  return rows.map((row) => {
    const sku = row.sku.trim()
    if (!sku) throw new Error('SKU của ' + row.color + (row.size ? ' / ' + row.size : '') + ' đang trống')
    const stock = Number(row.stock.trim() === '' ? NaN : row.stock)
    if (!Number.isSafeInteger(stock) || stock < 0) throw new Error('Tồn kho của ' + sku + ' phải là số nguyên không âm')
    return { sku, color: row.color, size: row.size, stock }
  })
}

/** Parses the accessories JSON box; throws a readable message instead of a raw SyntaxError. */
export function parseAccessoriesJson(text: string): ProductInput['accessories'] {
  let value: unknown
  try {
    value = JSON.parse(text.trim() || '[]')
  } catch (error) {
    throw new Error('Phụ kiện: JSON không hợp lệ (' + (error instanceof Error ? error.message : 'lỗi cú pháp') + ')')
  }
  if (!Array.isArray(value)) throw new Error('Phụ kiện phải là một mảng JSON, ví dụ []')
  return value as ProductInput['accessories']
}

export const isHexColor = (value: string) => /^#[0-9a-fA-F]{6}$/.test(value)

/** Drops blank rows; sends hex only when it is a valid #rrggbb. */
export function cleanColors(colors: { name: string; hex?: string }[]): { name: string; hex?: string }[] {
  const out: { name: string; hex?: string }[] = []
  for (const color of colors) {
    const name = color.name.trim()
    if (!name) continue
    const hex = (color.hex ?? '').trim()
    if (hex && !isHexColor(hex)) throw new Error('Mã màu của "' + name + '" phải có dạng #RRGGBB')
    out.push(hex ? { name, hex } : { name })
  }
  return out
}
