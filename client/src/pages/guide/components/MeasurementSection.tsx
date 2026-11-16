import { useEffect, useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import RichText from '@/components/ui/RichText'
import { guideImages, measurementFormFields, measurements } from '@/data/guide'
import { useToast } from '@/hooks/useToast'
import { accountApi, type Measurements } from '@/features/account/api'
import { useAuth } from '@/features/auth/useAuth'

const input = 'w-full bg-surface-bright px-3 py-2 text-body-md text-on-surface focus:outline-none focus:bg-surface'
const label = 'block font-label-regular text-label-regular text-on-surface-variant mb-1'

export default function MeasurementSection() {
  const { showToast } = useToast()
  const { user, refresh } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [step, setStep] = useState(1)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState<Record<string, string>>({ name: '', phone: '', neck: '', bust: '', waist: '', hips: '', shoulder: '', sleeve: '', length: '', body: '', notes: '' })
  const current = measurements.find((m) => m.step === step)!

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

  const submit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!user) {
      navigate(`/dang-nhap?next=${encodeURIComponent(location.pathname + '#bang-so-do')}`)
      return
    }
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

            <div className="mt-8 bg-surface-container p-8 shadow-sm">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Lưu trữ hồ sơ may đo</span>
                  <h3 className="font-headline-sm text-headline-sm text-primary">Nhập và Lưu Thông Số Của Quý Khách</h3>
                </div>
                <Icon name="app_registration" size={28} className="text-primary" />
              </div>
              <form className="space-y-4" onSubmit={submit}>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className={label} htmlFor="m-name">
                      Họ và tên quý khách
                    </label>
                    <input id="m-name" required type="text" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} className={`${input} px-4 py-2.5 border-none shadow-inner`} placeholder="Nguyễn Thị Thanh Hà" />
                  </div>
                  <div>
                    <label className={label} htmlFor="m-phone">
                      Số điện thoại / Zalo
                    </label>
                    <input id="m-phone" required type="tel" value={form.phone} onChange={(event) => setForm({ ...form, phone: event.target.value })} className={`${input} px-4 py-2.5 border-none shadow-inner`} placeholder="09xx xxx xxx" />
                  </div>
                </div>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-2">
                  {measurementFormFields.map((f) => (
                    <div key={f.id}>
                      <label className={label} htmlFor={`m-${f.id}`}>
                        {f.label}
                      </label>
                        <input id={`m-${f.id}`} type={f.type} min={f.type === 'number' ? 0 : undefined} value={form[f.id]} onChange={(event) => setForm({ ...form, [f.id]: event.target.value })} className={input} placeholder={f.placeholder} />
                    </div>
                  ))}
                </div>
                <div>
                  <label className={label} htmlFor="m-notes">
                    Ghi chú vóc dáng đặc biệt hoặc yêu cầu tà áo
                  </label>
                  <textarea
                    id="m-notes"
                    rows={2}
                    value={form.notes}
                    onChange={(event) => setForm({ ...form, notes: event.target.value })}
                    className={`${input} px-4 py-2.5 resize-none`}
                    placeholder="Ví dụ: Lưng hơi cong nhẹ, muốn tà xẻ cao kín cạp quần, thích mặc tà rộng truyền thống..."
                  />
                </div>
                <div className="pt-2 flex flex-col sm:flex-row items-center justify-between gap-4">
                  <span className="font-body-sm text-body-sm text-on-surface-variant flex items-center gap-1.5">
                    <Icon name="verified_user" size={16} className="text-secondary" />
                    Nghệ nhân chính sẽ kiểm tra tỉ lệ và gọi điện tư vấn trước khi cắt lụa.
                  </span>
                  <button
                    type="submit"
                    disabled={saving}
                    className="w-full sm:w-auto px-8 py-3 bg-primary text-on-primary font-label-uppercase text-label-uppercase tracking-widest shadow-md hover:bg-primary-container transition-colors"
                  >
                    {saving ? 'Đang lưu...' : user ? 'Lưu Hồ Sơ Số Đo' : 'Đăng nhập để lưu số đo'}
                  </button>
                </div>
                {error && <p role="alert" className="text-red-700">{error}</p>}
              </form>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

function measurementValues(profile: Measurements): Record<string, string> {
  return {
    neck: value(profile.neck_cm), bust: value(profile.bust_cm), waist: value(profile.waist_cm), hips: value(profile.hips_cm),
    shoulder: value(profile.shoulder_cm), sleeve: value(profile.sleeve_cm), length: value(profile.garment_length_cm), body: profile.body || '',
  }
}

function measurementPayload(form: Record<string, string>): Measurements {
  const value = (key: string) => form[key] === '' ? undefined : Number(form[key])
  return { neck_cm: value('neck'), bust_cm: value('bust'), waist_cm: value('waist'), hips_cm: value('hips'), shoulder_cm: value('shoulder'), sleeve_cm: value('sleeve'), garment_length_cm: value('length'), body: form.body.trim() || undefined }
}

function value(input?: number) { return input == null ? '' : String(input) }
