import { Link } from 'react-router-dom'
import Icon from '@/components/ui/Icon'
import { ROUTES } from '@/data/site'

export default function BespokeBanner() {
  return (
    <div className="mt-16 md:mt-24 p-8 md:p-12 bg-surface-container flex flex-col md:flex-row items-center gap-8 shadow-sm">
      <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0 text-primary">
        <Icon name="architecture" size={32} />
      </div>
      <div className="flex-1 text-center md:text-left">
        <h4 className="font-headline-sm text-headline-sm text-primary mb-2">Đặc Quyền May Đo Chuẩn Dáng Miễn Phí</h4>
        <p className="font-body-md text-body-md text-on-surface-variant leading-relaxed">
          Mọi mẫu áo dài tại AN SOIE đều có thể tinh chỉnh theo số đo riêng của quý cô mà không phát sinh thêm phụ phí. Từng đường cắt lượn eo, tà áo và vòng
          nách đều được người thợ cả chăm chút tỉ mẩn nhằm tôn vinh vóc dáng yêu kiều của phụ nữ Việt.
        </p>
      </div>
      <Link
        to={ROUTES.tailoring}
        className="flex-shrink-0 px-6 py-3.5 bg-primary text-on-primary font-label-uppercase text-label-uppercase tracking-widest hover:bg-primary-container transition-colors inline-block text-center shadow-md"
      >
        Gửi Số Đo Của Quý Cô
      </Link>
    </div>
  )
}
