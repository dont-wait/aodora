import type { FormEvent, ReactNode } from 'react'
import type { Category, ProductInput } from '@/features/admin/catalog'
import ColorEditor from './ColorEditor'
import { Button, Card, Field } from '../ui'
import { inputClass } from '../uiStyles'

interface Props {
  productId: string
  product: ProductInput
  categories: Category[]
  galleryText: string
  accessoriesText: string
  busy: boolean
  onChange: (next: ProductInput) => void
  onGalleryText: (value: string) => void
  onAccessoriesText: (value: string) => void
  onSubmit: (event: FormEvent) => void
  /** Create mode: the SKU builder is rendered inside the form, before the submit button. */
  children?: ReactNode
}

function Thumb({ src, alt }: { src: string; alt: string }) {
  return (
    <span className="block h-28 w-24 shrink-0 overflow-hidden border border-outline-variant/40 bg-surface-container">
      {src ? <img src={src} alt={alt} className="h-full w-full object-cover" onError={(event) => { event.currentTarget.style.visibility = 'hidden' }} /> : null}
    </span>
  )
}

export default function ProductForm({ productId, product, categories, galleryText, accessoriesText, busy, onChange, onGalleryText, onAccessoriesText, onSubmit, children }: Props) {
  const set = (patch: Partial<ProductInput>) => onChange({ ...product, ...patch })
  const galleryUrls = galleryText.split('\n').map((line) => line.split('|')[0].trim()).filter(Boolean)
  return (
    <form className="space-y-5" onSubmit={onSubmit}>
      <Card title="Thông tin chung" description={productId ? 'Lưu thông tin sẽ không thay đổi tồn kho, danh sách SKU, rating hay review_count.' : undefined}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Tên sản phẩm"><input required className={inputClass} value={product.name} onChange={(e) => set({ name: e.target.value })} /></Field>
          <Field label="Slug"><input required className={inputClass} value={product.slug} onChange={(e) => set({ slug: e.target.value })} /></Field>
        </div>
        <Field label="Danh mục">
          <select required className={inputClass} value={product.category_id} onChange={(e) => set({ category_id: e.target.value })}>
            <option value="">Chọn danh mục</option>
            {categories.map((v) => <option key={v.id} value={v.id}>{v.name}</option>)}
          </select>
        </Field>
        <Field label="Mô tả"><textarea className={inputClass} rows={3} value={product.description} onChange={(e) => set({ description: e.target.value })} /></Field>
        <Field label="Chất liệu"><input className={inputClass} value={product.material} onChange={(e) => set({ material: e.target.value })} /></Field>
        <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={product.is_active} onChange={(e) => set({ is_active: e.target.checked })} />Đang bán</label>
      </Card>

      <Card title="Giá">
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Giá bán (VND)"><input className={inputClass} type="number" min="0" step="1" required value={product.price} onChange={(e) => set({ price: Number(e.target.value) })} /></Field>
          <Field label="Giá gốc (để trống nếu không có)">
            <input className={inputClass} type="number" min="0" step="1" value={product.compare_at_price ?? ''} onChange={(e) => set({ compare_at_price: e.target.value === '' ? null : Number(e.target.value) })} />
          </Field>
        </div>
      </Card>

      <Card title="Hình ảnh">
        <div className="flex flex-wrap gap-4">
          <Thumb src={product.image?.src || ''} alt={product.image?.alt || 'Xem trước hình chính'} />
          <div className="grid flex-1 min-w-64 gap-4 md:grid-cols-2 content-start">
            <Field label="URL hình chính"><input className={inputClass} value={product.image?.src || ''} onChange={(e) => set({ image: e.target.value || product.image?.alt ? { src: e.target.value, alt: product.image?.alt || '' } : null })} /></Field>
            <Field label="Alt hình chính"><input className={inputClass} value={product.image?.alt || ''} onChange={(e) => set({ image: e.target.value || product.image?.src ? { src: product.image?.src || '', alt: e.target.value } : null })} /></Field>
          </div>
        </div>
        <Field label="Ảnh phụ (mỗi dòng: URL | mô tả ảnh)"><textarea className={inputClass} rows={3} value={galleryText} onChange={(e) => onGalleryText(e.target.value)} /></Field>
        {galleryUrls.length > 0 && <div className="flex flex-wrap gap-2" aria-label="Xem trước ảnh phụ">{galleryUrls.map((src, index) => <Thumb key={index + src} src={src} alt={'Ảnh phụ ' + (index + 1)} />)}</div>}
      </Card>

      <Card title="Màu sắc" description="Tên màu và mã màu hiển thị cho khách. Không ảnh hưởng tới SKU.">
        <ColorEditor colors={product.colors} onChange={(colors) => set({ colors })} />
      </Card>

      <Card title="Phụ kiện" description="JSON: id, name, description, price, image (tùy chọn), tag, tag_variant.">
        <Field label="Danh sách phụ kiện (JSON)"><textarea className={`${inputClass} font-mono text-xs`} rows={5} value={accessoriesText} onChange={(e) => onAccessoriesText(e.target.value)} /></Field>
      </Card>

      {children}

      <div className="flex justify-end"><Button type="submit" disabled={busy}>{productId ? 'Lưu sản phẩm' : 'Tạo sản phẩm'}</Button></div>
    </form>
  )
}
