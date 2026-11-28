// One rule set for the admin forms. server/internal/application/catalogadmin/rules.go is the source of
// truth: keep the limits below identical. Errors are keyed by the same field paths the server returns
// (name, price, colors[1].hex, variants[2].sku, accessories[0].price, gallery[3].alt, ...).
import type { Image, ProductInput } from './catalog.ts'
import { STANDARD_SIZES } from '../catalog/sizes.ts'

export type Errors = Record<string, string>

export const LIMITS = {
  nameMin: 3, nameMax: 150, slugMax: 120, descriptionMax: 5000, materialMax: 200,
  priceMin: 1000, priceMax: 1_000_000_000, galleryMax: 12, altMax: 200,
  colorsMax: 20, colorNameMax: 50, accessoriesMax: 20, accessoryNameMax: 100, accessoryDescriptionMax: 500, accessoryPriceMax: 100_000_000, accessoryTagMax: 30,
  stockMax: 100_000, categoryNameMin: 2, categoryNameMax: 100, categoryDescriptionMax: 2000, categorySortMax: 9999,
} as const

export const isHex = (value: string) => /^#[0-9a-fA-F]{6}$/.test(value)

export const SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/
export const SKU_PATTERN = /^[A-Z0-9][A-Z0-9-]{2,39}$/
export const HEX_PATTERN = /^#[0-9a-fA-F]{6}$/

const len = (s: string) => Array.from(s).length

/** Lower-case without diacritics (đ -> d), single spaces: used to compare colour names. */
export function foldText(s: string): string {
  return s.normalize('NFD').replace(/[\u{300}-\u{36f}]/gu, '').replace(/đ/gi, 'd').toLowerCase().split(/\s+/).filter(Boolean).join(' ')
}

/** "áo dài-đỏ m" -> "AO-DAI-DO-M" (same normalisation as the server). */
export function normalizeSku(s: string): string {
  return foldText(s).replaceAll(' ', '-').toUpperCase()
}

// ---- prices ----
/** 650000 -> "650.000"; empty / invalid -> "". */
export function formatThousands(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return ''
  return String(Math.trunc(value)).replace(/\B(?=(\d{3})+(?!\d))/g, '.')
}
/** "650.000" / "650 000" / "650000đ" -> 650000; "" -> null; anything fractional or negative -> NaN. */
export function parseThousands(text: string): number | null {
  const trimmed = text.trim()
  if (trimmed === '') return null
  const digits = trimmed.replace(/[.\s,]/g, '').replace(/(đ|₫|vnd)$/i, '')
  return /^\d+$/.test(digits) ? Number(digits) : Number.NaN
}

function checkText(errors: Errors, path: string, label: string, value: string, min: number, max: number) {
  const trimmed = value.trim()
  const n = len(trimmed)
  if (n >= min && n <= max) return
  errors[path] = min <= 1 ? label + ' bắt buộc, tối đa ' + max + ' ký tự' : label + ' phải từ ' + min + ' đến ' + max + ' ký tự'
}

export function checkImage(errors: Errors, path: string, image: Image | null | undefined) {
  if (!image) return
  if (!image.src) errors[path + '.src'] = 'Ảnh chưa được tải lên'
  const alt = image.alt.trim()
  if (alt.length < 1 || len(alt) > LIMITS.altMax) errors[path + '.alt'] = 'Mô tả ảnh bắt buộc, tối đa ' + LIMITS.altMax + ' ký tự'
}

export type ProductFormValues = Pick<ProductInput, 'name' | 'slug' | 'category_id' | 'description' | 'material' | 'price' | 'compare_at_price' | 'is_active' | 'image' | 'gallery' | 'colors' | 'accessories'>

