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
