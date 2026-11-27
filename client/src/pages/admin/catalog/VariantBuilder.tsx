import { useMemo } from 'react'
import { builderRows, type BuilderState } from '@/features/admin/variantBuilder'
import { STANDARD_SIZES } from '@/features/catalog/sizes'
import { Card, DataTable } from '../ui'
import { inputClass, tdClass, thClass } from '../uiStyles'

interface Props {
  slug: string
  value: BuilderState
  onChange: (next: BuilderState) => void
}

/** Create-product helper: size chips x colours => editable SKU table. State lives in the parent. */
export default function VariantBuilder({ slug, value, onChange }: Props) {
  const rows = useMemo(() => builderRows(slug, value), [slug, value])
  const toggle = (size: string) => onChange({ ...value, sizes: value.sizes.includes(size) ? value.sizes.filter((item) => item !== size) : [...value.sizes, size] })
  const patch = (key: string, edit: { sku?: string; stock?: string }) => onChange({ ...value, edits: { ...value.edits, [key]: { ...value.edits[key], ...edit } } })
  return (
    <Card title="SKU & tồn kho ban đầu" description="Chọn các cỡ và nhập các màu, bảng SKU sẽ được tạo sẵn. Có thể sửa mã SKU và tồn từng dòng.">
      <fieldset className="space-y-2">
        <legend className="text-sm font-medium mb-1">Cỡ (bỏ trống nếu sản phẩm không có cỡ)</legend>
        <div className="flex flex-wrap gap-2">
          {STANDARD_SIZES.map((size) => (
            <button
              key={size}
              type="button"
              aria-pressed={value.sizes.includes(size)}
              onClick={() => toggle(size)}
              className={`min-w-12 border px-3 py-1.5 text-sm ${value.sizes.includes(size) ? 'border-primary bg-primary text-on-primary' : 'border-outline-variant bg-surface hover:bg-surface-container'}`}
            >
              {size}
            </button>
          ))}
        </div>
      </fieldset>
      <label className="flex flex-col gap-1.5 text-sm font-medium">
        Màu (cách nhau bằng dấu phẩy)
        <input
          className={inputClass}
          placeholder="Đỏ đô, Xanh ngọc, Trắng"
          value={value.colorText}
          onChange={(event) => onChange({ ...value, colorText: event.target.value })}
        />
      </label>
      {rows.length === 0 ? (
        <p className="text-sm text-on-surface-variant">Nhập ít nhất một màu để tạo bảng SKU.</p>
      ) : (
        <DataTable label="SKU ban đầu">
          <thead>
            <tr><th className={thClass}>Màu</th><th className={thClass}>Cỡ</th><th className={thClass}>SKU</th><th className={thClass}>Tồn ban đầu</th></tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.key}>
                <td className={tdClass}>{row.color}</td>
                <td className={tdClass}>{row.size || '—'}</td>
                <td className={tdClass}><input className={`${inputClass} min-w-48 font-mono`} aria-label={`SKU ${row.color} ${row.size}`} value={row.sku} onChange={(event) => patch(row.key, { sku: event.target.value })} /></td>
                <td className={tdClass}><input className={`${inputClass} w-24`} inputMode="numeric" aria-label={`Tồn ${row.color} ${row.size}`} value={row.stock} onChange={(event) => patch(row.key, { stock: event.target.value })} /></td>
              </tr>
            ))}
          </tbody>
        </DataTable>
      )}
    </Card>
  )
}
