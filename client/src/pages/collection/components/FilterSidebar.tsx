import { useState, type FormEvent } from 'react'
import { sidebarFilterKeys, type SidebarFilters } from '../sidebarFilters'
import { STANDARD_SIZES, isStandardSize } from '@/features/catalog/sizes'
import { parseVNDInput } from '@/features/catalog/prices'
import MoneyInput from '@/components/ui/MoneyInput'

// One-tap ranges matching the shop's real prices (650k – 2,48tr).
const PRICE_PRESETS: { label: string; min: string; max: string }[] = [
  { label: 'Dưới 1 triệu', min: '', max: '1000000' },
  { label: '1 – 1,5 triệu', min: '1000000', max: '1500000' },
  { label: '1,5 – 2 triệu', min: '1500000', max: '2000000' },
  { label: 'Trên 2 triệu', min: '2000000', max: '' },
]
const asNumber = (value: string) => (value && /^\d+$/.test(value) ? Number(value) : null)
const readValues = (params: URLSearchParams): SidebarFilters => Object.fromEntries(sidebarFilterKeys.map((key) => [key, params.get(key) || ''])) as unknown as SidebarFilters
interface Props { params: URLSearchParams; onApply: (draft: SidebarFilters) => void; onClear: () => void }
const field = 'w-full p-2.5 border border-outline-variant bg-surface text-on-surface focus:outline-primary'
const label = 'block mb-2 font-label-uppercase text-label-uppercase text-on-surface'
export default function FilterSidebar({ params, onApply, onClear }: Props) {
  const [draft, setDraft] = useState<SidebarFilters>(() => readValues(params))
  const [error, setError] = useState('')
  function update(name: keyof SidebarFilters, value: string) { setDraft((previous) => ({ ...previous, [name]: value })) }
  function submit(event: FormEvent) {
    event.preventDefault()
    const minPrice = parseVNDInput(draft.min_price)
    const maxPrice = parseVNDInput(draft.max_price)
    if (minPrice === null || maxPrice === null || (minPrice !== '' && maxPrice !== '' && Number(minPrice) > Number(maxPrice))) {
      setError('Khoảng giá không hợp lệ. Hãy nhập số tiền đầy đủ bằng VNĐ, ví dụ 400.000.'); return
    }
    setError('')
    onApply({ ...Object.fromEntries(Object.entries(draft).map(([k, v]) => [k, v.trim()])),
      min_price: minPrice, max_price: maxPrice } as SidebarFilters)
  }
  return <aside id="filterSidebar" className="lg:col-span-3 space-y-6 bg-surface-container-low p-6 shadow-sm">
    <div className="flex items-center justify-between">
      <h3 className="font-title-editorial text-title-editorial text-primary font-bold">Lọc Chi Tiết</h3>
      <button type="button" onClick={() => { setDraft(readValues(new URLSearchParams())); onClear() }} className="underline text-secondary text-sm">Thiết lập lại</button>
    </div>
    <form onSubmit={submit} className="space-y-5">
      <label className="block"><span className={label}>Tìm theo tên, mô tả</span>
        <input value={draft.search} onChange={(e) => update('search', e.target.value)} className={field} placeholder="Nhập từ khóa..." /></label>
      <label className="block"><span className={label}>Chất liệu</span><input className={field} value={draft.material}
        onChange={(e) => update('material', e.target.value)} placeholder="Ví dụ: lụa, gấm, linen" /></label>
      <label className="block"><span className={label}>Màu sắc</span><input className={field} value={draft.color}
        onChange={(e) => update('color', e.target.value)} placeholder="Ví dụ: đỏ, xanh, hồng" /></label>
      <label className="block"><span className={label}>Kích cỡ</span><select className={field} value={draft.size} onChange={(e) => update('size', e.target.value)}>
        <option value="">Tất cả cỡ</option>
        {(draft.size && !isStandardSize(draft.size) ? [...STANDARD_SIZES, draft.size] : STANDARD_SIZES).map((size) => <option key={size} value={size}>{size}</option>)}
      </select></label>
      <fieldset><legend className={label}>Khoảng giá (VNĐ)</legend><div className="grid grid-cols-2 gap-2">
        <label><span className="text-sm">Từ</span><MoneyInput aria-label="Giá tối thiểu" className={field + ' tabular-nums'} placeholder="800.000" value={asNumber(draft.min_price)} onChange={(v) => update('min_price', v === null ? '' : String(v))} /></label>
        <label><span className="text-sm">Đến</span><MoneyInput aria-label="Giá tối đa" className={field + ' tabular-nums'} placeholder="1.500.000" value={asNumber(draft.max_price)} onChange={(v) => update('max_price', v === null ? '' : String(v))} /></label>
      </div><p className="mt-2 text-xs text-on-surface-variant">Chỉ cần gõ số, dấu chấm tự thêm. Gõ 800k = 800.000, 2tr = 2.000.000.</p>
        <div className="mt-3 flex flex-wrap gap-2">{PRICE_PRESETS.map((preset) => {
          const active = draft.min_price === preset.min && draft.max_price === preset.max
          return <button key={preset.label} type="button" aria-pressed={active} onClick={() => setDraft((previous) => ({ ...previous, min_price: preset.min, max_price: preset.max }))}
            className={`px-3 py-1.5 text-xs border ${active ? 'bg-primary text-on-primary border-primary' : 'border-outline-variant bg-surface text-on-surface hover:border-primary'}`}>{preset.label}</button>
        })}</div></fieldset>
      {error && <p role="alert" className="text-red-700 text-sm">{error}</p>}
      <button type="submit" className="bg-primary text-on-primary px-5 py-3 w-full">Áp dụng bộ lọc</button>
    </form>
    <p className="text-sm text-on-surface-variant">Màu và kích cỡ được lọc cùng nhau: chỉ hiện áo có đúng cặp màu – cỡ bạn chọn.</p>
  </aside>
}
