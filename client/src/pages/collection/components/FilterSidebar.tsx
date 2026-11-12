import { useState, type FormEvent } from 'react'
import { sidebarFilterKeys, type SidebarFilters } from '../sidebarFilters'
import { parseVNDInput, formatVNDInput } from '@/features/catalog/prices'
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
        onChange={(e) => update('material', e.target.value)} placeholder="Nhập tên chất liệu chính xác" /></label>
      <label className="block"><span className={label}>Màu sắc</span><input className={field} value={draft.color}
        onChange={(e) => update('color', e.target.value)} placeholder="Tên màu của SKU" /></label>
      <label className="block"><span className={label}>Kích cỡ</span><input className={field} value={draft.size}
        onChange={(e) => update('size', e.target.value)} placeholder="Ví dụ: M" /></label>
      <fieldset><legend className={label}>Khoảng giá (VNĐ)</legend><div className="grid grid-cols-2 gap-2">
        <label><span className="text-sm">Từ</span><input aria-label="Giá tối thiểu" className={field} type="text" inputMode="numeric" placeholder="400.000" value={formatVNDInput(draft.min_price)} onChange={(e) => update('min_price', e.target.value)} /></label>
        <label><span className="text-sm">Đến</span><input aria-label="Giá tối đa" className={field} type="text" inputMode="numeric" placeholder="500.000" value={formatVNDInput(draft.max_price)} onChange={(e) => update('max_price', e.target.value)} /></label>
      </div><p className="mt-2 text-xs text-on-surface-variant">Nhập số tiền đầy đủ (ví dụ 400.000 VNĐ, không phải 400).</p></fieldset>
      {error && <p role="alert" className="text-red-700 text-sm">{error}</p>}
      <button type="submit" className="bg-primary text-on-primary px-5 py-3 w-full">Áp dụng bộ lọc</button>
    </form>
    <p className="text-sm text-on-surface-variant">Bộ lọc được thực hiện trên MongoDB; màu và kích cỡ phải cùng một SKU.</p>
  </aside>
}
