import { useState } from 'react'
import Icon from '@/components/ui/Icon'
import StarRating from '@/components/ui/StarRating'
import { productDetail as p } from '@/data/productDetail'
import { useToast } from '@/hooks/useToast'
import { formatVND } from '@/utils/format'

type OrderMode = 'ready' | 'custom'

const HINT: Record<OrderMode, string> = {
  ready: '* Kích cỡ may sẵn theo bảng tỷ lệ chuẩn, giao nhanh trong 2-3 ngày làm việc.',
  custom: '* Dịch vụ nghệ nhân cắt may riêng theo chuẩn phom người, bảo đảm tà áo rủ êm và vừa vặn tuyệt đối.',
}

const modeBtn = 'py-3 px-3 text-center font-label-uppercase text-label-uppercase transition-all'
const field = 'w-full bg-surface-bright px-3 py-2 text-body-md font-body-md text-on-surface focus:outline-none focus:bg-surface-container-lowest shadow-sm'

export default function PurchasePanel({ onOpenSizeGuide }: { onOpenSizeGuide: () => void }) {
  const { showToast } = useToast()
  const [mode, setMode] = useState<OrderMode>('custom')
  const [color, setColor] = useState(p.colors[0])
  const [size, setSize] = useState('M')
  const [qty, setQty] = useState(1)
  const [wishlisted, setWishlisted] = useState(false)

  const toggleWishlist = () => {
    setWishlisted((w) => !w)
    showToast(wishlisted ? 'Đã bỏ khỏi danh sách yêu thích.' : 'Đã lưu tác phẩm vào danh sách yêu thích của quý khách.')
  }

  return (
    <div className="lg:col-span-5 flex flex-col">
      <div className="flex items-center justify-between pb-2">
        <span className="font-label-uppercase text-label-uppercase text-secondary tracking-widest uppercase">{p.collection}</span>
        <span className="font-body-sm text-body-sm text-on-surface-variant">Mã SP: {p.sku}</span>
      </div>
      <h1 className="font-headline-lg text-headline-lg text-primary tracking-tight font-normal leading-tight mt-1 mb-3">{p.name}</h1>

      <div className="flex items-center gap-3 pb-5 flex-wrap">
        <StarRating value={Math.round(p.rating)} />
        <span className="font-label-regular text-label-regular font-bold text-on-surface">{p.rating.toFixed(1)}</span>
        <span className="text-outline-variant font-light">•</span>
        <a className="font-body-sm text-body-sm text-on-surface-variant hover:text-primary underline transition-colors" href="#danh-gia-khach-hang">
          {p.reviewCount} Đánh giá khách hàng
        </a>
        <span className="text-outline-variant font-light">•</span>
        <span className="bg-secondary-fixed text-on-secondary-fixed px-2 py-0.5 font-label-uppercase text-label-uppercase font-bold">Atelier Bespoke</span>
      </div>

      <div className="bg-surface-container-low p-5 mb-6">
        <div className="flex items-baseline gap-3">
          <span className="font-display-hero-mobile text-display-hero-mobile text-primary font-normal">
            {formatVND(p.price).replace('đ', '')}
            <span className="text-headline-sm font-headline-sm">₫</span>
          </span>
          <span className="font-body-sm text-body-sm text-on-surface-variant line-through">{formatVND(p.oldPrice).replace('đ', '₫')}</span>
          <span className="bg-primary-fixed text-on-primary-fixed px-2 py-0.5 text-label-uppercase font-label-uppercase font-bold">-{p.discount}%</span>
        </div>
        <p className="font-body-sm text-body-sm text-on-surface-variant mt-2 flex items-center gap-2">
          <Icon name="check_circle" size={18} className="text-primary" />
          <span>
            Trọn bộ: <strong>{p.bundle}</strong>
          </span>
        </p>
      </div>

      <div className="flex flex-col gap-2 mb-6">
        <div role="tablist" className="grid grid-cols-2 bg-surface-container p-1 gap-1">
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'ready'}
            onClick={() => setMode('ready')}
            className={`${modeBtn} ${mode === 'ready' ? 'bg-primary text-on-primary shadow-sm font-bold' : 'text-on-surface-variant hover:text-on-surface'}`}
          >
            Chọn Size May Sẵn
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={mode === 'custom'}
            onClick={() => setMode('custom')}
            className={`${modeBtn} flex items-center justify-center gap-1.5 ${mode === 'custom' ? 'bg-primary text-on-primary shadow-sm font-bold' : 'text-on-surface-variant hover:text-on-surface'}`}
          >
            <Icon name="straighten" size={16} />
            May Đo Theo Số Đo (Khuyên Dùng)
          </button>
        </div>
        <p className="font-body-sm text-body-sm text-secondary italic">{HINT[mode]}</p>
      </div>

      <div className="mb-6">
        <div className="flex items-center justify-between mb-2">
          <span className="font-label-uppercase text-label-uppercase tracking-wider text-on-surface">Tông Màu Tơ Tằm:</span>
          <span className="font-label-regular text-label-regular text-primary font-bold">{color.name}</span>
        </div>
        <div className="flex items-center gap-3">
          {p.colors.map((c) => (
            <button
              key={c.hex}
              type="button"
              aria-label={`Chọn màu ${c.name}`}
              aria-pressed={color.hex === c.hex}
              onClick={() => {
                setColor(c)
                showToast(`Đã chọn sắc tơ: ${c.name}`)
              }}
              className={`w-10 h-10 p-0.5 transition-all shadow-sm ${color.hex === c.hex ? 'ring-2 ring-primary' : 'opacity-80 hover:opacity-100'}`}
            >
              <span className="w-full h-full block" style={{ backgroundColor: c.hex }} />
            </button>
          ))}
        </div>
      </div>

      {mode === 'ready' ? (
        <div className="mb-6">
          <div className="flex items-center justify-between mb-2">
            <span className="font-label-uppercase text-label-uppercase tracking-wider text-on-surface">Kích Cỡ Có Sẵn:</span>
            <button
              type="button"
              onClick={onOpenSizeGuide}
              className="font-label-regular text-label-regular text-primary hover:underline flex items-center gap-1"
            >
              <Icon name="help_outline" size={16} />
              Bảng chuẩn kích thước phụ nữ Việt
            </button>
          </div>
          <div className="grid grid-cols-5 gap-2">
            {p.sizes.map((s) => (
              <button
                key={s}
                type="button"
                aria-pressed={size === s}
                onClick={() => setSize(s)}
                className={`py-2.5 font-label-regular text-label-regular transition-colors ${size === s ? 'bg-primary text-on-primary font-bold shadow-sm' : 'bg-surface-container hover:bg-surface-container-high'}`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      ) : (
        <div className="mb-6 bg-surface-container-low p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="font-label-uppercase text-label-uppercase text-primary font-bold tracking-wider flex items-center gap-1.5">
              <Icon name="edit_note" size={18} />
              Phiếu Số Đo Thửa Riêng
            </span>
            <button type="button" onClick={onOpenSizeGuide} className="font-body-sm text-body-sm text-secondary hover:text-primary underline">
              Hướng dẫn tự đo tại nhà
            </button>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mb-3">
            {p.measurementFields.map((f) => (
              <div key={f.id} className="flex flex-col">
                <label htmlFor={`input-${f.id}`} className="font-label-regular text-label-regular text-on-surface-variant mb-1">
                  {f.label}
                </label>
                <input id={`input-${f.id}`} type="number" min={0} className={field} placeholder={f.placeholder} />
              </div>
            ))}
          </div>
          <div className="flex flex-col">
            <label htmlFor="input-notes" className="font-label-regular text-label-regular text-on-surface-variant mb-1">
              Ghi chú vóc dáng đặc biệt cho nghệ nhân
            </label>
            <textarea
              id="input-notes"
              rows={2}
              className="w-full bg-surface-bright p-2.5 text-body-sm font-body-sm text-on-surface focus:outline-none focus:bg-surface-container-lowest shadow-sm"
              placeholder="Ví dụ: bắp tay hơi đầy, thích mặc tà phủ qua gót 3cm, muốn may cổ lập lĩnh cao 3.5cm..."
            />
          </div>
        </div>
      )}

      <div className="flex flex-col gap-3 mb-6">
        <div className="flex items-center gap-3">
          <div className="flex items-center bg-surface-container px-2 py-1 flex-shrink-0">
            <button
              type="button"
              aria-label="Giảm số lượng"
              onClick={() => setQty((q) => Math.max(1, q - 1))}
              className="w-8 h-8 flex items-center justify-center text-on-surface-variant hover:text-primary"
            >
              <Icon name="remove" size={18} />
            </button>
            <span className="w-10 text-center font-label-regular text-label-regular font-bold text-on-surface" aria-live="polite">
              {qty}
            </span>
            <button
              type="button"
              aria-label="Tăng số lượng"
              onClick={() => setQty((q) => q + 1)}
              className="w-8 h-8 flex items-center justify-center text-on-surface-variant hover:text-primary"
            >
              <Icon name="add" size={18} />
            </button>
          </div>
          <button
            type="button"
            onClick={() => showToast('Đã thêm Áo Dài Cổ Phục Ngũ Thân vào giỏ hàng.')}
            className="flex-1 bg-primary hover:bg-primary-container text-on-primary py-3.5 px-6 font-label-uppercase text-label-uppercase tracking-wider shadow-sm transition-colors text-center"
          >
            Thêm Vào Giỏ Hàng
          </button>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            onClick={() => showToast('Chuyên viên tư vấn may đo sẽ liên hệ quý khách trong vòng 15 phút.')}
            className="flex-1 bg-surface-container-high hover:bg-surface-container-highest text-primary py-3.5 px-6 font-label-uppercase text-label-uppercase tracking-wider transition-colors text-center font-bold flex items-center justify-center gap-2"
          >
            <Icon name="support_agent" size={20} />
            Đặt May Đo Ngay (Tư Vấn 1:1)
          </button>
          <button
            type="button"
            aria-label="Lưu vào danh sách yêu thích"
            aria-pressed={wishlisted}
            onClick={toggleWishlist}
            className="w-12 h-12 bg-surface-container hover:bg-surface-container-high text-on-surface flex items-center justify-center transition-colors flex-shrink-0"
          >
            <Icon name={wishlisted ? 'favorite' : 'favorite_border'} size={22} filled={wishlisted} className={wishlisted ? 'text-primary' : ''} />
          </button>
        </div>
      </div>

      <div className="bg-surface-container-low p-4 flex items-start gap-3 text-on-surface-variant">
        <Icon name="local_shipping" size={22} className="text-secondary flex-shrink-0" />
        <div className="flex flex-col font-body-sm text-body-sm">
          <p>
            <strong className="text-on-surface">Giao hàng may sẵn:</strong> 2 - 3 ngày làm việc trên toàn quốc.
          </p>
          <p className="mt-0.5">
            <strong className="text-on-surface">May đo bespoke theo số đo riêng:</strong> Hoàn thiện thủ công tỉ mỉ trong 5 - 7 ngày bởi nghệ nhân kinh nghiệm
            trên 20 năm.
          </p>
        </div>
      </div>
    </div>
  )
}
