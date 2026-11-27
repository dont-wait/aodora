import type { FormEvent } from 'react'
import type { Category, CategoryInput } from '@/features/admin/catalog'
import type { Errors } from '@/features/admin/validation'
import ImageUploader from '../ImageUploader'
import { AutoTextarea, Button, Card, EmptyState, ErrorSummary, Field } from '../ui'
import { inputClass } from '../uiStyles'
import SlugField from './SlugField'

interface Props {
  categories: Category[]
  categoryId: string
  originalSlug: string
  value: CategoryInput
  errors: Errors
  busy: boolean
  onSelect: (id: string) => void
  onChange: (next: CategoryInput) => void
  onName: (name: string) => void
  onSlug: (slug: string) => void
  onSubmit: (event: FormEvent) => void
}

export default function CategoryForm({ categories, categoryId, originalSlug, value, errors, busy, onSelect, onChange, onName, onSlug, onSubmit }: Props) {
  return (
    <div className="grid items-start gap-6 lg:grid-cols-[18rem_1fr]">
      <Card title={'Danh mục (' + categories.length + ')'} actions={<Button onClick={() => onSelect('')}>+ Mới</Button>}>
        {categories.length === 0 && <EmptyState title="Chưa có danh mục" />}
        <ul className="space-y-2">
          {categories.map((row) => (
            <li key={row.id}>
              <button
                type="button"
                onClick={() => onSelect(row.id)}
                aria-current={row.id === categoryId ? 'true' : undefined}
                className={`block w-full rounded-lg border p-2 text-left text-sm ${row.id === categoryId ? 'border-primary bg-surface-container' : 'border-outline-variant/40 hover:bg-surface-container-low'}`}
              >
                {row.name} {row.is_active ? '' : '· Đang ẩn'}
              </button>
            </li>
          ))}
        </ul>
      </Card>
      <form onSubmit={onSubmit} noValidate className="space-y-4">
        <ErrorSummary errors={errors} />
        <Card title={categoryId ? 'Cập nhật danh mục' : 'Tạo danh mục'}>
          <Field label="Tên danh mục" path="name" error={errors.name} required hint="2–100 ký tự">
            <input className={inputClass} maxLength={100} value={value.name} onChange={(e) => onName(e.target.value)} />
          </Field>
          <SlugField slug={value.slug} onChange={onSlug} originalSlug={categoryId ? originalSlug : undefined} error={errors.slug} prefix="/bo-suu-tap?category=" />
          <Field label="Mô tả" path="description" error={errors.description} hint={value.description.length + '/2000 ký tự'}>
            <AutoTextarea value={value.description} onChange={(e) => onChange({ ...value, description: e.target.value })} />
          </Field>
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Danh mục cha" path="parent_id" error={errors.parent_id}>
              <select className={inputClass} value={value.parent_id} onChange={(e) => onChange({ ...value, parent_id: e.target.value })}>
                <option value="">Không có</option>
                {categories.filter((v) => v.id !== categoryId).map((v) => <option key={v.id} value={v.id}>{v.name}</option>)}
              </select>
            </Field>
            <Field label="Thứ tự hiển thị" path="sort_order" error={errors.sort_order} hint="Số nhỏ hiển thị trước (0–9999)">
              <input className={inputClass} type="number" min="0" max="9999" step="1" value={value.sort_order} onChange={(e) => onChange({ ...value, sort_order: e.target.value === '' ? 0 : Number(e.target.value) })} />
            </Field>
          </div>
          <ImageUploader purpose="category" label="Ảnh danh mục" items={value.image ? [value.image] : []} onChange={(items) => onChange({ ...value, image: items[0] ?? null })} altBase={value.name} pathPrefix="image" errors={errors} />
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={value.is_active} onChange={(e) => onChange({ ...value, is_active: e.target.checked })} />Hiển thị danh mục</label>
        </Card>
        <div className="flex justify-end"><Button type="submit" loading={busy}>Lưu danh mục</Button></div>
      </form>
    </div>
  )
}
