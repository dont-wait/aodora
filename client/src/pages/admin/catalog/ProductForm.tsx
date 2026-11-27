import type { FormEvent, ReactNode } from 'react'
import type { Category, ProductInput } from '@/features/admin/catalog'
import { fieldId, type Errors } from '@/features/admin/validation'
import ImageUploader from '../ImageUploader'
import { AutoTextarea, Button, Card, ErrorSummary, Field, MoneyInput } from '../ui'
import { inputClass } from '../uiStyles'
import AccessoryEditor from './AccessoryEditor'
import ColorEditor from './ColorEditor'
import SlugField from './SlugField'

interface Props {
  productId: string
  product: ProductInput
  originalSlug: string
  categories: Category[]
  errors: Errors
  busy: boolean
  /** Number of SKUs already saved (existing product). */
  skuCount: number
  /** Colour name (lower-case) -> reason it is locked because SKUs use it. */
  lockedColors: Record<string, string>
  onChange: (next: ProductInput) => void
  onName: (name: string) => void
  onSlug: (slug: string) => void
  onSubmit: (event: FormEvent) => void
  /** Create mode: the SKU builder is rendered inside the form, before the submit button. */
  children?: ReactNode
}

export default function ProductForm({ productId, product, originalSlug, categories, errors, busy, skuCount, lockedColors, onChange, onName, onSlug, onSubmit, children }: Props) {
  const set = (patch: Partial<ProductInput>) => onChange({ ...product, ...patch })
  const isNew = !productId
  const category = categories.find((c) => c.id === product.category_id)
  const canActivate = !!product.image?.src && (isNew || skuCount > 0)
  const activeHint = product.is_active || canActivate ? undefined : !product.image?.src ? 'Cần có ảnh chính trước khi bật Đang bán.' : 'Cần có ít nhất 1 SKU trước khi bật Đang bán.'
  const promote = (index: number) => {
    const picked = product.gallery[index]
    const rest = product.gallery.filter((_, i) => i !== index)
    set({ image: picked, gallery: product.image ? [product.image, ...rest] : rest })
  }
  return (
    <form className="space-y-5" onSubmit={onSubmit} noValidate>
      <ErrorSummary errors={errors} />
      <Card title="Thông tin chung" description={productId ? 'Lưu thông tin sẽ không thay đổi tồn kho, danh sách SKU, rating hay review_count.' : undefined}>
        <Field label="Tên sản phẩm" path="name" error={errors.name} required hint="3–150 ký tự">
          <input className={inputClass} maxLength={150} value={product.name} onChange={(e) => onName(e.target.value)} />
        </Field>
        <SlugField slug={product.slug} onChange={onSlug} originalSlug={isNew ? undefined : originalSlug} error={errors.slug} prefix="/san-pham/" />
        <Field label="Danh mục" path="category_id" error={errors.category_id} required>
          <select className={inputClass} value={product.category_id} onChange={(e) => set({ category_id: e.target.value })}>
            <option value="">Chọn danh mục</option>
            {categories.map((v) => <option key={v.id} value={v.id}>{v.name}{v.is_active ? '' : ' (đang ẩn)'}</option>)}
          </select>
        </Field>
        {category && !category.is_active && <p role="status" className="text-xs text-amber-800">Danh mục này đang ẩn nên khách chưa thấy sản phẩm trong cửa hàng.</p>}
        <Field label="Mô tả" path="description" error={errors.description} hint={product.description.length + '/5000 ký tự'}>
          <AutoTextarea value={product.description} onChange={(e) => set({ description: e.target.value })} />
        </Field>
        <Field label="Chất liệu" path="material" error={errors.material} hint="Tối đa 200 ký tự">
          <input className={inputClass} maxLength={200} value={product.material} onChange={(e) => set({ material: e.target.value })} />
        </Field>
        <div className="space-y-1">
          <label className="flex items-center gap-2 text-sm">
            <input id={fieldId('is_active')} type="checkbox" checked={product.is_active} disabled={!canActivate && !product.is_active} aria-invalid={errors.is_active ? true : undefined} aria-describedby={activeHint || errors.is_active ? fieldId('is_active') + '-error' : undefined} onChange={(e) => set({ is_active: e.target.checked })} />
            Đang bán (hiển thị với khách)
          </label>
          {(errors.is_active || activeHint) && <p id={fieldId('is_active') + '-error'} role={errors.is_active ? 'alert' : undefined} className="text-xs text-on-surface-variant">{errors.is_active || activeHint}</p>}
        </div>
      </Card>

      <Card title="Giá">
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Giá bán (VND)" path="price" error={errors.price} required hint="Từ 1.000 đến 1.000.000.000 · chỉ gõ số, hoặc 800k, 2tr">
            <MoneyInput value={product.price > 0 ? product.price : null} onChange={(v) => set({ price: v ?? 0 })} />
          </Field>
          <Field label="Giá gốc (nếu đang giảm giá)" path="compare_at_price" error={errors.compare_at_price} hint="Phải lớn hơn giá bán; để trống nếu không giảm giá">
            <MoneyInput value={product.compare_at_price ?? null} onChange={(v) => set({ compare_at_price: v })} />
          </Field>
        </div>
      </Card>

      <Card title="Hình ảnh" description="Tải ảnh lên trực tiếp; ảnh chính hiển thị đầu tiên trên trang sản phẩm.">
        <ImageUploader
          purpose="product"
          label="Ảnh chính"
          items={product.image ? [product.image] : []}
          onChange={(items) => set({ image: items[0] ?? null })}
          altBase={product.name}
          pathPrefix="image"
          errors={errors}
        />
        <ImageUploader
          purpose="product"
          label="Ảnh phụ"
          multiple
          max={12}
          items={product.gallery}
          onChange={(gallery) => set({ gallery })}
          onPromote={promote}
          altBase={product.name}
          pathPrefix="gallery"
          errors={errors}
          hint="Tối đa 12 ảnh. Kéo thả hoặc dùng nút ← → để sắp xếp."
        />
      </Card>

      <Card title="Màu sắc" description="Tên màu hiển thị cho khách. SKU chỉ được chọn màu có trong danh sách này.">
        <ColorEditor colors={product.colors} onChange={(colors) => set({ colors })} errors={errors} locked={lockedColors} />
      </Card>

      <Card title="Phụ kiện" description="Phụ kiện đi kèm gợi ý cho khách trên trang sản phẩm.">
        <AccessoryEditor accessories={product.accessories} onChange={(accessories) => set({ accessories })} errors={errors} productName={product.name} />
      </Card>

      {children}

      <div className="flex justify-end"><Button type="submit" loading={busy}>{productId ? 'Lưu sản phẩm' : 'Tạo sản phẩm'}</Button></div>
    </form>
  )
}