export function validateProduct(p: ProductFormValues, ctx: { skuCount: number; isNew: boolean; wasActive?: boolean } = { skuCount: 0, isNew: true }): Errors {
  const e: Errors = {}
  checkText(e, 'name', 'Tên sản phẩm', p.name, LIMITS.nameMin, LIMITS.nameMax)
  if (!p.slug || p.slug.length > LIMITS.slugMax || !SLUG_PATTERN.test(p.slug)) e.slug = 'Đường dẫn chỉ gồm chữ thường, số và dấu gạch ngang, tối đa ' + LIMITS.slugMax + ' ký tự'
  if (!p.category_id) e.category_id = 'Chọn danh mục cho sản phẩm'
  if (!Number.isInteger(p.price) || p.price < LIMITS.priceMin || p.price > LIMITS.priceMax) e.price = 'Giá bán phải từ ' + formatThousands(LIMITS.priceMin) + ' đến ' + formatThousands(LIMITS.priceMax) + ' VND'
  if (p.compare_at_price !== null && p.compare_at_price !== undefined) {
    if (!Number.isInteger(p.compare_at_price) || p.compare_at_price <= p.price || p.compare_at_price > LIMITS.priceMax) e.compare_at_price = 'Giá gốc phải lớn hơn giá bán (hoặc để trống)'
  }
  if (len(p.description) > LIMITS.descriptionMax) e.description = 'Mô tả tối đa ' + LIMITS.descriptionMax + ' ký tự'
  if (len(p.material) > LIMITS.materialMax) e.material = 'Chất liệu tối đa ' + LIMITS.materialMax + ' ký tự'

  checkImage(e, 'image', p.image)
  if (p.gallery.length > LIMITS.galleryMax) e.gallery = 'Tối đa ' + LIMITS.galleryMax + ' ảnh phụ'
  // Duplicates of the main image are allowed; the storefront gallery shows each image once.
  p.gallery.forEach((g, i) => checkImage(e, 'gallery[' + i + ']', g))

  if (p.colors.length > LIMITS.colorsMax) e.colors = 'Tối đa ' + LIMITS.colorsMax + ' màu'
  const names = new Set<string>()
  p.colors.forEach((c, i) => {
    checkText(e, 'colors[' + i + '].name', 'Tên màu', c.name, 1, LIMITS.colorNameMax)
    const key = foldText(c.name)
    if (key && names.has(key)) e['colors[' + i + '].name'] = 'Tên màu bị trùng'
    names.add(key)
    if (c.hex && !HEX_PATTERN.test(c.hex)) e['colors[' + i + '].hex'] = 'Mã màu phải có dạng #RRGGBB'
  })

  if (p.accessories.length > LIMITS.accessoriesMax) e.accessories = 'Tối đa ' + LIMITS.accessoriesMax + ' phụ kiện'
  const ids = new Set<string>()
  p.accessories.forEach((a, i) => {
    const path = 'accessories[' + i + ']'
    if (!a.id || ids.has(a.id)) e[path + '.id'] = 'Mã phụ kiện bị trùng'
    ids.add(a.id)
    checkText(e, path + '.name', 'Tên phụ kiện', a.name, 1, LIMITS.accessoryNameMax)
    if (len(a.description) > LIMITS.accessoryDescriptionMax) e[path + '.description'] = 'Mô tả phụ kiện tối đa ' + LIMITS.accessoryDescriptionMax + ' ký tự'
    if (!Number.isInteger(a.price) || a.price < 0 || a.price > LIMITS.accessoryPriceMax) e[path + '.price'] = 'Giá phụ kiện phải từ 0 đến ' + formatThousands(LIMITS.accessoryPriceMax) + ' VND'
    if (a.tag && len(a.tag) > LIMITS.accessoryTagMax) e[path + '.tag'] = 'Nhãn tối đa ' + LIMITS.accessoryTagMax + ' ký tự'
    checkImage(e, path + '.image', a.image)
  })

  const turningOn = p.is_active && (ctx.isNew || !ctx.wasActive)
  if (turningOn && !p.image?.src) e.is_active = 'Cần có ảnh chính trước khi bật Đang bán'
  else if (turningOn && !ctx.isNew && ctx.skuCount < 1) e.is_active = 'Cần có ít nhất 1 SKU trước khi bật Đang bán'
  return e
}

export function validateCategory(c: { name: string; slug: string; description: string; sort_order: number; image: Image | null }): Errors {
  const e: Errors = {}
  checkText(e, 'name', 'Tên danh mục', c.name, LIMITS.categoryNameMin, LIMITS.categoryNameMax)
  if (!c.slug || c.slug.length > LIMITS.slugMax || !SLUG_PATTERN.test(c.slug)) e.slug = 'Đường dẫn chỉ gồm chữ thường, số và dấu gạch ngang, tối đa ' + LIMITS.slugMax + ' ký tự'
  if (len(c.description) > LIMITS.categoryDescriptionMax) e.description = 'Mô tả tối đa ' + LIMITS.categoryDescriptionMax + ' ký tự'
  if (!Number.isInteger(c.sort_order) || c.sort_order < 0 || c.sort_order > LIMITS.categorySortMax) e.sort_order = 'Thứ tự phải từ 0 đến ' + LIMITS.categorySortMax
  checkImage(e, 'image', c.image)
  return e
}

export type SkuRow = { color: string; size: string; sku: string; stock: string }

