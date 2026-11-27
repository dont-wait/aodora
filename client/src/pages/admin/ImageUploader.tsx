import { useCallback, useEffect, useId, useRef, useState, type DragEvent } from 'react'
import type { Image } from '@/features/admin/catalog'
import { checkImageDimensions, checkImageFile, defaultAlt, moveItem, readImageSize, uploadImage, uploadedImage, type UploadPurpose } from '@/features/admin/upload'
import { fieldId, type Errors } from '@/features/admin/validation'
import { Button } from './ui'
import { inputClass } from './uiStyles'

interface Props {
  purpose: UploadPurpose
  items: Image[]
  onChange: (items: Image[]) => void
  /** Several images (gallery) or exactly one (main image, category, accessory, content). */
  multiple?: boolean
  max?: number
  /** Text used to pre-fill the alt of a new image: "<altBase> — ảnh 2". */
  altBase: string
  /** Error path prefix: "image" -> image.alt / image.src; "gallery" -> gallery[i].alt / gallery[i].src. */
  pathPrefix: string
  errors?: Errors
  label: string
  hint?: string
  /** Gallery only: make item i the main image. */
  onPromote?: (index: number) => void
  disabled?: boolean
}

type Pending = { id: string; name: string; preview: string; progress: number; error: string }

const itemPath = (prefix: string, multiple: boolean, i: number, leaf: 'alt' | 'src') => (multiple ? prefix + '[' + i + '].' + leaf : prefix + '.' + leaf)

