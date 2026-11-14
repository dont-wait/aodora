import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react'
import { adminCatalog, AdminApiError, type Image } from '@/features/admin/catalog'
import {
  blankStory, blankFabric, blankCraft, editableContent, moveEntry, validateEditorialInput,
  type StoryContent, type FabricContent, type CraftContent, type ProductContentInput,
} from '@/features/admin/content'

const fieldClass = 'w-full border border-outline/40 bg-white text-on-surface p-2.5 rounded-sm'
const actionClass = 'bg-primary text-on-primary px-4 py-2 rounded-sm disabled:opacity-50'
const secondaryClass = 'border border-outline/40 px-3 py-2 rounded-sm disabled:opacity-40'

function Field({ label, children }: { label: string; children: ReactNode }) {
  return <label className="flex flex-col gap-1.5 text-sm font-medium">{label}{children}</label>
}
function PictureFields({ image, onChange }: { image: Image | null; onChange: (image: Image | null) => void }) {
  function change(src: string, alt: string) { onChange(src || alt ? { src, alt } : null) }
  return <div className="grid sm:grid-cols-2 gap-3">
    <Field label="URL ảnh minh họa (không bắt buộc)">
      <input className={fieldClass} value={image?.src ?? ''} onChange={(e) => change(e.target.value, image?.alt ?? '')}/>
    </Field>
    <Field label="Mô tả ảnh (alt)">
      <input className={fieldClass} value={image?.alt ?? ''} onChange={(e) => change(image?.src ?? '', e.target.value)}/>
    </Field>
  </div>
}
function RowActions({ index, count, move, remove }: { index: number; count: number; move: (to: number) => void; remove: () => void }) {
  return <div className="flex flex-wrap items-center gap-2 text-sm">
    <button type="button" className={secondaryClass} disabled={index === 0} onClick={() => move(index - 1)}>↑ Lên</button>
    <button type="button" className={secondaryClass} disabled={index === count - 1} onClick={() => move(index + 1)}>↓ Xuống</button>
    <button type="button" className="text-red-700 underline px-2" onClick={remove}>Xóa mục</button>
  </div>
}

