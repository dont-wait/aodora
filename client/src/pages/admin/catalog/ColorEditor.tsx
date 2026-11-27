import { fieldId, isHex, type Errors } from '@/features/admin/validation'
import { Button, Field } from '../ui'
import { inputClass } from '../uiStyles'

type ColorRow = { name: string; hex?: string }

/**
 * Colour list of a product. `locked` maps a colour name (lower-case) to the reason it cannot be renamed or
 * removed (SKUs use it), so SKU colours always stay in sync with this list.
 */
export default function ColorEditor({ colors, onChange, errors, locked = {} }: { colors: ColorRow[]; onChange: (next: ColorRow[]) => void; errors: Errors; locked?: Record<string, string> }) {
  const update = (index: number, patch: Partial<ColorRow>) => onChange(colors.map((row, i) => (i === index ? { ...row, ...patch } : row)))
  return (
    <div className="space-y-3">
      {colors.length === 0 && <p className="text-sm text-on-surface-variant">Chưa có màu nào. Thêm màu trước, sau đó chọn màu khi tạo SKU.</p>}
      {colors.map((row, index) => {
        const hex = row.hex ?? ''
        const reason = locked[row.name.trim().toLowerCase()]
        const p = 'colors[' + index + ']'
        return (
          <div key={index} className="grid items-start gap-2 md:grid-cols-[1fr_auto_9rem_auto]">
            <Field label={'Tên màu ' + (index + 1)} path={p + '.name'} error={errors[p + '.name']} hint={reason}>
              <input className={inputClass} maxLength={50} placeholder="Ví dụ: Đỏ đô" disabled={!!reason} value={row.name} onChange={(event) => update(index, { name: event.target.value })} />
            </Field>
            <Field label="Chọn màu">
              <input
                type="color"
                className="h-10 w-12 cursor-pointer rounded-lg border border-outline-variant bg-surface-container-lowest p-1"
                aria-label={'Chọn màu ' + (index + 1)}
                value={isHex(hex) ? hex : '#000000'}
                onChange={(event) => update(index, { hex: event.target.value.toUpperCase() })}
              />
            </Field>
            <Field label="Mã màu" path={p + '.hex'} error={errors[p + '.hex']}>
              <input className={inputClass + ' font-mono'} placeholder="#RRGGBB" maxLength={7} value={hex} onChange={(event) => update(index, { hex: event.target.value })} />
            </Field>
            <div className="md:pt-6" id={fieldId(p)}>
              <Button variant="danger" disabled={!!reason} title={reason} aria-label={'Xóa màu ' + (index + 1)} onClick={() => onChange(colors.filter((_, i) => i !== index))}>Xóa</Button>
            </div>
          </div>
        )
      })}
      {errors.colors && <p role="alert" className="text-xs text-red-700">{errors.colors}</p>}
      <Button variant="secondary" disabled={colors.length >= 20} onClick={() => onChange([...colors, { name: '', hex: '' }])}>+ Thêm màu</Button>
    </div>
  )
}
