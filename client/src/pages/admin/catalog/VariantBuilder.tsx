import { useMemo } from 'react'
import type { VariantDraft } from '@/features/admin/catalog'
import { builderRows, type BuilderState } from '@/features/admin/variantBuilder'
import { STANDARD_SIZES } from '@/features/catalog/sizes'
import { fieldId, normalizeSku, type Errors } from '@/features/admin/validation'
import { Card, DataTable } from '../ui'
import { inputClass, tdClass, thClass } from '../uiStyles'

interface Props {
  slug: string
  /** Colour names of the product (the only colours a SKU may use). */
  colorOptions: string[]
  value: BuilderState
  errors: Errors
  onChange: (next: BuilderState) => void
}

const chip = (on: boolean) => `min-w-12 rounded-lg border px-3 py-1.5 text-sm ${on ? 'border-primary bg-primary text-on-primary' : 'border-outline-variant bg-surface-container-lowest hover:bg-surface-container'}`

/** Create-product helper: size chips x product colours => editable SKU table. State lives in the parent. */
export default function VariantBuilder({ slug, colorOptions, value, errors, onChange }: Props) {
  const rows: VariantDraft[] = useMemo(() => builderRows(slug, value, colorOptions), [slug, value, colorOptions])
  const picked = (name: string) => value.colors.some((c) => c.toLowerCase() === name.toLowerCase())
  const toggleSize = (size: string) => onChange({ ...value, sizes: value.sizes.includes(size) ? value.sizes.filter((s) => s !== size) : [...value.sizes, size] })
  const toggleColor = (name: string) => onChange({ ...value, colors: picked(name) ? value.colors.filter((c) => c.toLowerCase() !== name.toLowerCase()) : [...value.colors, name] })
  const patch = (key: string, edit: { sku?: string; stock?: string }) => onChange({ ...value, edits: { ...value.edits, [key]: { ...value.edits[key], ...edit } } })
  return (
    <Card title="SKU & tồn kho ban đầu" description="Chọn các cỡ và màu (lấy từ danh sách màu ở trên), bảng SKU sẽ được tạo sẵn. Có thể sửa mã SKU và tồn từng dòng.">
      <fieldset className="space-y-2">
        <legend className="mb-1 text-[13px] font-medium text-on-surface-variant">Cỡ (bỏ trống nếu sản phẩm không có cỡ)</legend>
        <div className="flex flex-wrap gap-2">
          {STANDARD_SIZES.map((size) => <button key={size} type="button" aria-pressed={value.sizes.includes(size)} onClick={() => toggleSize(size)} className={chip(value.sizes.includes(size))}>{size}</button>)}
        </div>
      </fieldset>
      <fieldset className="space-y-2">
        <legend className="mb-1 text-[13px] font-medium text-on-surface-variant">Màu</legend>
        {colorOptions.length === 0 ? (
          <p className="text-sm text-on-surface-variant">Thêm ít nhất một màu ở mục "Màu sắc" phía trên để tạo SKU.</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {colorOptions.map((name) => <button key={name} type="button" aria-pressed={picked(name)} onClick={() => toggleColor(name)} className={chip(picked(name))}>{name}</button>)}
            <button type="button" className="px-2 text-sm text-primary underline" onClick={() => onChange({ ...value, colors: [...colorOptions] })}>Chọn tất cả</button>
          </div>
        )}
      </fieldset>
      {errors.variants && <p id={fieldId('variants') + '-error'} role="alert" className="text-xs text-red-700">{errors.variants}</p>}
      {rows.length > 0 && (
        <DataTable label="SKU ban đầu">
          <thead>
            <tr><th className={thClass}>Màu</th><th className={thClass}>Cỡ</th><th className={thClass}>SKU</th><th className={thClass}>Tồn ban đầu</th></tr>
          </thead>
          <tbody>
            {rows.map((row, i) => {
              const p = 'variants[' + i + ']'
              const skuError = errors[p + '.sku'] || errors[p + '.color'] || errors[p + '.size']
              const stockError = errors[p + '.stock']
              return (
                <tr key={row.key}>
                  <td className={tdClass}>{row.color}</td>
                  <td className={tdClass}>{row.size || '—'}</td>
                  <td className={tdClass}>
                    <input id={fieldId(p + '.sku')} className={inputClass + ' min-w-48 font-mono'} aria-label={'SKU ' + row.color + ' ' + row.size} aria-invalid={skuError ? true : undefined} aria-describedby={skuError ? fieldId(p + '.sku') + '-error' : undefined}
                      value={row.sku} onChange={(event) => patch(row.key, { sku: event.target.value.toUpperCase() })} onBlur={(event) => patch(row.key, { sku: normalizeSku(event.target.value) })} />
                    {skuError && <p id={fieldId(p + '.sku') + '-error'} role="alert" className="mt-1 text-xs text-red-700">{skuError}</p>}
                  </td>
                  <td className={tdClass}>
                    <input id={fieldId(p + '.stock')} className={inputClass + ' w-28 tabular-nums'} inputMode="numeric" aria-label={'Tồn ' + row.color + ' ' + row.size} aria-invalid={stockError ? true : undefined} aria-describedby={stockError ? fieldId(p + '.stock') + '-error' : undefined}
                      value={row.stock} onChange={(event) => patch(row.key, { stock: event.target.value })} />
                    {stockError && <p id={fieldId(p + '.stock') + '-error'} role="alert" className="mt-1 text-xs text-red-700">{stockError}</p>}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </DataTable>
      )}
    </Card>
  )
}
