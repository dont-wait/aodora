export interface Img {
  src: string
  alt: string
}

export type Variant = 'primary' | 'secondary' | 'surface' | 'tertiary' | 'primaryContainer' | 'secondaryContainer'

export interface Product {
  id: string
  slug: string
  name: string
  material: string
  description: string
  price: number
  oldPrice?: number
  image: Img
  /** Nhãn chính hiển thị góc trên trái */
  badge: { label: string; variant: Variant }
  /** Nhãn phụ (ví dụ nhóm sản phẩm) */
  tag?: { label: string; variant: 'surface' | 'secondaryContainer' }
  category: CategoryId
  colors: string[]
  colorNote: string
  sizes: string[]
  bespoke: boolean
  cta: string
}

export type CategoryId = 'co-phuc' | 'dan-gian' | 'cach-tan' | 'hy-su'

export interface Category {
  id: CategoryId
  name: string
  count: number
}
