import Icon from '@/components/ui/Icon'
import Modal from '@/components/ui/Modal'
import { sizeTable } from '@/data/productDetail'

export default function SizeGuideModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <Modal open={open} onClose={onClose}>
      <div className="flex items-center justify-between pb-4">
        <div>
          <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">Bảng Tham Khảo Chuẩn Dáng</span>
          <h3 className="font-headline-md text-headline-md text-primary font-normal">Kích Thước Áo Dài Phụ Nữ Việt</h3>
        </div>
        <button type="button" aria-label="Đóng" onClick={onClose} className="text-on-surface-variant hover:text-primary p-1">
          <Icon name="close" size={24} />
        </button>
      </div>
      <p className="font-body-md text-body-md text-on-surface-variant my-4">
        Các thông số dưới đây là chuẩn rập may sẵn. Nếu quý khách có số đo nằm giữa hai kích cỡ hoặc vóc dáng đặc biệt, chúng tôi luôn khuyến nghị lựa chọn chế
        độ <strong>"May Đo Theo Số Đo Riêng"</strong> hoàn toàn không tính thêm phụ phí.
      </p>
      <div className="overflow-x-auto my-4">
        <table className="w-full text-left font-body-sm text-body-sm">
          <thead className="bg-surface-container font-label-uppercase text-label-uppercase text-primary">
            <tr>
              {sizeTable.headers.map((h) => (
                <th key={h} className="p-3">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {sizeTable.rows.map((row, i) => (
              <tr key={row[0]} className={`${i % 2 === 0 ? 'bg-surface-container-low' : 'bg-surface'} hover:bg-surface-container`}>
                {row.map((cell, j) => (
                  <td key={j} className={`p-3 ${j === 0 ? 'font-bold text-primary' : ''}`}>
                    {cell}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="bg-surface-container p-4 mt-6 flex items-center justify-between gap-4">
        <span className="font-body-sm text-body-sm text-on-surface">Cần hỗ trợ chuyên viên may đo tư vấn trực tiếp qua Zalo/Điện thoại?</span>
        <button
          type="button"
          onClick={onClose}
          className="bg-primary text-on-primary px-4 py-2 font-label-uppercase text-label-uppercase tracking-wider flex-shrink-0"
        >
          Đã Hiểu
        </button>
      </div>
    </Modal>
  )
}
