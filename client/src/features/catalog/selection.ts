import type { CatalogProduct, CatalogVariant } from './types'

export interface CartSelection {
  product_id: string
  product_slug: string
  sku: string
  color: string
  size: string
  quantity: number
  fulfillment_mode: 'ready'
}

// KR-03 can register its real cart integration here. Until then no success is
// reported and nothing is written to fake local state.
export type CartAddHandler = (selection: CartSelection) => Promise<void>
let cartHandler: CartAddHandler | null = null
export const registerCartAddHandler = (handler: CartAddHandler) => {
  cartHandler = handler
  return () => { if (cartHandler === handler) cartHandler = null }
}
export function getSelectedVariant(product: CatalogProduct, color: string, size: string): CatalogVariant | undefined {
  return product.variants.find((v) => v.color === color && v.size === size)
}
export function createCartSelection(product: CatalogProduct, color: string, size: string, quantity: number): CartSelection {
  if (!Number.isSafeInteger(quantity) || quantity < 1) throw new Error('Số lượng không hợp lệ.')
  const variant = getSelectedVariant(product, color, size)
  if (product.variants.length === 0 && product.sku && product.stock >= quantity && product.ready_available) {
    return { product_id: product.id, product_slug: product.slug, sku: product.sku, color: '', size: '', quantity, fulfillment_mode: 'ready' }
  }
  if (!variant || !variant.available || variant.stock < quantity || product.stock < quantity) {
    throw new Error('Màu, kích cỡ hoặc số lượng này không còn hàng. Vui lòng chọn lại.')
  }
  return { product_id: product.id, product_slug: product.slug, sku: variant.sku, color: variant.color, size: variant.size, quantity, fulfillment_mode: 'ready' }
}
export async function addSelectedProduct(selection: CartSelection) {
  if (!cartHandler) throw new Error('Giỏ hàng chưa được tích hợp. Vui lòng thử lại khi chức năng này sẵn sàng.')
  await cartHandler(selection)
}