export default function ImageUploader({ purpose, items, onChange, multiple = false, max = 12, altBase, pathPrefix, errors = {}, label, hint, onPromote, disabled = false }: Props) {
  const inputId = useId()
  const fileInput = useRef<HTMLInputElement>(null)
  const [pending, setPending] = useState<Pending[]>([])
  const [dragging, setDragging] = useState(false)
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const itemsRef = useRef(items)
  const altBaseRef = useRef(altBase)
  const onChangeRef = useRef(onChange)
  const controllers = useRef(new Map<string, AbortController>())
  useEffect(() => {
    itemsRef.current = items
    altBaseRef.current = altBase
    onChangeRef.current = onChange
  })
  useEffect(() => {
    const live = controllers.current
    return () => { live.forEach((c) => c.abort()); live.clear() }
  }, [])

  const patchPending = (id: string, patch: Partial<Pending>) => setPending((list) => list.map((p) => (p.id === id ? { ...p, ...patch } : p)))
  const dropPending = (id: string) => setPending((list) => {
    const gone = list.find((p) => p.id === id)
    if (gone) URL.revokeObjectURL(gone.preview)
    return list.filter((p) => p.id !== id)
  })

  const startUpload = useCallback(async (file: File) => {
    const id = crypto.randomUUID()
    const preview = URL.createObjectURL(file)
    setPending((list) => [...list, { id, name: file.name, preview, progress: 0, error: '' }])
    const fail = (message: string) => patchPending(id, { error: message })
    const problem = checkImageFile(file)
    if (problem) return fail(problem)
    try {
      const { width, height } = await readImageSize(file)
      const sizeProblem = checkImageDimensions(width, height)
      if (sizeProblem) return fail(sizeProblem)
    } catch (err) {
      return fail(err instanceof Error ? err.message : 'Không đọc được ảnh.')
    }
    const controller = new AbortController()
    controllers.current.set(id, controller)
    try {
      const result = await uploadImage(file, purpose, { signal: controller.signal, onProgress: (fraction) => patchPending(id, { progress: fraction }) })
      const current = itemsRef.current
      const next = multiple ? [...current, uploadedImage(result, defaultAlt(altBaseRef.current, current.length + 1))] : [uploadedImage(result, defaultAlt(altBaseRef.current, null))]
      itemsRef.current = next
      onChangeRef.current(next)
      dropPending(id)
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') dropPending(id)
      else fail(err instanceof Error ? err.message : 'Không thể tải ảnh lên.')
    } finally {
      controllers.current.delete(id)
    }
  }, [multiple, purpose])

  function accept(fileList: FileList | File[] | null) {
    if (!fileList || disabled) return
    let files = Array.from(fileList)
    if (!multiple) files = files.slice(0, 1)
    else {
      const room = Math.max(0, max - itemsRef.current.length - pending.filter((p) => !p.error).length)
      files = files.slice(0, room)
    }
    files.forEach((file) => void startUpload(file))
    if (fileInput.current) fileInput.current.value = ''
  }
  function update(next: Image[]) { itemsRef.current = next; onChange(next) }
  const onDrop = (event: DragEvent) => { event.preventDefault(); setDragging(false); accept(event.dataTransfer.files) }
  const full = multiple && items.length + pending.filter((p) => !p.error).length >= max

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="text-[13px] font-medium text-on-surface-variant">{label}</span>
        {multiple && <span className="text-xs text-on-surface-variant">{items.length}/{max} ảnh</span>}
      </div>
      {items.length > 0 && (
        <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3" aria-label={label}>
          {items.map((item, i) => {
            const altPath = itemPath(pathPrefix, multiple, i, 'alt')
            const srcPath = itemPath(pathPrefix, multiple, i, 'src')
            const altError = errors[altPath]
            const srcError = errors[srcPath]
            return (
              <li
                key={item.src + i}
                draggable={multiple && !disabled}
                onDragStart={() => setDragIndex(i)}
                onDragOver={(e) => { if (dragIndex !== null) e.preventDefault() }}
                onDrop={(e) => { if (dragIndex !== null) { e.preventDefault(); update(moveItem(items, dragIndex, i)); setDragIndex(null) } }}
                onDragEnd={() => setDragIndex(null)}
                className={`space-y-2 rounded-lg border bg-surface-container-lowest p-2 ${dragIndex === i ? 'border-primary opacity-60' : 'border-outline-variant/60'}`}
              >
                <div className="relative aspect-[3/4] overflow-hidden rounded-md bg-surface-container">
                  <img src={item.src} alt={item.alt} className="h-full w-full object-cover" onError={(e) => { e.currentTarget.style.visibility = 'hidden' }} />
                  {multiple && <span className="absolute left-1 top-1 rounded bg-black/60 px-1.5 text-xs text-white">{i + 1}</span>}
                </div>
                <input
                  id={fieldId(altPath)}
                  className={inputClass}
                  aria-label={'Mô tả ảnh ' + (multiple ? i + 1 : '')}
                  aria-invalid={altError ? true : undefined}
                  aria-describedby={altError ? fieldId(altPath) + '-error' : undefined}
                  placeholder="Mô tả ảnh (alt)"
                  value={item.alt}
                  disabled={disabled}
                  onChange={(e) => update(items.map((x, j) => (j === i ? { ...x, alt: e.target.value } : x)))}
                />
                {altError && <p id={fieldId(altPath) + '-error'} role="alert" className="text-xs text-red-700">{altError}</p>}
                {srcError && <p id={fieldId(srcPath) + '-error'} role="alert" className="text-xs text-red-700">{srcError}</p>}
                <div className="flex flex-wrap gap-1">
                  {multiple && (
                    <>
                      <Button variant="ghost" className="!h-8 !px-2" aria-label={'Chuyển ảnh ' + (i + 1) + ' lên trước'} disabled={disabled || i === 0} onClick={() => update(moveItem(items, i, i - 1))}>←</Button>
                      <Button variant="ghost" className="!h-8 !px-2" aria-label={'Chuyển ảnh ' + (i + 1) + ' ra sau'} disabled={disabled || i === items.length - 1} onClick={() => update(moveItem(items, i, i + 1))}>→</Button>
                      {onPromote && <Button variant="secondary" className="!h-8 !px-2 text-xs" disabled={disabled} onClick={() => onPromote(i)}>Đặt làm ảnh chính</Button>}
                    </>
                  )}
                  <Button variant="danger" className="!h-8 !px-2 text-xs" disabled={disabled} onClick={() => update(items.filter((_, j) => j !== i))}>Xóa ảnh</Button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
      {pending.length > 0 && (
        <ul className="space-y-2" aria-label="Ảnh đang tải lên">
          {pending.map((p) => (
            <li key={p.id} className="flex items-center gap-3 rounded-lg border border-outline-variant/60 bg-surface-container-lowest p-2">
              <img src={p.preview} alt="" className="h-14 w-11 rounded object-cover" />
              <div className="min-w-0 flex-1 space-y-1">
                <p className="truncate text-[13px]">{p.name}</p>
                {p.error ? (
                  <p role="alert" className="text-xs text-red-700">{p.error}</p>
                ) : (
                  <div role="progressbar" aria-label={'Đang tải ' + p.name} aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(p.progress * 100)} className="h-2 overflow-hidden rounded bg-surface-container-high">
                    <div className="h-full bg-primary transition-[width]" style={{ width: Math.round(p.progress * 100) + '%' }} />
                  </div>
                )}
              </div>
              {p.error ? (
                <Button variant="ghost" className="!h-8 !px-2 text-xs" onClick={() => dropPending(p.id)}>Đóng</Button>
              ) : (
                <Button variant="ghost" className="!h-8 !px-2 text-xs" onClick={() => controllers.current.get(p.id)?.abort()}>Hủy</Button>
              )}
            </li>
          ))}
        </ul>
      )}
      {!full && (
        <div
          onDragOver={(e) => { if (dragIndex === null) { e.preventDefault(); setDragging(true) } }}
          onDragLeave={() => setDragging(false)}
          onDrop={onDrop}
          className={`flex flex-wrap items-center justify-center gap-3 rounded-lg border-2 border-dashed p-4 text-center ${dragging ? 'border-primary bg-surface-container' : 'border-outline-variant bg-surface-container-low'} ${disabled ? 'opacity-60' : ''}`}
        >
          <input ref={fileInput} id={inputId} type="file" accept="image/jpeg,image/png,image/webp" multiple={multiple} disabled={disabled} className="sr-only" onChange={(e) => accept(e.target.files)} />
          <label htmlFor={inputId} className="inline-flex h-10 cursor-pointer items-center rounded-lg border border-primary bg-surface-container-lowest px-4 text-sm font-medium text-primary hover:bg-surface-container focus-within:outline focus-within:outline-2">
            {items.length > 0 && !multiple ? 'Đổi ảnh' : multiple ? 'Thêm ảnh' : 'Chọn ảnh'}
          </label>
          <span className="text-[13px] text-on-surface-variant">hoặc kéo thả {multiple ? 'nhiều ảnh' : 'ảnh'} vào đây</span>
        </div>
      )}
      <p className="text-xs text-on-surface-variant">{hint ?? 'JPG, PNG hoặc WebP, tối đa 10 MB, cạnh ngắn từ 400 px.'}</p>
      {errors[pathPrefix] && <p role="alert" className="text-xs text-red-700">{errors[pathPrefix]}</p>}
    </div>
  )
}
