import { useState, type FormEvent } from 'react'
import { bespokeCities, bespokeInterests, bespokePerks } from '@/data/home'
import { useConsultSubmit } from '@/features/tailoring/useConsultSubmit'
import Icon from '@/components/ui/Icon'

export default function BespokeSection() {
  const { submitting, success, error, fieldErrors, submit } = useConsultSubmit()
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [city, setCity] = useState(bespokeCities[0])
  const [interest, setInterest] = useState(bespokeInterests[0])
  const [notes, setNotes] = useState('')

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const ok = await submit({
      name,
      phone,
      extras: [['Thành phố', city], ['Dòng áo quan tâm', interest], ['Ghi chú', notes]],
    })
    if (ok) {
      setName(''); setPhone(''); setCity(bespokeCities[0]); setInterest(bespokeInterests[0]); setNotes('')
    }
  }

  return (
    <section className="w-full py-16 lg:py-24 bg-primary text-on-primary relative overflow-hidden" id="dat-may-do">
      <div className="absolute -right-24 -bottom-24 w-96 h-96 bg-primary-container rounded-full blur-3xl opacity-50 pointer-events-none"></div>
      <div className="max-w-[1440px] mx-auto px-margin md:px-margin-tablet lg:px-margin-desktop relative z-10">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
          <div className="lg:col-span-7 space-y-6">
            <div className="inline-flex items-center gap-2 bg-primary-container/80 px-3.5 py-1.5 rounded-full text-secondary-fixed">
              <span className="material-symbols-outlined text-[16px]">verified</span>
              <span className="font-label-uppercase text-label-uppercase uppercase tracking-wider">Đặc Quyền Bespoke Cao Cấp</span>
            </div>
            <h2 className="font-headline-lg lg:font-display-hero text-headline-lg lg:text-display-hero text-on-primary leading-tight">
              May Đo Tận Nơi &<br />
              Nhận Cuốn Sách Lụa Mẫu Miễn Phí
            </h2>
            <p className="font-body-lg text-body-lg text-on-primary-container max-w-xl font-light leading-relaxed">
              Nghệ nhân may đo của AODORA sẽ trực tiếp mang hộp 32 mẫu lụa tơ tằm thực tế đến tư vấn phong thái và lấy số đo riêng tại không gian nhà quý khách
              (Áp dụng tại Hà Nội & TP.HCM).
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
              {bespokePerks.map((perk) => (
                <div key={perk.title} className="flex items-start gap-3">
                  <Icon name={perk.icon} size={24} className="text-secondary-fixed" />
                  <div>
                    <strong className="block font-label-regular text-label-regular text-on-primary">{perk.title}</strong>
                    <span className="font-body-sm text-body-sm text-on-primary-container">{perk.description}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="lg:col-span-5 bg-surface text-on-surface p-8 shadow-2xl">
            <div className="mb-6">
              <h3 className="font-headline-sm text-headline-sm text-primary">Đặt Lịch Nghệ Nhân Thăm Khám</h3>
              <p className="font-body-sm text-body-sm text-on-surface-variant mt-1">Chuyên viên may đo sẽ liên hệ xác nhận trong vòng 30 phút.</p>
            </div>
            <form className="space-y-4" onSubmit={(e) => void handleSubmit(e)} noValidate>
              <div>
                <label htmlFor="bespoke-name" className="block font-label-uppercase text-label-uppercase text-secondary mb-1">Họ và tên quý khách *</label>
                <input
                  id="bespoke-name"
                  name="name"
                  autoComplete="name"
                  className="w-full bg-surface-container px-4 py-3 text-body-md text-on-surface focus:outline-none focus:bg-surface-bright"
                  placeholder="Ví dụ: Hoàng Như Ngọc"
                  required
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  aria-invalid={Boolean(fieldErrors.name)}
                  aria-describedby={fieldErrors.name ? 'bespoke-name-error' : undefined}
                />
                {fieldErrors.name && <p id="bespoke-name-error" className="text-red-700 text-body-sm mt-1">{fieldErrors.name}</p>}
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label htmlFor="bespoke-phone" className="block font-label-uppercase text-label-uppercase text-secondary mb-1">Số điện thoại *</label>
                  <input
                    id="bespoke-phone"
                    name="phone"
                    autoComplete="tel"
                    className="w-full bg-surface-container px-4 py-3 text-body-md text-on-surface focus:outline-none focus:bg-surface-bright"
                    placeholder="0988 xxx xxx"
                    required
                    type="tel"
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    aria-invalid={Boolean(fieldErrors.phone)}
                    aria-describedby={fieldErrors.phone ? 'bespoke-phone-error' : undefined}
                  />
                  {fieldErrors.phone && <p id="bespoke-phone-error" className="text-red-700 text-body-sm mt-1">{fieldErrors.phone}</p>}
                </div>
                <div>
                  <label htmlFor="bespoke-city" className="block font-label-uppercase text-label-uppercase text-secondary mb-1">Thành phố</label>
                  <select id="bespoke-city" className="w-full bg-surface-container px-4 py-3 text-body-md text-on-surface focus:outline-none focus:bg-surface-bright" name="city" value={city} onChange={(e) => setCity(e.target.value)}>
                    {bespokeCities.map((c) => (
                      <option key={c}>{c}</option>
                    ))}
                  </select>
                </div>
              </div>
              <div>
                <label htmlFor="bespoke-interest" className="block font-label-uppercase text-label-uppercase text-secondary mb-1">Dịp may mặc &amp; Dòng áo quan tâm</label>
                <select id="bespoke-interest" className="w-full bg-surface-container px-4 py-3 text-body-md text-on-surface focus:outline-none focus:bg-surface-bright" name="interest" value={interest} onChange={(e) => setInterest(e.target.value)}>
                  {bespokeInterests.map((c) => (
                    <option key={c}>{c}</option>
                  ))}
                </select>
              </div>
              <div>
                <label htmlFor="bespoke-notes" className="block font-label-uppercase text-label-uppercase text-secondary mb-1">Ghi chú yêu cầu đặc biệt (tùy chọn)</label>
                <textarea
                  id="bespoke-notes"
                  name="notes"
                  className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface-bright resize-none"
                  placeholder="Ví dụ: Cần may gấp trong 5 ngày, yêu cầu đo tại nhà riêng vào cuối tuần..."
                  rows={2}
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                ></textarea>
              </div>
              {success && <p role="status" className="bg-surface-container text-primary px-4 py-3 text-body-sm">{success}</p>}
              {error && <p role="alert" className="bg-error-container text-red-700 px-4 py-3 text-body-sm">{error}</p>}
              <button
                className="w-full bg-primary hover:bg-primary-container text-on-primary py-4 font-label-uppercase text-label-uppercase transition-colors shadow-md disabled:opacity-60"
                type="submit"
                disabled={submitting}
              >
                {submitting ? 'Đang gửi...' : 'Gửi Yêu Cầu May Đo & Nhận Sách Lụa'}
              </button>
            </form>
            <p className="font-body-sm text-body-sm text-outline text-center mt-4">
              Hotline nghệ nhân tư vấn riêng: <strong className="text-primary">0988 888 888</strong>
            </p>
          </div>
        </div>
      </div>
    </section>
  )
}
