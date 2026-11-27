import { useState } from 'react'
import { Link } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import Modal from '@/components/ui/Modal'
import { SIZE_CHART, STANDARD_SIZES, estimateSizeFromBody, isStandardSize, lengthFit, suggestSize, type StandardSize } from '@/features/catalog/sizes'

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
  const [body, setBody] = useState({ bust: '', waist: '', hip: '', height: '', weight: '' })
  const hasGirths = Boolean(body.bust && body.waist && body.hip)
  const estimated = !hasGirths && Boolean(body.height && body.weight)
  // Girths decide whether the bodice closes; height and weight only give an estimate.
  const suggestion = hasGirths
    ? suggestSize(Number(body.bust), Number(body.waist), Number(body.hip))
    : estimated ? estimateSizeFromBody(Number(body.height), Number(body.weight)) : undefined
  const available = suggestion ? offered.size === 0 || offered.has(suggestion) : false
  const length = suggestion && body.height ? lengthFit(suggestion, Number(body.height)) : null
  const fields = [['bust', 'Vòng ngực', 'cm'], ['waist', 'Vòng eo', 'cm'], ['hip', 'Vòng mông', 'cm'], ['height', 'Chiều cao', 'cm'], ['weight', 'Cân nặng', 'kg']] as const

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
      <p className="text-xs text-on-surface-variant mt-1">Nhập 3 vòng để chọn chính xác; thêm chiều cao để kiểm tra độ dài tà và tay. Chưa đo vòng thì nhập chiều cao + cân nặng để ước lượng.</p>
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-3 mt-3 items-end">
        {fields.map(([key, text, unit]) => <label key={key} className="flex flex-col gap-1 text-sm text-on-surface-variant">
          <span className="whitespace-nowrap">{text}</span>
          <span className="relative block">
            <input type="number" inputMode="decimal" min={key === 'weight' ? 25 : 50} max={key === 'weight' ? 150 : 200} value={body[key]} onChange={(event) => setBody({ ...body, [key]: event.target.value })}
              className="w-full bg-surface pl-3 pr-9 py-2 border border-outline-variant focus:border-primary outline-none tabular-nums [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none" />
            <span aria-hidden="true" className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs">{unit}</span>
          </span>
        </label>)}
      </div>
      <div className="mt-3 text-sm min-h-5 space-y-1" aria-live="polite">
        {suggestion === null && <p>{estimated ? 'Chiều cao, cân nặng nằm ngoài bảng size may sẵn' : 'Số đo nằm ngoài bảng size may sẵn'} — áo may đo riêng sẽ vừa hơn.</p>}
        {suggestion && available && <p>{estimated ? 'Size ước lượng' : 'Size phù hợp'}: <strong className="text-primary">{suggestion}</strong>.{' '}
          {estimated ? 'Đây chỉ là ước lượng theo chiều cao, cân nặng; hãy đo 3 vòng để chọn chính xác.' : 'Nếu vòng nào nằm giữa hai size, hãy chọn size lớn hơn.'}</p>}
        {suggestion && !available && <p>Bạn hợp size <strong className="text-primary">{suggestion}</strong>, nhưng mẫu này chưa có size đó.</p>}
        {suggestion && length === 'short' && <p className="text-secondary">Với chiều cao {body.height} cm, tà và tay áo size {suggestion} (dành cho {SIZE_CHART[suggestion]!.height.join('–')} cm) có thể hơi ngắn. Nên đặt may đo để áo vừa cả dáng lẫn độ dài.</p>}
        {suggestion && length === 'long' && <p className="text-secondary">Với chiều cao {body.height} cm, tà và tay áo size {suggestion} (dành cho {SIZE_CHART[suggestion]!.height.join('–')} cm) có thể hơi dài. Bạn có thể nhờ tiệm sửa lai, hoặc đặt may đo cho vừa.</p>}
      </div>
    </fieldset>

    <p className="text-on-surface-variant text-sm mt-4 leading-relaxed">Áo dài ôm sát nên chênh 1–2 cm cũng ảnh hưởng dáng áo. Xem <Link to="/dich-vu-may-do" onClick={onClose} className="underline text-primary">cách đo 8 số đo</Link> hoặc đặt may đo để áo vừa từng đường cong.</p>
    <button type="button" onClick={onClose} className="mt-6 bg-primary text-on-primary px-5 py-2">Đã hiểu</button>
  </Modal>
}
