import { isHexColor } from '@/features/admin/catalog'
import { Button } from '../ui'
import { inputClass } from '../uiStyles'

type ColorRow = { name: string; hex?: string }

export default function ColorEditor({ colors, onChange }: { colors: ColorRow[]; onChange: (next: ColorRow[]) => void }) {
  const update = (index: number, patch: Partial<ColorRow>) => onChange(colors.map((row, i) => (i === index ? { ...row, ...patch } : row)))
  return (
    <div className="space-y-2">
      {colors.length === 0 && <p className="text-sm text-on-surface-variant">Chưa có màu nào. Thêm màu để khách thấy ô chọn màu trên trang sản phẩm.</p>}
      {colors.map((row, index) => {
        const hex = row.hex ?? ''
        const invalid = hex !== '' && !isHexColor(hex)
        return (
          <div key={index} className="flex flex-wrap items-center gap-2">
            <input className={`${inputClass} flex-1 min-w-36`} aria-label={`Tên màu ${index + 1}`} placeholder="Tên màu, ví dụ: Đỏ đô" value={row.name} onChange={(event) => update(index, { name: event.target.value })} />
            <input
              type="color"
              className="h-10 w-12 cursor-pointer border border-outline-variant bg-surface p-1"
              aria-label={`Chọn màu ${index + 1}`}
              value={isHexColor(hex) ? hex : '#000000'}
              onChange={(event) => update(index, { hex: event.target.value })}
            />
            <input
              className={`${inputClass} w-28 font-mono`}
              aria-label={`Mã hex màu ${index + 1}`}
              aria-invalid={invalid}
              placeholder="#RRGGBB"
              maxLength={7}
              value={hex}
              onChange={(event) => update(index, { hex: event.target.value })}
            />
            <Button variant="danger" aria-label={`Xóa màu ${index + 1}`} onClick={() => onChange(colors.filter((_, i) => i !== index))}>Xóa</Button>
            {invalid && <span role="alert" className="basis-full text-xs text-red-700">Mã màu phải có dạng #RRGGBB</span>}
          </div>
        )
      })}
      <Button variant="secondary" onClick={() => onChange([...colors, { name: '', hex: '' }])}>+ Thêm màu</Button>
    </div>
  )
}
