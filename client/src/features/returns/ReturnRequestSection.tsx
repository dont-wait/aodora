import { useCallback, useEffect, useRef, useState } from 'react'
import { formatMoney } from '@/features/catalog/prices'
import { formatOrderDate } from '@/features/orders/presentation'
import type { Order } from '@/features/orders/types'
import { SkeletonBlock, SkeletonStatus } from '@/components/ui/Skeleton'
import {
  EVIDENCE_ACCEPT, MAX_EVIDENCE, RETURN_REASONS, RETURN_STEPS, draftBody, evidenceKindOf, evidenceProblem, reasonLabels, returnStatusLabels, returnsApi, timeLeft, validateReturnDraft,
  ReturnApiError, type Eligibility, type Evidence, type ReturnDraft, type ReturnReason, type ReturnRequest,
} from './returns'

const box = 'border border-outline-variant p-5 sm:p-6 space-y-4'
const inputClass = 'w-full border border-outline-variant bg-surface p-3'

/** Return-request block on the account order page: progress of past requests and the 3-day request form. */
export default function ReturnRequestSection({ order }: { order: Order }) {
  const [state, setState] = useState<{ kind: 'loading' } | { kind: 'ready'; data: Eligibility } | { kind: 'error'; message: string }>({ kind: 'loading' })
  const [open, setOpen] = useState(false)
  const load = useCallback(() => {
    returnsApi.eligibility(order.id).then(
      (data) => setState({ kind: 'ready', data }),
      (error: unknown) => setState({ kind: 'error', message: error instanceof Error ? error.message : 'Không tải được thông tin hoàn hàng.' }),
    )
  }, [order.id])
  useEffect(load, [load])

  if (order.status !== 'delivered') return null
  if (state.kind === 'loading') return <section className={box}><SkeletonStatus label="Đang tải thông tin hoàn hàng" className="space-y-3"><SkeletonBlock className="h-6 w-1/3" /><SkeletonBlock className="h-4 w-full" /><SkeletonBlock className="h-10 w-48" /></SkeletonStatus></section>
  if (state.kind === 'error')
    return (
      <section className={box}>
        <p role="alert" className="text-error">{state.message}</p>
        <button type="button" className="underline text-primary" onClick={load}>Thử lại</button>
      </section>
    )
  const { data } = state
  const left = data.deadline ? timeLeft(data.deadline) : ''
  return (
    <section className={box} aria-labelledby="return-heading">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h2 id="return-heading" className="font-headline-md text-2xl text-primary">Hoàn hàng</h2>
        {data.deadline && (
          <p className="text-sm text-on-surface-variant">
            Hạn gửi yêu cầu: <time dateTime={data.deadline}>{formatOrderDate(data.deadline)}</time>
            {left && <strong className="text-primary"> ({left})</strong>}
          </p>
        )}
      </div>
      {data.requests.map((r) => <RequestCard key={r.id} request={r} onChanged={load} />)}
      {data.eligible ? (
        open ? (
          <ReturnForm order={order} onCancel={() => setOpen(false)} onDone={() => { setOpen(false); load() }} />
        ) : (
          <div className="space-y-3">
            <p className="text-sm text-on-surface-variant">
              Trong vòng 3 ngày kể từ khi nhận áo, bạn có thể gửi yêu cầu hoàn hàng kèm <strong>ảnh hoặc video minh chứng</strong>. Cửa hàng sẽ xem và phản hồi sớm nhất.
            </p>
            <button type="button" className="bg-primary text-on-primary px-6 py-3" onClick={() => setOpen(true)}>Yêu cầu hoàn hàng</button>
          </div>
        )
      ) : (
        data.reason && data.requests.length === 0 && <p className="text-sm text-on-surface-variant">{data.reason}</p>
      )}
    </section>
  )
}

