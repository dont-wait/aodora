import type { CatalogProduct } from './types'
import type { Img } from '../../types'

export function productGallery(product: Pick<CatalogProduct, 'name' | 'image' | 'gallery' | 'images'>): Img[] {
  const seen = new Set<string>()
  return [product.image, ...(product.gallery || []), ...(product.images || []).map((src) => ({ src, alt: product.name }))]
    .filter((image): image is Img => {
      if (!image?.src || seen.has(image.src)) return false
      seen.add(image.src)
      return true
    })
}
