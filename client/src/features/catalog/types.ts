import type { Img } from '@/types'
import type { BadgeVariant } from '@/components/ui/Badge'

export interface CatalogCategory {
  id: string
  key?: string
  slug: string
  name: string
  description?: string
  parent_id?: string
  sort_order: number
  image?: Img
  badge?: { label: string; variant: string }
}
export interface CatalogColor { name?: string; hex?: string }
export interface CatalogVariant { sku: string; color: string; size: string; stock: number; available: boolean }
export interface CatalogLabel { label: string; variant: string }
export interface CatalogProductSummary {
  id: string; category_id: string; slug: string; name: string
  description?: string; material?: string; price: number
  old_price?: number; compare_at_price?: number; image?: Img
  badge?: CatalogLabel; tag?: CatalogLabel; colors: CatalogColor[]
  sizes: string[]; bespoke: boolean; ready_available: boolean; stock: number
  created_at: string
}
export interface CatalogContent { eyebrow?: string; title?: string; description?: string; image?: Img; paragraphs?: string[]; care?: CatalogContentItem[]; steps?: CatalogContentItem[] }
export interface CatalogContentItem { icon?: string; title: string; description: string }
export interface CatalogProduct extends CatalogProductSummary {
  sku?: string; variants: CatalogVariant[]; images: string[]; gallery: Img[]
  color_note?: string; cta?: string; bundle?: string; collection?: string
  discount?: number; availability?: string
  trust: CatalogContentItem[]
  accessories: { id: string; name: string; description: string; price: number; image?: Img; tag?: string; tag_variant?: string }[]
  story?: CatalogContent; fabric?: CatalogContent; craft?: CatalogContent
  reviews: { count: number; average: number }; updated_at: string
}
export interface CatalogPage {
  data: CatalogProductSummary[]; total: number; page: number; page_size: number; total_pages: number
}
export type CatalogSort = 'newest' | 'price_asc' | 'price_desc' | 'name_asc' | 'name_desc'
export interface CatalogFilters {
  search?: string; category?: string; min_price?: string; max_price?: string
  color?: string; size?: string; material?: string; sort?: CatalogSort; page?: number; page_size?: number
}
export const sortOptions: { value: CatalogSort; label: string }[] = [
  { value: 'newest', label: 'Mới nhất' },
  { value: 'price_asc', label: 'Giá tăng dần' },
  { value: 'price_desc', label: 'Giá giảm dần' },
  { value: 'name_asc', label: 'Tên A–Z' },
  { value: 'name_desc', label: 'Tên Z–A' },
]
export const badgeVariant = (name?: string): BadgeVariant => {
  const valid: BadgeVariant[] = ['primary', 'secondary', 'surface', 'tertiary', 'primaryContainer', 'secondaryContainer', 'secondaryFixed']
  return valid.find((v) => v === name) ?? 'surface'
}
