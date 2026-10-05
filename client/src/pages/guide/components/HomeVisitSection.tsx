import { useState, type FormEvent } from 'react'
import { useToast } from '@/hooks/useToast'

export default function HomeVisitSection() {
  const { showToast } = useToast()
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const form = e.currentTarget
    setSubmitting(true)
    // TODO: gọi API đặt lịch may đo tận nơi
    window.setTimeout(() => {
      showToast('Cảm ơn quý khách! Lịch hẹn may đo đã được ghi nhận. Quản lý tư gia sẽ gọi xác nhận trong ít phút.')
      form.reset()
      setSubmitting(false)
    }, 400)
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
              Dành cho quý cô bận rộn hoặc mong muốn trải nghiệm may đo riêng tư, thư thái nhất. Chuyên viên may đo và stylist của AN SOIE sẽ mang toàn bộ
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
            <form className="space-y-4" onSubmit={handleSubmit}>
              <div>
                <label className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Tên của bạn</label>
                <input
                  className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none"
                  placeholder="Trần Khánh Vân"
                  required
                  type="text"
                />
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Số điện thoại</label>
                  <input
                    className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none"
                    placeholder="0912 xxx xxx"
                    required
                    type="tel"
                  />
                </div>
                <div>
                  <label className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Khu vực hẹn</label>
                  <select className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none">
                    <option value="HN">Hà Nội (Nội thành & Lân cận)</option>
                    <option value="HCM">TP. Hồ Chí Minh (Các quận nội thành)</option>
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Ngày mong muốn</label>
                  <input
                    className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none"
                    type="date"
                  />
                </div>
                <div>
                  <label className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Khung giờ hẹn</label>
                  <select className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none">
                    <option>Buổi sáng (09:00 - 11:30)</option>
                    <option>Buổi chiều (14:00 - 17:00)</option>
                    <option>Buổi tối (18:30 - 20:30)</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="block font-label-regular text-label-regular text-on-surface-variant mb-1">Địa chỉ tư gia hoặc văn phòng</label>
                <input
                  className="w-full bg-surface-container px-4 py-2.5 text-body-md text-on-surface focus:outline-none focus:bg-surface border-none"
                  placeholder="Số nhà, tên đường, phường/quận..."
                  required
                  type="text"
                />
              </div>
              <button
                className="w-full py-3.5 bg-primary text-on-primary font-label-uppercase text-label-uppercase tracking-widest shadow-md hover:bg-primary-container transition-colors mt-2"
                type="submit"
                disabled={submitting}
              >
                Xác Nhận Đặt Lịch Chuyên Viên
              </button>
            </form>
          </div>
        </div>
      </div>
    </section>
  )
}