export default function ProductContentEditor({ productId, onVersion, onForbidden }: {
  productId: string; onVersion: (version: number) => void; onForbidden: () => void
}) {
  const mounted = useRef(true)
  const [data, setData] = useState<ProductContentInput | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [revision, setRevision] = useState(0)

  useEffect(() => {
    mounted.current = true
    return () => { mounted.current = false }
  }, [])

  useEffect(() => {
    let current = true
    adminCatalog.content(productId)
      .then((record) => { if (current) { setData(editableContent(record)); setError('') } })
      .catch((err: unknown) => {
        if (!current) return
        if (err instanceof AdminApiError && (err.status === 401 || err.status === 403)) onForbidden()
        else setError(err instanceof Error ? err.message : 'Không thể tải nội dung sản phẩm')
      })
      .finally(() => { if (current) setLoading(false) })
    return () => { current = false }
    // onForbidden is a stable callback from the page, and revision triggers a fresh read.
  }, [productId, revision, onForbidden])

  function updateStory(patch: Partial<StoryContent>) { setData((s) => s && { ...s, story: { ...(s.story ?? blankStory()), ...patch } }) }
  function updateFabric(patch: Partial<FabricContent>) { setData((s) => s && { ...s, fabric: { ...(s.fabric ?? blankFabric()), ...patch } }) }
  function updateCraft(patch: Partial<CraftContent>) { setData((s) => s && { ...s, craft: { ...(s.craft ?? blankCraft()), ...patch } }) }
  function reload() { setLoading(true); setError(''); setSuccess(''); setRevision((n) => n + 1) }
  async function save(event: FormEvent) {
    event.preventDefault()
    if (!data) return
    setError('');setSuccess('');setSaving(true)
    try {
      validateEditorialInput(data)
      const saved = await adminCatalog.updateContent(productId, data)
      if (!mounted.current) return
      setData(editableContent(saved))
      onVersion(saved.version)
      setSuccess('Đã lưu câu chuyện, chất liệu và quy trình chế tác vào MongoDB')
    } catch (err) {
      if (!mounted.current) return
      if (err instanceof AdminApiError && (err.status === 401 || err.status === 403)) onForbidden()
      else setError(err instanceof Error ? err.message : 'Không thể lưu nội dung')
    } finally { if (mounted.current) setSaving(false) }
  }

  return <section className="bg-surface p-5 space-y-5" aria-label="Biên tập nội dung mẫu áo">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div><h2 className="font-title-editorial text-2xl text-primary">Câu chuyện, chất liệu & chế tác</h2>
        <p className="text-sm text-secondary">Nội dung riêng của mẫu áo này. Không thay đổi giá, SKU hoặc tồn kho.</p></div>
      <button type="button" className={secondaryClass} onClick={reload} disabled={saving}>Tải lại nội dung</button>
    </div>
    {loading && <p role="status">Đang tải nội dung sản phẩm...</p>}
    {!loading && !data && <p role="alert">{error || 'Không thể đọc nội dung.'}</p>}
    {data && !loading && <form onSubmit={(e) => void save(e)} className="space-y-6">
      <fieldset className="border border-outline/30 p-4 space-y-4 min-w-0">
        <legend className="px-2 font-bold text-primary">Câu chuyện thiết kế</legend>
        {!data.story ? <button type="button" className={secondaryClass} onClick={() => setData({ ...data, story: blankStory() })}>+ Thêm câu chuyện</button> : <>
          <div className="flex justify-end"><button type="button" className="underline text-red-700" onClick={() => setData({ ...data, story: null })}>Xóa câu chuyện</button></div>
          <div className="grid md:grid-cols-2 gap-3">
            <Field label="Dòng dẫn (eyebrow)"><input className={fieldClass} maxLength={160} value={data.story.eyebrow} onChange={(e) => updateStory({ eyebrow: e.target.value })}/></Field>
            <Field label="Tiêu đề"><input className={fieldClass} maxLength={250} value={data.story.title} onChange={(e) => updateStory({ title: e.target.value })}/></Field>
          </div>
          <PictureFields image={data.story.image} onChange={(image) => updateStory({ image })}/>
          <h3 className="font-semibold">Các đoạn kể (hiển thị theo thứ tự)</h3>
          {data.story.paragraphs.map((p, i) => <div key={i} className="border border-outline/30 p-3 space-y-2">
            <Field label={`Đoạn ${i + 1}`}><textarea rows={4} className={fieldClass} value={p} maxLength={3000} onChange={(e) => updateStory({ paragraphs: data.story!.paragraphs.map((v, j) => j === i ? e.target.value : v) })}/></Field>
            <RowActions index={i} count={data.story!.paragraphs.length} move={(to) => updateStory({ paragraphs: moveEntry(data.story!.paragraphs, i, to) })} remove={() => updateStory({ paragraphs: data.story!.paragraphs.filter((_, j) => i !== j) })}/>
          </div>)}
          <button type="button" className={secondaryClass} onClick={() => updateStory({ paragraphs: [...data.story!.paragraphs, ''] })}>+ Thêm đoạn</button>
        </>}
      </fieldset>
      <fieldset className="border border-outline/30 p-4 space-y-4 min-w-0">
        <legend className="px-2 font-bold text-primary">Chất liệu & bảo quản</legend>
        {!data.fabric ? <button type="button" className={secondaryClass} onClick={() => setData({ ...data, fabric: blankFabric() })}>+ Thêm chất liệu</button> : <>
          <div className="flex justify-end"><button type="button" className="underline text-red-700" onClick={() => setData({ ...data, fabric: null })}>Xóa phần chất liệu</button></div>
          <div className="grid md:grid-cols-2 gap-3">
            <Field label="Dòng dẫn"><input className={fieldClass} maxLength={160} value={data.fabric.eyebrow} onChange={(e) => updateFabric({ eyebrow: e.target.value })}/></Field>
            <Field label="Tiêu đề"><input className={fieldClass} maxLength={250} value={data.fabric.title} onChange={(e) => updateFabric({ title: e.target.value })}/></Field>
          </div>
          <Field label="Mô tả chất liệu"><textarea rows={3} className={fieldClass} maxLength={4000} value={data.fabric.description} onChange={(e) => updateFabric({ description: e.target.value })}/></Field>
          <PictureFields image={data.fabric.image} onChange={(image) => updateFabric({ image })}/>
          <h3 className="font-semibold">Hướng dẫn bảo quản</h3>
          {data.fabric.care.map((item, i) => <div key={i} className="border border-outline/30 p-3 space-y-2">
            <div className="grid sm:grid-cols-2 gap-3">
              <Field label={`Mục ${i+1} · Tiêu đề`}><input className={fieldClass} maxLength={200} value={item.title} onChange={(e) => updateFabric({ care: data.fabric!.care.map((v,j) => i === j ? { ...v, title: e.target.value } : v) })}/></Field>
              <Field label="Biểu tượng (không bắt buộc)"><input className={fieldClass} maxLength={80} value={item.icon ?? ''} onChange={(e) => updateFabric({ care: data.fabric!.care.map((v,j) => i === j ? { ...v, icon: e.target.value } : v) })}/></Field>
            </div>
            <Field label="Nội dung hướng dẫn"><textarea rows={2} className={fieldClass} maxLength={2000} value={item.description} onChange={(e) => updateFabric({ care: data.fabric!.care.map((v,j) => i === j ? { ...v, description: e.target.value } : v) })}/></Field>
            <RowActions index={i} count={data.fabric!.care.length} move={(to) => updateFabric({ care: moveEntry(data.fabric!.care, i, to) })} remove={() => updateFabric({ care: data.fabric!.care.filter((_,j) => i !== j) })}/>
          </div>)}
          <button type="button" className={secondaryClass} onClick={() => updateFabric({ care: [...data.fabric!.care, { title: '', description: '', icon: '' }] })}>+ Thêm hướng dẫn</button>
        </>}
      </fieldset>
      <fieldset className="border border-outline/30 p-4 space-y-4 min-w-0">
        <legend className="px-2 font-bold text-primary">Quy trình chế tác</legend>
        {!data.craft ? <button type="button" className={secondaryClass} onClick={() => setData({ ...data, craft: blankCraft() })}>+ Thêm quy trình</button> : <>
          <div className="flex justify-end"><button type="button" className="underline text-red-700" onClick={() => setData({ ...data, craft: null })}>Xóa phần chế tác</button></div>
          <div className="grid md:grid-cols-2 gap-3">
            <Field label="Dòng dẫn"><input className={fieldClass} maxLength={160} value={data.craft.eyebrow} onChange={(e) => updateCraft({ eyebrow: e.target.value })}/></Field>
            <Field label="Tiêu đề"><input className={fieldClass} maxLength={250} value={data.craft.title} onChange={(e) => updateCraft({ title: e.target.value })}/></Field>
          </div>
          <Field label="Mô tả quy trình"><textarea rows={3} className={fieldClass} maxLength={4000} value={data.craft.description} onChange={(e) => updateCraft({ description: e.target.value })}/></Field>
          <PictureFields image={data.craft.image} onChange={(image) => updateCraft({ image })}/>
          <h3 className="font-semibold">Các bước chế tác</h3>
          {data.craft.steps.map((item, i) => <div key={i} className="border border-outline/30 p-3 space-y-2">
            <Field label={`Bước ${i+1} · Tên bước`}><input className={fieldClass} maxLength={200} value={item.title} onChange={(e) => updateCraft({ steps: data.craft!.steps.map((v,j) => i === j ? { ...v, title: e.target.value } : v) })}/></Field>
            <Field label="Mô tả bước"><textarea rows={2} className={fieldClass} maxLength={2000} value={item.description} onChange={(e) => updateCraft({ steps: data.craft!.steps.map((v,j) => i === j ? { ...v, description: e.target.value } : v) })}/></Field>
            <RowActions index={i} count={data.craft!.steps.length} move={(to) => updateCraft({ steps: moveEntry(data.craft!.steps, i, to) })} remove={() => updateCraft({ steps: data.craft!.steps.filter((_,j) => i !== j) })}/>
          </div>)}
          <button type="button" className={secondaryClass} onClick={() => updateCraft({ steps: [...data.craft!.steps, { title: '', description: '' }] })}>+ Thêm bước</button>
        </>}
      </fieldset>
      <div className="flex flex-wrap items-center gap-3">
        <button type="submit" className={actionClass} disabled={saving}>{saving ? 'Đang lưu...' : 'Lưu nội dung mẫu áo'}</button>
        <span className="text-xs text-secondary">Nếu dữ liệu đã thay đổi ở tab khác, API trả 409. Tải lại trước khi sửa tiếp.</span>
      </div>
      {error && <p role="alert" className="text-red-700">{error}</p>}
      {success && <p role="status" className="text-green-700">{success}</p>}
    </form>}
  </section>
}