/** Rows of the create-product SKU table; paths are variants[i].sku / .stock / .color / .size. */
export function validateSkuRows(rows: SkuRow[], colorNames: string[]): Errors {
  const e: Errors = {}
  if (rows.length === 0) { e.variants = 'Chọn ít nhất một màu để tạo SKU'; return e }
  if (rows.length > 100) e.variants = 'Tối đa 100 SKU cho mỗi sản phẩm'
  const skus = new Map<string, number>()
  const combos = new Map<string, number>()
  const colorKeys = new Set(colorNames.map((c) => c.trim().toLowerCase()))
  rows.forEach((row, i) => {
    const p = 'variants[' + i + '].'
    const sku = normalizeSku(row.sku)
    if (!SKU_PATTERN.test(sku)) e[p + 'sku'] = 'SKU gồm 3–40 ký tự: chữ in hoa, số và dấu gạch ngang, bắt đầu bằng chữ hoặc số'
    else if (skus.has(sku)) e[p + 'sku'] = 'SKU bị trùng với dòng ' + ((skus.get(sku) as number) + 1)
    skus.set(sku, skus.get(sku) ?? i)
    if (!colorKeys.has(row.color.trim().toLowerCase())) e[p + 'color'] = 'Màu phải chọn trong danh sách màu của sản phẩm'
    if (row.size && !(STANDARD_SIZES as readonly string[]).includes(row.size)) e[p + 'size'] = 'Cỡ phải là một trong ' + STANDARD_SIZES.join(', ')
    const combo = row.color.trim().toLowerCase() + '|' + row.size
    if (combos.has(combo)) e[p + 'color'] = 'Cặp màu/cỡ bị trùng với dòng ' + ((combos.get(combo) as number) + 1)
    combos.set(combo, combos.get(combo) ?? i)
    const stock = row.stock.trim() === '' ? Number.NaN : Number(row.stock)
    if (!Number.isInteger(stock) || stock < 0 || stock > LIMITS.stockMax) e[p + 'stock'] = 'Tồn phải là số nguyên từ 0 đến ' + formatThousands(LIMITS.stockMax)
  })
  return e
}

/** Stock adjustment box ("+5", "-2"): integer, non-zero, |n| <= 100000, never below zero stock. */
export function validateAdjustment(text: string, currentStock: number): string {
  const t = text.trim()
  if (!/^[+-]?\d+$/.test(t)) return 'Nhập số nguyên khác 0, ví dụ -2 hoặc 5'
  const n = Number(t)
  if (n === 0) return 'Số điều chỉnh phải khác 0'
  if (Math.abs(n) > LIMITS.stockMax) return 'Số điều chỉnh tối đa ' + formatThousands(LIMITS.stockMax)
  if (currentStock + n < 0) return 'Không thể trừ ' + -n + ': kho chỉ còn ' + currentStock
  return ''
}

/** Single new SKU (add form). */
export function validateNewSku(v: { sku: string; color: string; size: string }, colorNames: string[], existing: { sku: string; color: string; size: string }[]): Errors {
  const e: Errors = {}
  const sku = normalizeSku(v.sku)
  if (!SKU_PATTERN.test(sku)) e.sku = 'SKU gồm 3–40 ký tự: chữ in hoa, số và dấu gạch ngang, bắt đầu bằng chữ hoặc số'
  else if (existing.some((o) => o.sku.toLowerCase() === sku.toLowerCase())) e.sku = 'SKU đã tồn tại trong sản phẩm này'
  if (colorNames.length > 0 && !colorNames.some((c) => c.trim().toLowerCase() === v.color.trim().toLowerCase())) e.color = 'Chọn một màu trong danh sách màu của sản phẩm'
  else if (existing.some((o) => o.color.toLowerCase() === v.color.toLowerCase() && o.size === v.size)) e.color = 'Đã có SKU cùng màu và cỡ'
  if (v.size && !(STANDARD_SIZES as readonly string[]).includes(v.size)) e.size = 'Cỡ phải là một trong ' + STANDARD_SIZES.join(', ')
  return e
}

export const errorCount = (errors: Errors) => Object.values(errors).filter(Boolean).length

/** DOM id of the control for a field path: "colors[1].hex" -> "f-colors-1-hex". */
export function fieldId(path: string): string {
  return 'f-' + path.replace(/[[\]. ]+/g, '-').replace(/-+$/g, '')
}

/** Server error field -> form path. Most paths are identical; a few legacy names are aliased. */
export function mapServerField(field: string): string {
  if (!field) return ''
  const alias: Record<string, string> = { images: 'image', 'colors.hex': 'colors', 'accessories.price': 'accessories' }
  return alias[field] ?? field
}
