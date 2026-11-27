import { useState, type FormEvent } from 'react'
import { useConsultSubmit } from '@/features/tailoring/useConsultSubmit'

const slots = ['Buổi sáng (09:00 - 11:30)', 'Buổi chiều (14:00 - 17:00)', 'Buổi tối (18:30 - 20:30)']

export default function HomeVisitSection() {
  const { submitting, success, error, fieldErrors: serverErrors, submit } = useConsultSubmit()
  const [name, setName] = useState('')
  const [phone, setPhone] = useState('')
  const [area, setArea] = useState('Hà Nội (Nội thành & Lân cận)')
  const [date, setDate] = useState('')
  const [slot, setSlot] = useState(slots[0])
  const [address, setAddress] = useState('')
  const [addressError, setAddressError] = useState('')
  const fieldErrors: Record<string, string> = { ...serverErrors, ...(addressError ? { address: addressError } : {}) }

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!address.trim()) {
      setAddressError('Vui lòng nhập địa chỉ để chuyên viên đến đo.')
    } else {
      setAddressError('')
    }
    if (!address.trim()) return
    const ok = await submit({
      name,
      phone,
      extras: [['Dịch vụ', 'May đo tận nơi'], ['Khu vực', area], ['Ngày mong muốn', date], ['Khung giờ', slot], ['Địa chỉ', address]],
    })
    if (ok) {
      setName(''); setPhone(''); setArea('Hà Nội (Nội thành & Lân cận)'); setDate(''); setSlot(slots[0]); setAddress('')
    }
  }

  return (
    <section className="w-full py-20 px-margin-tablet lg:px-margin-desktop bg-surface" id="dat-lich-tan-noi">
      <div className="max-w-[1360px] mx-auto">
        <div className="bg-surface-container-low p-8 lg:p-14 shadow-sm grid grid-cols-1 lg:grid-cols-12 gap-12 items-center">
          <div className="lg:col-span-6 flex flex-col gap-6">
            <div className="flex items-center gap-2">
              <span className="px-2.5 py-1 bg-primary text-on-primary font-label-uppercase text-label-uppercase">Đặc quyền cao cấp</span>
              <span className="font-label-uppercase text-label-uppercase text-secondary">Hà Nội • TP. Hồ Chí Minh</span>
            </div>
            <h2 className="font-headline-lg text-headline-lg text-primary tracking-tight">Dịch Vụ 'May Đo Tận Nơi' Tại Tư Gia & Cơ Quan</h2>
            <p className="font-body-md text-body-md text-on-surface-variant leading-relaxed">
              Dành cho quý cô bận rộn hoặc mong muốn trải nghiệm may đo riêng tư, thư thái nhất. Chuyên viên may đo và stylist của AODORA sẽ mang toàn bộ
              catalogue mẫu vải tơ tằm thượng hạng, bảng màu nhuộm tự nhiên và thước may đến tận nơi bạn yêu cầu.
            </p>
            <ul className="space-y-3 font-body-md text-body-md text-on-surface">
              <li className="flex items-center gap-3">
                <span className="material-symbols-outlined text-secondary text-[20px]">palette</span>
                <span>Trực tiếp cảm nhận chất vải lụa tơ tằm, sa tằm, gấm vân hoàng gia tận mắt.</span>
              </li>
              <li className="flex items-center gap-3">
                <span className="material-symbols-outlined text-secondary text-[20px]">person_check</span>
                <span>Chuyên viên giàu kinh nghiệm lấy 18 thông số tỉ mỉ, kiểm tra độ rơi tà áo.</span>
              </li>
              <li className="flex items-center gap-3">
                <span className="material-symbols-outlined text-secondary text-[20px]">local_shipping</span>
                <span>Giao áo tận tay và hỗ trợ thử phom trực tiếp kèm chỉnh sửa nếu chưa hài lòng.</span>
              </li>
            </ul>
          </div>

          <div className="lg:col-span-6 bg-surface-bright p-8 shadow-md">
            <h3 className="font-headline-sm text-headline-sm text-primary mb-2">Đăng Ký Lịch Đo May Tận Nơi</h3>
            <p className="font-body-sm text-body-sm text-on-surface-variant mb-6">
              Xin quý khách vui lòng để lại thông tin, nhân viên tư vấn sẽ liên hệ xác nhận trong 30 phút.
            </p>
            <form className="space-y-4" onSubmit={(e) => void handleSubmit(e)} noValidate>
              <div>
                <label htmlFor="visit-name" className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Tên của bạn</label>
                <input id="visit-name" name="name" autoComplete="name" className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none" placeholder="Trần Khánh Vân" required type="text" value={name} onChange={(e) => setName(e.target.value)} aria-invalid={Boolean(fieldErrors.name)} aria-describedby={fieldErrors.name ? 'visit-name-error' : undefined} />
                {fieldErrors.name && <p id="visit-name-error" className="text-red-700 text-body-sm mt-1">{fieldErrors.name}</p>}
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="visit-phone" className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Số điện thoại</label>
                  <input id="visit-phone" name="phone" autoComplete="tel" className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none" placeholder="0912 xxx xxx" required type="tel" value={phone} onChange={(e) => setPhone(e.target.value)} aria-invalid={Boolean(fieldErrors.phone)} aria-describedby={fieldErrors.phone ? 'visit-phone-error' : undefined} />
                  {fieldErrors.phone && <p id="visit-phone-error" className="text-red-700 text-body-sm mt-1">{fieldErrors.phone}</p>}
                </div>
                <div>
                  <label htmlFor="visit-area" className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Khu vực hẹn</label>
                  <select id="visit-area" name="area" className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none" value={area} onChange={(e) => setArea(e.target.value)}>
                    <option value="Hà Nội (Nội thành & Lân cận)">Hà Nội (Nội thành &amp; Lân cận)</option>
                    <option value="TP. Hồ Chí Minh (Các quận nội thành)">TP. Hồ Chí Minh (Các quận nội thành)</option>
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label htmlFor="visit-date" className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Ngày mong muốn</label>
                  <input id="visit-date" name="date" className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
                </div>
                <div>
                  <label htmlFor="visit-slot" className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Khung giờ hẹn</label>
                  <select id="visit-slot" name="slot" className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none" value={slot} onChange={(e) => setSlot(e.target.value)}>
                    {slots.map((s) => <option key={s}>{s}</option>)}
                  </select>
                </div>
              </div>
              <div>
                <label htmlFor="visit-address" className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Địa chỉ tư gia hoặc văn phòng</label>
                <input id="visit-address" name="address" autoComplete="street-address" className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none" placeholder="Số nhà, tên đường, phường/quận..." required type="text" value={address} onChange={(e) => setAddress(e.target.value)} aria-invalid={Boolean(fieldErrors.address)} aria-describedby={fieldErrors.address ? 'visit-address-error' : undefined} />
                {fieldErrors.address && <p id="visit-address-error" className="text-red-700 text-body-sm mt-1">{fieldErrors.address}</p>}
              </div>
              {success && <p role="status" className="bg-surface-container text-primary px-4 py-3 text-body-sm">{success}</p>}
              {error && <p role="alert" className="bg-error-container text-red-700 px-4 py-3 text-body-sm">{error}</p>}
              <button
                className="w-full py-3.5 bg-primary text-on-primary font-label-uppercase text-label-uppercase tracking-widest shadow-md hover:bg-primary-container transition-colors mt-2 disabled:opacity-60"
                type="submit"
                disabled={submitting}
              >
                {submitting ? 'Đang gửi...' : 'Gửi Yêu Cầu Đặt Lịch'}
              </button>
            </form>
          </div>
        </div>
      </div>
    </section>
  )
}
