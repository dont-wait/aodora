import { useEffect, useState, type FormEvent } from 'react'
import { readOrderToken } from '@/features/orders/checkout'
import type { Order } from '@/features/orders/types'
import ReviewFields, { type ReviewDraft } from './ReviewFields'
import { formatReviewImages, guestReviewApi, parseReviewImages, reviewErrorMessage, type OwnReview } from './api'

type Line = { productId: string; name: string }
const emptyDraft: ReviewDraft = { rating: 5, content: '', imagesText: '' }

function productLines(order: Order): Line[] {
  const seen = new Set<string>()
  return order.items.filter((item) => !seen.has(item.product_id) && seen.add(item.product_id)).map((item) => ({ productId: item.product_id, name: item.product_name }))
}

function draftFrom(review: OwnReview | undefined): ReviewDraft {
  return review ? { rating: review.rating, content: review.content || '', imagesText: formatReviewImages(review.images) } : emptyDraft
}

/**
 * Review form for a delivered guest order. Access comes only from the order
 * token stored on this browser at checkout; the server re-checks ownership,
 * delivery and that each product is in the order.
 */
export default function GuestOrderReviews({ order }: { order: Order }) {
  const [token] = useState(() => {
    try { return readOrderToken(order.id) } catch { return null }
  })
  const [reviews, setReviews] = useState<Record<string, OwnReview>>({})
  const [loadError, setLoadError] = useState('')
  const lines = productLines(order)

  useEffect(() => {
    if (!token) return
    let active = true
    guestReviewApi.list(order.id, token)
      .then((items) => { if (active) setReviews(Object.fromEntries(items.map((item) => [item.product_id, item]))) })
      .catch((error: unknown) => { if (active) setLoadError(reviewErrorMessage(error)) })
    return () => { active = false }
  }, [order.id, token])

  if (!token || order.status !== 'delivered') return null
  return (
    <section aria-labelledby="guest-review-title" className="border border-outline-variant p-5 space-y-6">
      <div>
        <p className="text-secondary text-sm uppercase tracking-widest">Trải nghiệm của bạn</p>
        <h2 id="guest-review-title" className="font-headline-md text-2xl text-primary mt-1">Đánh giá sản phẩm đã nhận</h2>
        <p className="text-sm text-on-surface-variant mt-2">Mỗi mẫu trong đơn có một đánh giá; gửi lại sẽ cập nhật đánh giá cũ.</p>
      </div>
      {loadError && <p role="alert" className="text-red-700">{loadError}</p>}
      {lines.map((line) => (
        <GuestReviewForm key={line.productId} orderId={order.id} token={token} line={line} existing={reviews[line.productId]} onSaved={(saved) => setReviews((current) => ({ ...current, [line.productId]: saved }))} />
      ))}
    </section>
  )
}

function GuestReviewForm({ orderId, token, line, existing, onSaved }: { orderId: string; token: string; line: Line; existing?: OwnReview; onSaved: (review: OwnReview) => void }) {
  const [draft, setDraft] = useState<ReviewDraft | null>(null)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ kind: 'error' | 'ok'; text: string } | null>(null)
  const value = draft ?? draftFrom(existing)

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true); setMessage(null)
    try {
      const saved = await guestReviewApi.submit(orderId, line.productId, token, { rating: value.rating, content: value.content, images: parseReviewImages(value.imagesText) })
      onSaved(saved); setDraft(null)
      setMessage({ kind: 'ok', text: 'Đã lưu đánh giá.' })
    } catch (error) {
      setMessage({ kind: 'error', text: reviewErrorMessage(error) })
    } finally { setSaving(false) }
  }

  return (
    <form onSubmit={submit} className="border-t border-outline-variant/50 pt-5">
      <h3 className="font-semibold text-primary">{line.name}</h3>
      <ReviewFields idPrefix={`guest-${line.productId}`} value={value} onChange={setDraft} />
      {message && <p role={message.kind === 'error' ? 'alert' : 'status'} className={`mt-4 ${message.kind === 'error' ? 'text-red-700' : 'text-secondary'}`}>{message.text}</p>}
      <button disabled={saving} type="submit" className="mt-5 bg-primary text-on-primary px-6 py-3 disabled:opacity-50">{saving ? 'Đang lưu...' : existing ? 'Cập nhật đánh giá' : 'Gửi đánh giá'}</button>
    </form>
  )
}