function RequestCard({ request, onChanged }: { request: ReturnRequest; onChanged: () => void }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const ended = request.status === 'rejected' || request.status === 'cancelled'
  const reached = RETURN_STEPS.indexOf(request.status)
  async function cancel() {
    if (!window.confirm('Huỷ yêu cầu hoàn hàng này? Bạn vẫn có thể gửi lại nếu còn trong hạn 3 ngày.')) return
    setBusy(true)
    setError('')
    try {
      await returnsApi.cancel(request.id)
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Không huỷ được yêu cầu.')
    } finally {
      setBusy(false)
    }
  }
  return (
    <article className="bg-surface-container-low p-4 space-y-3">
      <div className="flex flex-wrap justify-between gap-2">
        <p className="font-semibold text-primary">{returnStatusLabels[request.status]}</p>
        <p className="text-sm text-on-surface-variant">Gửi lúc <time dateTime={request.created_at}>{formatOrderDate(request.created_at)}</time></p>
      </div>
      {!ended && (
        <ol className="grid grid-cols-5 gap-1 text-[11px] sm:text-xs" aria-label="Tiến trình hoàn hàng">
          {RETURN_STEPS.map((step, i) => (
            <li key={step} className="space-y-1" aria-current={i === reached ? 'step' : undefined}>
              <span className={`block h-1.5 ${i <= reached ? 'bg-primary' : 'bg-outline-variant'}`} />
              <span className={i <= reached ? 'text-primary' : 'text-on-surface-variant'}>{returnStatusLabels[step]}</span>
            </li>
          ))}
        </ol>
      )}
      <ul className="text-sm space-y-1">
        {request.items.map((it) => (
          <li key={it.sku}>{it.product_name} {[it.size, it.color].filter(Boolean).join(' · ')} × {it.quantity}</li>
        ))}
      </ul>
      <p className="text-sm">Lý do: {reasonLabels[request.reason]}{request.note ? ` — ${request.note}` : ''}</p>
      <p className="text-sm">Số tiền dự kiến hoàn: <strong>{formatMoney(request.refund_amount)} ₫</strong></p>
      {request.admin_note && (
        <p className={`text-sm p-3 ${request.status === 'rejected' ? 'bg-error-container text-on-error-container' : 'bg-surface'}`}>Cửa hàng: {request.admin_note}</p>
      )}
      <EvidenceStrip evidence={request.evidence} />
      {error && <p role="alert" className="text-error text-sm">{error}</p>}
      {request.status === 'requested' && (
        <button type="button" disabled={busy} className="border border-outline text-on-surface-variant px-4 py-2 text-sm disabled:opacity-40" onClick={() => void cancel()}>
          {busy ? 'Đang huỷ…' : 'Huỷ yêu cầu'}
        </button>
      )}
    </article>
  )
}

/** Plays a return video; when the browser decodes only the sound (e.g. HEVC), says so and offers the file. */
function EvidenceVideo({ url, large }: { url: string; large: boolean }) {
  const [noPicture, setNoPicture] = useState(false)
  const [failed, setFailed] = useState(false)
  if (!large)
    return (
      <a href={url} target="_blank" rel="noopener noreferrer" className="flex h-full w-full items-center justify-center bg-black/70 text-white text-xs" aria-label="Mở video minh chứng">
        ▶ Video
      </a>
    )
  return (
    <div className="space-y-1">
      <video
        src={url}
        controls
        playsInline
        preload="metadata"
        className="w-full max-h-[70vh] bg-black"
        aria-label="Video minh chứng"
        onLoadedMetadata={(e) => setNoPicture(e.currentTarget.videoWidth === 0)}
        onError={() => setFailed(true)}
      />
      {(noPicture || failed) && (
        <p className="text-xs text-on-surface-variant">
          Trình duyệt này không hiển thị được hình của video (thường là video HEVC quay từ điện thoại).{' '}
          <a href={url} download className="underline text-primary">Tải video về</a> để xem bằng trình phát trên máy.
        </p>
      )}
    </div>
  )
}

export function EvidenceStrip({ evidence, large = false }: { evidence: Evidence[]; large?: boolean }) {
  const size = large ? 'w-40 h-40 sm:w-48 sm:h-48' : 'w-20 h-20'
  const videos = evidence.filter((e) => e.kind === 'video')
  const images = evidence.filter((e) => e.kind !== 'video')
  if (large)
    return (
      <div className="space-y-3">
        {videos.map((e) => (
          <div key={e.url} className="max-w-2xl"><EvidenceVideo url={e.url} large /></div>
        ))}
        {images.length > 0 && (
          <ul className="flex flex-wrap gap-2" aria-label="Ảnh minh chứng">
            {images.map((e) => (
              <li key={e.url} className={`${size} bg-surface-container overflow-hidden border border-outline-variant`}>
                <a href={e.url} target="_blank" rel="noopener noreferrer"><img src={e.url} alt="Ảnh minh chứng" className="w-full h-full object-cover" loading="lazy" /></a>
              </li>
            ))}
          </ul>
        )}
      </div>
    )
  return (
    <ul className="flex flex-wrap gap-2" aria-label="Minh chứng">
      {evidence.map((e) => (
        <li key={e.url} className={`${size} bg-surface-container overflow-hidden border border-outline-variant`}>
          {e.kind === 'video' ? (
            <EvidenceVideo url={e.url} large={false} />
          ) : (
            <a href={e.url} target="_blank" rel="noopener noreferrer">
              <img src={e.url} alt="Ảnh minh chứng" className="w-full h-full object-cover" loading="lazy" />
            </a>
          )}
        </li>
      ))}
    </ul>
  )
}

