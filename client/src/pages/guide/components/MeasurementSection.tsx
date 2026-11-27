import { useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import { Link, useLocation } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import RichText from '@/components/ui/RichText'
import { guideImages, measurementFormFields, measurements } from '@/data/guide'
import { useToast } from '@/hooks/useToast'
import { accountApi, type Measurements } from '@/features/account/api'
import { useAuth } from '@/features/auth/useAuth'
import { getProduct } from '@/features/catalog/api'
import { tailoringApi, TailoringApiError } from '@/features/tailoring/api'
import {
  buildTailoringRequest, emptyTailoringForm, formFieldForServerField, measurementNumbers, productRefFromSearch, tailoringSuccessMessage, validateTailoringForm,
  type TailoringFormValues,
} from '@/features/tailoring/form'

const input = 'w-full bg-surface-bright px-3 py-2 text-body-md text-on-surface focus:outline-none focus:bg-surface'
const label = 'block font-label-regular text-label-regular text-on-surface-variant mb-1'

export default function MeasurementSection() {
  const { showToast } = useToast()
  const { user, refresh } = useAuth()
  const location = useLocation()
  const [step, setStep] = useState(1)
  const [saving, setSaving] = useState(false)
  const [sending, setSending] = useState(false)
  const sendingRef = useRef(false)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [sentId, setSentId] = useState('')
  const [loadedProduct, setLoadedProduct] = useState<{ slug: string; name: string } | null>(null)
  const [form, setForm] = useState<TailoringFormValues>({ ...emptyTailoringForm })
  const current = measurements.find((m) => m.step === step)!
  const product = useMemo(() => productRefFromSearch(location.search), [location.search])
  const productName = loadedProduct && loadedProduct.slug === product.slug ? loadedProduct.name : ''
  const setField = (key: keyof TailoringFormValues, value: string) => {
    setForm((previous) => ({ ...previous, [key]: value }))
    setFieldErrors((previous) => (previous[key] ? { ...previous, [key]: '' } : previous))
  }

  useEffect(() => {
    const slug = product.slug
    if (!slug) return
    const controller = new AbortController()
    getProduct(slug, controller.signal).then((p) => setLoadedProduct({ slug, name: p.name })).catch(() => { /* Chỉ hiện slug khi không tải được tên. */ })
    return () => controller.abort()
  }, [product.slug])

  useEffect(() => {
    if (!user) return
    let active = true
    accountApi.measurements()
      .then((data) => {
        if (!active) return
        const profile = data.measurement_profile || {}
        setForm((previous) => ({ ...previous, name: user.full_name, phone: user.phone || '', ...measurementValues(profile), notes: data.measurement_notes || '' }))
      })
      .catch(() => { /* Hồ sơ chưa có dữ liệu vẫn dùng form trống. */ })
    return () => { active = false }
  }, [user])

  // Gửi yêu cầu tư vấn: ai cũng gửi được. Chỉ báo thành công khi nhận 201 hợp lệ; lỗi giữ nguyên dữ liệu đã nhập.
  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (sendingRef.current) return
    setSentId(''); setError('')
    const errors = validateTailoringForm(form)
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) { setError('Vui lòng kiểm tra lại các trường được đánh dấu.'); return }
    sendingRef.current = true
    setSending(true)
    try {
      const receipt = await tailoringApi.submit(buildTailoringRequest(form, product))
      setSentId(receipt.id)
      setForm({ ...emptyTailoringForm })
    } catch (err) {
      if (err instanceof TailoringApiError) {
        if (err.field) setFieldErrors({ [formFieldForServerField(err.field)]: err.message })
        if (err.status === 401) setError('Phiên đăng nhập đã hết hạn. Vui lòng đăng nhập lại; nội dung bạn đã nhập vẫn được giữ trên trang này.')
        else if (err.status === 503) setError('Hệ thống tạm thời chưa nhận được yêu cầu. Yêu cầu CHƯA được gửi, vui lòng thử lại sau ít phút.')
        else setError(`${err.message} Yêu cầu chưa được gửi.`)
      } else {
        setError('Không kết nối được máy chủ. Yêu cầu chưa được gửi, nội dung bạn nhập vẫn được giữ.')
      }
    } finally {
      sendingRef.current = false
      setSending(false)
    }
  }

  // Hành vi KR-02: lưu hồ sơ số đo của tài khoản (tách biệt với việc gửi yêu cầu).
  const saveProfile = async () => {
    if (!user) return
    // Cùng kiểm tra với gửi yêu cầu (họ tên + SĐT bắt buộc, số đo hợp lệ): ô số đo là text nên không còn được trình duyệt chặn.
    const errors = validateTailoringForm(form)
    setFieldErrors(errors); setSentId('')
    if (Object.keys(errors).length > 0) { setError('Vui lòng kiểm tra lại các trường được đánh dấu.'); return }
    setSaving(true); setError('')
    try {
      await accountApi.updateProfile({ full_name: form.name.trim(), phone: form.phone.trim() })
      await accountApi.updateMeasurements({ measurement_profile: measurementPayload(form), measurement_notes: form.notes.trim() })
      await refresh()
      showToast('Đã lưu thành công hồ sơ số đo của bạn.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Không thể lưu hồ sơ số đo.')
    } finally { setSaving(false) }
  }

  return (
    <section id="bang-so-do" className="w-full py-20 px-margin-tablet lg:px-margin-desktop bg-surface-container-lowest">
      <div className="max-w-[1360px] mx-auto">
        <div className="flex flex-col items-center text-center mb-16 gap-3">
          <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Quy trình chi tiết</span>
          <h2 className="font-headline-lg text-headline-lg text-primary">8 Thông Số Vàng Tạo Nên Cốt Cách Áo Dài</h2>
          <p className="font-body-md text-body-md text-on-surface-variant max-w-xl">
            Nhấp chọn từng vị trí để xem chi tiết cách đặt thước của thợ may thủ công AN SOIE.
          </p>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 items-start">
          <div className="lg:col-span-5 bg-surface-container-high p-8 flex flex-col items-center justify-center relative shadow-sm">
            <div className="w-full flex items-center justify-between mb-4">
              <span className="font-label-uppercase text-label-uppercase text-on-surface-variant">Sơ đồ đo nhân trắc học</span>
              <span className="font-label-uppercase text-label-uppercase text-secondary">{current.indicator}</span>
            </div>
            <div className="relative w-full max-w-[340px] aspect-[1/2] bg-surface-container flex items-center justify-center overflow-hidden shadow-inner">
              <img src={guideImages.diagram.src} alt={guideImages.diagram.alt} className="w-full h-full object-cover opacity-85" />
              {measurements.map((m) => (
                <button
                  key={m.step}
                  type="button"
                  aria-label={`Số đo ${m.step}: ${m.title}`}
                  aria-pressed={m.step === step}
                  onClick={() => setStep(m.step)}
                  className={`absolute ${m.dotPosition} w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold transition-all shadow-md hover:scale-125 ${
                    m.step === step ? 'bg-primary text-on-primary ring-4 ring-primary/20' : 'bg-surface-bright text-on-surface ring-2 ring-primary/30'
                  }`}
                >
                  {m.step}
                </button>
              ))}
            </div>
            <div className="mt-6 w-full p-4 bg-surface text-center">
              <span className="font-title-editorial text-title-editorial italic text-primary">{current.focusName}</span>
              <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">{current.focusDesc}</p>
            </div>
          </div>

          <div className="lg:col-span-7 flex flex-col gap-4">
            <div className="space-y-3">
              {measurements.map((m) => {
                const active = m.step === step
                return (
                  <div
                    key={m.step}
                    role="button"
                    tabIndex={0}
                    onClick={() => setStep(m.step)}
                    onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && setStep(m.step)}
                    className={`p-5 cursor-pointer transition-all duration-200 ${active ? 'bg-surface-container-high shadow-md' : 'bg-surface-container-low'}`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <span
                          className={`w-7 h-7 flex items-center justify-center font-bold text-xs ${active ? 'bg-primary text-on-primary' : 'bg-surface-container-highest text-on-surface'}`}
                        >
                          {String(m.step).padStart(2, '0')}
                        </span>
                        <h3 className="font-headline-sm text-title-editorial text-on-surface font-medium">{m.title}</h3>
                      </div>
                      <Icon name={m.icon} size={20} className="text-secondary" />
                    </div>
                    <p className="font-body-sm text-body-sm text-on-surface-variant mt-2 pl-10">
                      <RichText text={m.guide} />
                    </p>
                  </div>
                )
              })}
            </div>

            <div className="mt-8 bg-surface-container p-6 sm:p-8 shadow-sm min-w-0">
              <div className="flex items-center justify-between mb-6 gap-3">
                <div className="min-w-0">
                  <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Yêu cầu tư vấn may đo</span>
                  <h3 className="font-headline-sm text-headline-sm text-primary">Gửi Thông Số Cho Cửa Hàng</h3>
                </div>
                <Icon name="app_registration" size={28} className="text-primary shrink-0" />
              </div>
              <p className="font-body-sm text-body-sm text-on-surface-variant mb-4">
                Bạn không cần đăng nhập. Chỉ cần họ tên và số điện thoại; số đo và ghi chú có thể bổ sung sau. Gửi yêu cầu chưa phải đặt đơn: cửa hàng sẽ liên hệ xác nhận số đo và giá.
              </p>
              {(product.id || product.slug) && (
                <p className="mb-4 p-3 bg-surface text-body-sm text-on-surface" data-testid="tailoring-product">
                  Đang tư vấn cho mẫu: <strong>{productName || product.slug || 'mẫu đã chọn'}</strong>
                  {productName && product.slug ? <span className="text-on-surface-variant"> ({product.slug})</span> : null}
                  {fieldErrors.product && <span role="alert" className="block text-red-700">{fieldErrors.product}</span>}
                </p>
              )}
              {sentId && (
                <div role="status" className="mb-4 p-4 bg-surface text-on-surface border-l-4 border-primary">
                  <p>{tailoringSuccessMessage}</p>
                  <p className="text-body-sm text-on-surface-variant mt-1">Mã tham chiếu: <span className="font-mono">{sentId.slice(-8).toUpperCase()}</span></p>
                </div>
              )}
              <form className="space-y-4" onSubmit={submit} noValidate>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className={label} htmlFor="m-name">Họ và tên quý khách *</label>
                    <input id="m-name" required type="text" value={form.name} onChange={(event) => setField('name', event.target.value)} aria-invalid={!!fieldErrors.name} className={`${input} px-4 py-2.5 border-none shadow-inner`} placeholder="Nguyễn Thị Thanh Hà" />
                    <FieldError message={fieldErrors.name} />
                  </div>
                  <div>
                    <label className={label} htmlFor="m-phone">Số điện thoại / Zalo *</label>
                    <input id="m-phone" required type="tel" value={form.phone} onChange={(event) => setField('phone', event.target.value)} aria-invalid={!!fieldErrors.phone} className={`${input} px-4 py-2.5 border-none shadow-inner`} placeholder="09xx xxx xxx" />
                    <FieldError message={fieldErrors.phone} />
                  </div>
                </div>
                <div>
                  <label className={label} htmlFor="m-contact">Kênh liên hệ ưu tiên (không bắt buộc)</label>
                  <select id="m-contact" value={form.contact} onChange={(event) => setField('contact', event.target.value)} className={`${input} px-4 py-2.5`}>
                    <option value="">Không chọn</option>
                    <option value="phone">Điện thoại</option>
                    <option value="zalo">Zalo</option>
                  </select>
                  <FieldError message={fieldErrors.contact} />
                </div>
                <p className="font-body-sm text-body-sm text-on-surface-variant pt-2">Số đo (không bắt buộc). Ô để trống sẽ không được gửi.</p>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  {measurementFormFields.map((f) => (
                    <div key={f.id} className="min-w-0">
                      <label className={label} htmlFor={`m-${f.id}`}>{f.label}</label>
                      <input id={`m-${f.id}`} type={f.type === 'number' ? 'text' : f.type} inputMode={f.type === 'number' ? 'decimal' : undefined} value={form[f.id as keyof TailoringFormValues]} onChange={(event) => setField(f.id as keyof TailoringFormValues, event.target.value)} aria-invalid={!!fieldErrors[f.id]} className={input} placeholder={f.placeholder} />
                      <FieldError message={fieldErrors[f.id]} />
                    </div>
                  ))}
                </div>
                <div>
                  <label className={label} htmlFor="m-notes">Ghi chú vóc dáng đặc biệt hoặc yêu cầu tà áo</label>
                  <textarea
                    id="m-notes"
                    rows={2}
                    value={form.notes}
                    onChange={(event) => setField('notes', event.target.value)}
                    aria-invalid={!!fieldErrors.notes}
                    className={`${input} px-4 py-2.5 resize-none`}
                    placeholder="Ví dụ: Lưng hơi cong nhẹ, muốn tà xẻ cao kín cạp quần, thích mặc tà rộng truyền thống..."
                  />
                  <FieldError message={fieldErrors.notes} />
                </div>
                <div className="pt-2 flex flex-col gap-4">
                  <span className="font-body-sm text-body-sm text-on-surface-variant flex items-start gap-1.5">
                    <Icon name="verified_user" size={16} className="text-secondary shrink-0 mt-0.5" />
                    Nghệ nhân chính sẽ kiểm tra tỉ lệ và gọi điện tư vấn trước khi cắt lụa.
                  </span>
                  <div className="flex flex-col sm:flex-row gap-3">
                    <button
                      type="submit"
                      disabled={sending}
                      className="w-full sm:w-auto px-8 py-3 bg-primary text-on-primary font-label-uppercase text-label-uppercase tracking-widest shadow-md hover:bg-primary-container transition-colors disabled:opacity-60"
                    >
                      {sending ? 'Đang gửi yêu cầu...' : 'Gửi Yêu Cầu Tư Vấn'}
                    </button>
                    {user ? (
                      <button type="button" onClick={saveProfile} disabled={saving || sending} className="w-full sm:w-auto px-8 py-3 bg-surface-container-high text-primary font-label-uppercase text-label-uppercase tracking-widest hover:bg-surface-container-highest transition-colors disabled:opacity-60">
                        {saving ? 'Đang lưu...' : 'Lưu vào hồ sơ số đo'}
                      </button>
                    ) : (
                      <Link to={`/dang-nhap?next=${encodeURIComponent(location.pathname + location.search + '#bang-so-do')}`} className="w-full sm:w-auto px-8 py-3 text-center text-primary font-label-uppercase text-label-uppercase tracking-widest underline">
                        Đăng nhập để lưu hồ sơ số đo
                      </Link>
                    )}
                  </div>
                </div>
                {error && (
                  <p role="alert" className="text-red-700">
                    {error}
                    {error.startsWith('Phiên đăng nhập') && <> <Link className="underline" to={`/dang-nhap?next=${encodeURIComponent(location.pathname + location.search + '#bang-so-do')}`}>Đăng nhập lại</Link></>}
                  </p>
                )}
              </form>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

function FieldError({ message }: { message?: string }) {
  return message ? <p role="alert" className="mt-1 text-body-sm text-red-700">{message}</p> : null
}

function measurementValues(profile: Measurements): Partial<TailoringFormValues> {
  return {
    neck: value(profile.neck_cm), bust: value(profile.bust_cm), waist: value(profile.waist_cm), hips: value(profile.hips_cm),
    shoulder: value(profile.shoulder_cm), sleeve: value(profile.sleeve_cm), length: value(profile.garment_length_cm), body: profile.body || '',
  }
}

function measurementPayload(form: TailoringFormValues): Measurements {
  // Parser dùng chung: ô trống ⇒ bỏ qua, không bao giờ NaN/âm/0 (đã validate trước khi gọi).
  return { ...measurementNumbers(form), body: form.body.trim() || undefined }
}

function value(input?: number) { return input == null ? '' : String(input) }
