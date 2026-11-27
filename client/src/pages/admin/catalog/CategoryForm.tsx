import type { FormEvent } from 'react'
import type { Category, CategoryInput } from '@/features/admin/catalog'
import { Button, Card, EmptyState, Field } from '../ui'
import { inputClass } from '../uiStyles'

interface Props {
  categories: Category[]
  categoryId: string
  value: CategoryInput
  busy: boolean
  onSelect: (id: string) => void
  onChange: (next: CategoryInput) => void
  onSubmit: (event: FormEvent) => void
}

export default function CategoryForm({ categories, categoryId, value, busy, onSelect, onChange, onSubmit }: Props) {
  return (
    <div className="grid gap-6 lg:grid-cols-[18rem_1fr] items-start">
      <Card title={`Danh mục (${categories.length})`} actions={<Button onClick={() => onSelect('')}>+ Mới</Button>}>
        {categories.length === 0 && <EmptyState title="Chưa có danh mục" />}
        <ul className="space-y-2">
          {categories.map((row) => (
            <li key={row.id}>
              <button
                type="button"
                onClick={() => onSelect(row.id)}
                aria-current={row.id === categoryId ? 'true' : undefined}
                className={`block w-full border p-2 text-left text-sm ${row.id === categoryId ? 'border-primary bg-surface-container' : 'border-outline-variant/40 hover:bg-surface-container-low'}`}
              >
                {row.name} {row.is_active ? '' : '· Đang ẩn'}
              </button>
            </li>
          ))}
        </ul>
      </Card>
      <form onSubmit={onSubmit}>
        <Card title={categoryId ? 'Cập nhật danh mục' : 'Tạo danh mục'}>
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Tên"><input className={inputClass} required value={value.name} onChange={(e) => onChange({ ...value, name: e.target.value })} /></Field>
            <Field label="Slug"><input className={inputClass} required value={value.slug} onChange={(e) => onChange({ ...value, slug: e.target.value })} /></Field>
          </div>
          <Field label="Mô tả"><textarea className={inputClass} rows={3} value={value.description} onChange={(e) => onChange({ ...value, description: e.target.value })} /></Field>
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="Danh mục cha">
              <select className={inputClass} value={value.parent_id} onChange={(e) => onChange({ ...value, parent_id: e.target.value })}>
                <option value="">Không có</option>
                {categories.filter((v) => v.id !== categoryId).map((v) => <option key={v.id} value={v.id}>{v.name}</option>)}
              </select>
            </Field>
            <Field label="Thứ tự"><input className={inputClass} type="number" min="0" value={value.sort_order} onChange={(e) => onChange({ ...value, sort_order: Number(e.target.value) })} /></Field>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <Field label="URL hình danh mục">
              <input className={inputClass} value={value.image?.src || ''} onChange={(e) => onChange({ ...value, image: e.target.value || value.image?.alt ? { src: e.target.value, alt: value.image?.alt || '' } : null })} />
            </Field>
            <Field label="Alt hình danh mục">
              <input className={inputClass} value={value.image?.alt || ''} onChange={(e) => onChange({ ...value, image: e.target.value || value.image?.src ? { src: value.image?.src || '', alt: e.target.value } : null })} />
            </Field>
          </div>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={value.is_active} onChange={(e) => onChange({ ...value, is_active: e.target.checked })} />Hiển thị danh mục</label>
          <Button type="submit" disabled={busy}>Lưu danh mục</Button>
        </Card>
      </form>
    </div>
  )
}
