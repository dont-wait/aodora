export type Image = { src: string; alt: string }
export type CategoryInput = {
  name: string; slug: string; description: string; parent_id: string
  is_active: boolean; sort_order: number; image?: Image
}
export type Category = CategoryInput & { id: string }
export type Variant = { sku: string; color: string; size: string; stock: number }
export type ProductInput = {
  name: string; slug: string; category_id: string; description: string; material: string
  price: number; compare_at_price: number | null; is_active: boolean
  images: string[]; image?: Image; gallery: Image[]
  colors: { name: string; hex: string }[]
  accessories: { id: string; name: string; description: string; price: number; image: Image }[]
}
export type Product = ProductInput & { id: string; stock: number; variants: Variant[] }
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
  createProduct: (v: ProductInput & { variants: Variant[] }) => request<Product>('/products', 'POST', v),
  updateProduct: (id: string, v: ProductInput) => request<Product>(`/products/${encodeURIComponent(id)}`, 'PUT', v),
  addVariant: (id: string, v: Variant) => request<Product>(`/products/${encodeURIComponent(id)}/variants`, 'POST', v),
  updateVariant: (id: string, sku: string, v: Pick<Variant, 'color' | 'size'>) => request<Product>(`/products/${encodeURIComponent(id)}/variants/${encodeURIComponent(sku)}`, 'PATCH', v),
  adjustStock: (id: string, sku: string, delta: number) => request<Product>(`/products/${encodeURIComponent(id)}/inventory/${encodeURIComponent(sku)}`, 'POST', { delta }),
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
    const stock = Number(parts[3])
    if (!Number.isSafeInteger(stock) || stock < 0) throw new Error('Tồn kho SKU phải là số nguyên không âm')
    return { sku: parts[0], color: parts[1], size: parts[2], stock }
  })
}
