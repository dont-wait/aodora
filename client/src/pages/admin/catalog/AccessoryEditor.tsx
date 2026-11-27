import type { ProductInput } from '@/features/admin/catalog'
import { moveItem } from '@/features/admin/upload'
import { fieldId, type Errors } from '@/features/admin/validation'
import ImageUploader from '../ImageUploader'
import { AutoTextarea, Button, Field, MoneyInput } from '../ui'
import { inputClass } from '../uiStyles'

type Accessory = ProductInput['accessories'][number]

const newId = () => 'pk-' + crypto.randomUUID().slice(0, 8)

export default function AccessoryEditor({ accessories, onChange, errors, productName }: { accessories: Accessory[]; onChange: (next: Accessory[]) => void; errors: Errors; productName: string }) {
  const patch = (i: number, change: Partial<Accessory>) => onChange(accessories.map((a, j) => (j === i ? { ...a, ...change } : a)))
  return (
    <div className="space-y-4">
      {accessories.length === 0 && <p className="text-sm text-on-surface-variant">Chưa có phụ kiện đi kèm.</p>}
      {accessories.map((a, i) => {
        const p = 'accessories[' + i + ']'
        return (
          <fieldset key={a.id} className="space-y-3 rounded-lg border border-outline-variant/60 p-4">
            <legend className="px-2 text-[13px] font-semibold">Phụ kiện {i + 1}</legend>
            <div className="grid gap-3 md:grid-cols-2">
              <Field label="Tên phụ kiện" path={p + '.name'} error={errors[p + '.name']} required>
                <input className={inputClass} maxLength={100} value={a.name} onChange={(e) => patch(i, { name: e.target.value })} />
              </Field>
              <Field label="Giá (VND)" path={p + '.price'} error={errors[p + '.price']} hint="Nhập 0 nếu tặng kèm">
                <MoneyInput value={a.price} onChange={(v) => patch(i, { price: v ?? 0 })} />
              </Field>
            </div>
            <Field label="Mô tả" path={p + '.description'} error={errors[p + '.description']} hint={a.description.length + '/500 ký tự'}>
              <AutoTextarea minRows={2} value={a.description} onChange={(e) => patch(i, { description: e.target.value })} />
            </Field>
            <Field label="Nhãn nhỏ (không bắt buộc)" path={p + '.tag'} error={errors[p + '.tag']} hint="Ví dụ: Bán chạy, Mới">
              <input className={inputClass} maxLength={30} value={a.tag ?? ''} onChange={(e) => patch(i, { tag: e.target.value })} />
            </Field>
            <ImageUploader
              purpose="accessory"
              label="Ảnh phụ kiện"
              items={a.image ? [a.image] : []}
              onChange={(items) => patch(i, { image: items[0] ?? null })}
              altBase={a.name || productName}
              pathPrefix={p + '.image'}
              errors={errors}
            />
            <div className="flex flex-wrap gap-2" id={fieldId(p)}>
              <Button variant="ghost" aria-label={'Chuyển phụ kiện ' + (i + 1) + ' lên'} disabled={i === 0} onClick={() => onChange(moveItem(accessories, i, i - 1))}>↑ Lên</Button>
              <Button variant="ghost" aria-label={'Chuyển phụ kiện ' + (i + 1) + ' xuống'} disabled={i === accessories.length - 1} onClick={() => onChange(moveItem(accessories, i, i + 1))}>↓ Xuống</Button>
              <Button variant="danger" onClick={() => onChange(accessories.filter((_, j) => j !== i))}>Xóa phụ kiện</Button>
            </div>
          </fieldset>
        )
      })}
      {errors.accessories && <p role="alert" className="text-xs text-red-700">{errors.accessories}</p>}
      <Button variant="secondary" disabled={accessories.length >= 20} onClick={() => onChange([...accessories, { id: newId(), name: '', description: '', price: 0 }])}>+ Thêm phụ kiện</Button>
    </div>
  )
}