type Uploading = { id: string; name: string; kind: 'image' | 'video'; preview: string; progress: 'uploading' | 'done' | 'error'; evidence?: Evidence; error?: string }

function ReturnForm({ order, onCancel, onDone }: { order: Order; onCancel: () => void; onDone: () => void }) {
  const [quantities, setQuantities] = useState<Record<string, number>>(() => (order.items.length === 1 ? { [order.items[0].sku]: order.items[0].quantity } : {}))
  const [reason, setReason] = useState<ReturnReason | ''>('')
  const [note, setNote] = useState('')
  const [files, setFiles] = useState<Uploading[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [fileError, setFileError] = useState('')
  const [sending, setSending] = useState(false)
  const input = useRef<HTMLInputElement>(null)
  const previews = useRef<string[]>([])
  useEffect(() => () => previews.current.forEach((u) => URL.revokeObjectURL(u)), [])

  const evidence = files.flatMap((f) => (f.evidence ? [f.evidence] : []))
  const uploading = files.some((f) => f.progress === 'uploading')
  const refund = order.items.reduce((sum, it) => sum + Number(it.unit_price) * (quantities[it.sku] ?? 0), 0)

  function pick(list: FileList | null) {
    if (!list) return
    setFileError('')
    const attached: { kind: 'image' | 'video' }[] = files.filter((f) => f.progress !== 'error').map((f) => ({ kind: f.kind }))
    const problems: string[] = []
    for (const file of Array.from(list)) {
      const problem = evidenceProblem(file, attached)
      if (problem) { problems.push(problem); continue }
      const kind = evidenceKindOf(file) as 'image' | 'video'
      attached.push({ kind })
      const preview = URL.createObjectURL(file)
      previews.current.push(preview)
      const id = `${file.name}-${file.size}-${Math.random().toString(36).slice(2)}`
      setFiles((prev) => [...prev, { id, name: file.name, kind, preview, progress: 'uploading' }])
      returnsApi.uploadEvidence(file).then(
        (ev) => setFiles((prev) => prev.map((f) => (f.id === id ? { ...f, progress: 'done', evidence: ev } : f))),
        (e: unknown) => setFiles((prev) => prev.map((f) => (f.id === id ? { ...f, progress: 'error', error: e instanceof Error ? e.message : 'Tải lên thất bại' } : f))),
      )
    }
    if (problems.length) setFileError(problems.join(' '))
    if (input.current) input.current.value = ''
  }

  async function submit() {
    const draft: ReturnDraft = { quantities, reason, note, evidence }
    const found = validateReturnDraft(draft)
    setErrors(found)
    if (Object.keys(found).length || uploading) return
    setSending(true)
    try {
      await returnsApi.create(draftBody(order.id, draft))
      onDone()
    } catch (e) {
      if (e instanceof ReturnApiError && e.field) setErrors({ [e.field.split('[')[0]]: e.message })
      else setErrors({ form: e instanceof Error ? e.message : 'Không gửi được yêu cầu.' })
    } finally {
      setSending(false)
    }
  }

  return (
    <form className="space-y-5" onSubmit={(e) => { e.preventDefault(); void submit() }} noValidate>
      <fieldset className="space-y-2">
        <legend className="font-semibold mb-2">1. Áo cần hoàn</legend>
        {order.items.map((it) => (
          <div key={it.sku} className="flex flex-wrap items-center justify-between gap-3 border border-outline-variant p-3">
            <label className="flex items-center gap-3 min-w-0">
              <input type="checkbox" checked={(quantities[it.sku] ?? 0) > 0} onChange={(e) => setQuantities((q) => ({ ...q, [it.sku]: e.target.checked ? it.quantity : 0 }))} />
              <span className="min-w-0 break-words">{it.product_name} {[it.size, it.color].filter(Boolean).join(' · ')}</span>
            </label>
            {it.quantity > 1 && (quantities[it.sku] ?? 0) > 0 && (
              <label className="flex items-center gap-2 text-sm">
                Số lượng
                <select className="border border-outline-variant bg-surface p-1" value={quantities[it.sku]} onChange={(e) => setQuantities((q) => ({ ...q, [it.sku]: Number(e.target.value) }))}>
                  {Array.from({ length: it.quantity }, (_, i) => i + 1).map((n) => <option key={n} value={n}>{n}</option>)}
                </select>
              </label>
            )}
          </div>
        ))}
        {errors.items && <p role="alert" className="text-error text-sm">{errors.items}</p>}
      </fieldset>

      <label className="block space-y-2">
        <span className="font-semibold">2. Lý do</span>
        <select className={inputClass} value={reason} onChange={(e) => setReason(e.target.value as ReturnReason)}>
          <option value="">Chọn lý do…</option>
          {RETURN_REASONS.map((r) => <option key={r} value={r}>{reasonLabels[r]}</option>)}
        </select>
        {errors.reason && <p role="alert" className="text-error text-sm">{errors.reason}</p>}
      </label>

      <label className="block space-y-2">
        <span className="font-semibold">Mô tả thêm {reason === 'other' ? '(bắt buộc)' : '(không bắt buộc)'}</span>
        <textarea className={inputClass} rows={3} maxLength={1000} value={note} onChange={(e) => setNote(e.target.value)} placeholder="Ví dụ: đường may ở tay áo bị bung khoảng 3 cm" />
        {errors.note && <p role="alert" className="text-error text-sm">{errors.note}</p>}
      </label>

      <div className="space-y-2">
        <p className="font-semibold">3. Ảnh hoặc video minh chứng (bắt buộc)</p>
        <p className="text-sm text-on-surface-variant">
          Chụp rõ chỗ lỗi và nhãn áo, hoặc quay video ngắn (video được nén lại sau khi tải lên, có thể mất khoảng nửa phút). Tối đa {MAX_EVIDENCE} file: ảnh JPG/PNG/WebP đến 10 MB, video MP4/MOV/WebM đến 50 MB.
        </p>
        <ul className="flex flex-wrap gap-2">
          {files.map((f) => (
            <li key={f.id} className="relative w-24 h-24 border border-outline-variant bg-surface-container overflow-hidden">
              {f.kind === 'video' ? <video src={f.preview} muted className="w-full h-full object-cover" /> : <img src={f.preview} alt={f.name} className="w-full h-full object-cover" />}
              {f.progress !== 'done' && (
                <span className={`absolute inset-0 flex items-center justify-center text-xs text-center p-1 ${f.progress === 'error' ? 'bg-error/80 text-on-error' : 'bg-black/40 text-white'}`} title={f.error}>
                  {f.progress === 'error' ? 'Lỗi tải lên' : f.kind === 'video' ? 'Đang tải và xử lý video…' : 'Đang tải…'}
                </span>
              )}
              {f.kind === 'video' && f.progress === 'done' && <span className="absolute bottom-0 left-0 bg-black/60 text-white text-[10px] px-1">Video</span>}
              <button type="button" aria-label={`Bỏ ${f.name}`} className="absolute top-0 right-0 bg-surface/90 w-6 h-6 leading-none" onClick={() => setFiles((prev) => prev.filter((x) => x.id !== f.id))}>×</button>
            </li>
          ))}
          {files.length < MAX_EVIDENCE && (
            <li>
              <button type="button" className="w-24 h-24 border border-dashed border-primary text-primary text-sm" onClick={() => input.current?.click()}>+ Thêm ảnh / video</button>
            </li>
          )}
        </ul>
        <input ref={input} type="file" accept={EVIDENCE_ACCEPT} multiple className="sr-only" tabIndex={-1} aria-hidden="true" onChange={(e) => pick(e.target.files)} />
        {files.filter((f) => f.progress === 'error').map((f) => <p key={f.id} role="alert" className="text-error text-sm">{f.name}: {f.error}</p>)}
        {fileError && <p role="alert" className="text-error text-sm">{fileError}</p>}
        {errors.evidence && <p role="alert" className="text-error text-sm">{errors.evidence}</p>}
      </div>

      {refund > 0 && <p>Số tiền dự kiến hoàn: <strong>{formatMoney(refund)} ₫</strong> <span className="text-sm text-on-surface-variant">(cửa hàng xác nhận sau khi nhận lại áo)</span></p>}
      {errors.form && <p role="alert" className="text-error">{errors.form}</p>}
      <div className="flex flex-wrap gap-3">
        <button type="submit" disabled={sending || uploading} className="bg-primary text-on-primary px-6 py-3 disabled:opacity-50">
          {sending ? 'Đang gửi…' : uploading ? 'Đang tải minh chứng…' : 'Gửi yêu cầu hoàn hàng'}
        </button>
        <button type="button" disabled={sending} className="border border-primary text-primary px-6 py-3" onClick={onCancel}>Để sau</button>
      </div>
    </form>
  )
}
