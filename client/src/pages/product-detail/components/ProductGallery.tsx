import { useState } from 'react'
import Icon from '@/components/ui/Icon'
import { productDetail } from '@/data/productDetail'
import { useToast } from '@/hooks/useToast'

export default function ProductGallery() {
  const { showToast } = useToast()
  const { mainImage, gallery, badge, trust } = productDetail
  const images = gallery
  const [active, setActive] = useState(0)
  // Ảnh lớn ban đầu là ảnh chính; sau khi chọn thumbnail sẽ hiển thị ảnh tương ứng.
  const [picked, setPicked] = useState(false)
  const current = picked ? images[active] : mainImage

  return (
    <div className="lg:col-span-7 flex flex-col gap-6">
      <div className="relative bg-surface-container aspect-[3/4] w-full overflow-hidden shadow-sm group">
        <img
          src={current.src}
          alt={current.alt}
          className="w-full h-full object-cover object-top transition-transform duration-700 ease-out group-hover:scale-105"
        />
        <div className="absolute top-4 left-4 bg-primary text-on-primary px-3 py-1 text-label-uppercase font-label-uppercase tracking-widest">{badge}</div>
        <button
          type="button"
          aria-label="Phóng to hình ảnh"
          onClick={() => showToast('Nhấp đúp chuột để phóng to chi tiết thớ lụa dệt hoa văn.')}
          className="absolute bottom-4 right-4 bg-surface/90 hover:bg-surface text-on-surface p-2.5 backdrop-blur shadow-sm transition-all"
        >
          <Icon name="zoom_in" size={20} />
        </button>
      </div>

      <div className="grid grid-cols-4 gap-3 sm:gap-4">
        {images.map((img, i) => (
          <button
            key={img.src}
            type="button"
            aria-label={img.alt}
            aria-pressed={picked && active === i}
            onClick={() => {
              setActive(i)
              setPicked(true)
            }}
            className={`aspect-square bg-surface-container overflow-hidden p-1 transition-all focus:outline-none ${(picked ? active === i : i === 0) ? 'shadow-sm ring-2 ring-primary' : 'opacity-70 hover:opacity-100'}`}
          >
            <img src={img.src} alt={img.alt} className="w-full h-full object-cover object-top" />
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 bg-surface-container-low p-5 mt-2">
        {trust.map((t) => (
          <div key={t.title} className="flex items-start gap-3">
            <Icon name={t.icon} size={26} className="text-secondary flex-shrink-0" />
            <div className="flex flex-col">
              <span className="font-label-regular text-label-regular font-bold text-on-surface">{t.title}</span>
              <span className="font-body-sm text-body-sm text-on-surface-variant">{t.description}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
