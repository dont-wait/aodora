import Icon from '@/components/ui/Icon'

export type ReviewDraft = { rating: number; content: string; imagesText: string }

const input = 'w-full bg-surface-container px-4 py-3 border border-outline-variant/60 focus:border-primary focus:outline-none'

/** Rating, comment and image fields shared by account and guest review forms. */
export default function ReviewFields({ idPrefix, value, onChange }: { idPrefix: string; value: ReviewDraft; onChange: (value: ReviewDraft) => void }) {
  return (
    <>
      <div className="mt-5">
        <span className="block text-sm text-on-surface-variant mb-2">Điểm đánh giá</span>
        <div className="flex gap-1">
          {[1, 2, 3, 4, 5].map((rating) => (
            <button key={rating} type="button" aria-label={`${rating} sao`} aria-pressed={value.rating === rating} onClick={() => onChange({ ...value, rating })} className={`w-10 h-10 inline-flex items-center justify-center ${value.rating >= rating ? 'text-secondary bg-secondary-fixed' : 'text-outline bg-surface-container'}`}>
              <Icon name="star" filled={value.rating >= rating} size={18} />
            </button>
          ))}
        </div>
      </div>
      <label className="block space-y-2 mt-5" htmlFor={`${idPrefix}-content`}>
        <span className="text-sm text-on-surface-variant">Nhận xét</span>
        <textarea id={`${idPrefix}-content`} rows={4} maxLength={3000} value={value.content} onChange={(event) => onChange({ ...value, content: event.target.value })} className={input} placeholder="Chia sẻ cảm nhận của bạn về mẫu áo..." />
      </label>
      <label className="block space-y-2 mt-5" htmlFor={`${idPrefix}-images`}>
        <span className="text-sm text-on-surface-variant">Ảnh (mỗi dòng URL | mô tả, không bắt buộc)</span>
        <textarea id={`${idPrefix}-images`} rows={2} value={value.imagesText} onChange={(event) => onChange({ ...value, imagesText: event.target.value })} className={input} placeholder="https://... | Ảnh áo sau khi nhận" />
      </label>
    </>
  )
}
