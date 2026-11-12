import Icon from '@/components/ui/Icon'
import Modal from '@/components/ui/Modal'

export default function SizeGuideModal({ open, onClose, sizes }: { open: boolean; onClose: () => void; sizes: string[] }) {
  return <Modal open={open} onClose={onClose}>
    <div className="flex items-center justify-between pb-4">
      <div><p className="uppercase tracking-widest text-secondary text-sm">Hướng dẫn chọn kích thước</p>
        <h3 className="font-headline-md text-headline-md text-primary">Thông Tin Size May Sẵn</h3></div>
      <button type="button" aria-label="Đóng" onClick={onClose} className="p-2"><Icon name="close" size={24} /></button>
    </div>
    <p className="text-on-surface-variant leading-relaxed my-4">Các kích cỡ đang được công bố cho mẫu áo này:</p>
    <div className="flex gap-2 flex-wrap mb-5">{sizes.map((size) => <span key={size} className="bg-surface-container px-4 py-2">{size}</span>)}</div>
    <p className="text-on-surface-variant">Bảng số đo tương ứng mỗi size chưa được cung cấp qua catalog. Không nên suy đoán thông số cơ thể chỉ từ ký hiệu size; hãy liên hệ tư vấn may đo khi cần.</p>
    <button type="button" onClick={onClose} className="mt-6 bg-primary text-on-primary px-5 py-2">Đã hiểu</button>
  </Modal>
}
