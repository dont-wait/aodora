import { useState } from 'react'
import { Link } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import Modal from '@/components/ui/Modal'
import { SIZE_CHART, STANDARD_SIZES, isStandardSize, suggestSize, type StandardSize } from '@/features/catalog/sizes'

const range = ([min, max]: [number, number]) => `${min}–${max}`
const COLUMNS = [
  ['bust', 'Vòng ngực'],
  ['waist', 'Vòng eo'],
  ['hip', 'Vòng mông'],
  ['height', 'Chiều cao'],
  ['weight', 'Cân nặng (kg)'],
] as const

export default function SizeGuideModal({ open, onClose, sizes }: { open: boolean; onClose: () => void; sizes: string[] }) {
  const offered = new Set(sizes)
  // Show the product's sizes; if it lists none we know, show the whole chart.
  const rows = STANDARD_SIZES.filter((size) => SIZE_CHART[size] && (offered.size === 0 || offered.has(size) || !sizes.some(isStandardSize)))
  const [body, setBody] = useState({ bust: '', waist: '', hip: '' })
  const suggestion = body.bust && body.waist && body.hip ? suggestSize(Number(body.bust), Number(body.waist), Number(body.hip)) : undefined
  const available = suggestion ? offered.size === 0 || offered.has(suggestion) : false

  return <Modal open={open} onClose={onClose} labelledBy="size-guide-title">
    <div className="flex items-center justify-between pb-4">
      <div><p className="uppercase tracking-widest text-secondary text-sm">Hướng dẫn chọn kích thước</p>
        <h3 id="size-guide-title" className="font-headline-md text-headline-md text-primary">Thông Tin Size May Sẵn</h3></div>
      <button type="button" aria-label="Đóng" onClick={onClose} className="p-2"><Icon name="close" size={24} /></button>
    </div>

    {sizes.length > 0 && <>
      <p className="text-on-surface-variant leading-relaxed mt-2 mb-3">Các kích cỡ đang có cho mẫu áo này:</p>
      <div className="flex gap-2 flex-wrap mb-5">{sizes.map((size) => <span key={size} className={`px-4 py-2 ${size === suggestion ? 'bg-primary text-on-primary' : 'bg-surface-container'}`}>{size}</span>)}</div>
    </>}

    <div className="overflow-x-auto -mx-1 px-1">
      <table className="w-full min-w-[480px] text-sm border-collapse">
        <caption className="text-left text-on-surface-variant pb-2">Số đo cơ thể (cm), đo sát người</caption>
        <thead>
          <tr className="bg-surface-container text-on-surface">
            <th scope="col" className="text-left font-semibold px-3 py-2">Size</th>
            {COLUMNS.map(([key, label]) => <th key={key} scope="col" className="text-right font-semibold px-3 py-2">{label}</th>)}
          </tr>
        </thead>
        <tbody>
          {rows.map((size: StandardSize) => {
            const m = SIZE_CHART[size]!
            const active = size === suggestion
            return <tr key={size} className={`border-b border-outline-variant/40 ${active ? 'bg-secondary-container/60 font-semibold' : ''}`}>
              <th scope="row" className="text-left px-3 py-2 text-primary">{size}</th>
              {COLUMNS.map(([key]) => <td key={key} className="text-right px-3 py-2 tabular-nums">{range(m[key])}</td>)}
            </tr>
          })}
        </tbody>
      </table>
    </div>
    {offered.has('Freesize') && <p className="text-on-surface-variant text-sm mt-2">Freesize: dáng suông, vừa người khoảng 42–55 kg.</p>}

    <fieldset className="mt-6 bg-surface-container-low p-4">
      <legend className="font-semibold text-on-surface px-1">Gợi ý size theo số đo của bạn</legend>
      <div className="grid grid-cols-3 gap-3 mt-2">
        {(['bust', 'waist', 'hip'] as const).map((key) => <label key={key} className="text-sm text-on-surface-variant">
          {COLUMNS.find(([k]) => k === key)![1]} (cm)
          <input type="number" inputMode="decimal" min={50} max={150} value={body[key]} onChange={(event) => setBody({ ...body, [key]: event.target.value })} className="mt-1 w-full bg-surface px-3 py-2 border border-outline-variant focus:border-primary outline-none" />
        </label>)}
      </div>
      <p className="mt-3 text-sm min-h-5" aria-live="polite">
        {suggestion === null && 'Số đo nằm ngoài bảng size may sẵn — áo may đo riêng sẽ vừa hơn.'}
        {suggestion && available && <>Size phù hợp: <strong className="text-primary">{suggestion}</strong>. Nếu vòng nào nằm giữa hai size, hãy chọn size lớn hơn.</>}
        {suggestion && !available && <>Bạn hợp size <strong className="text-primary">{suggestion}</strong>, nhưng mẫu này chưa có size đó.</>}
      </p>
    </fieldset>

    <p className="text-on-surface-variant text-sm mt-4 leading-relaxed">Áo dài ôm sát nên chênh 1–2 cm cũng ảnh hưởng dáng áo. Xem <Link to="/dich-vu-may-do" onClick={onClose} className="underline text-primary">cách đo 8 số đo</Link> hoặc đặt may đo để áo vừa từng đường cong.</p>
    <button type="button" onClick={onClose} className="mt-6 bg-primary text-on-primary px-5 py-2">Đã hiểu</button>
  </Modal>
}
