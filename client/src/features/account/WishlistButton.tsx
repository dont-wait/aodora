import { useNavigate, useLocation } from 'react-router-dom'
import type { MouseEvent } from 'react'
import Icon from '@/components/ui/Icon'
import { useAuth } from '@/features/auth/useAuth'
import { useToast } from '@/hooks/useToast'
import { useWishlist } from './useWishlist'

export default function WishlistButton({ productId, className = '' }: { productId: string; className?: string }) {
  const { user } = useAuth()
  const { isSaved, toggle, loading } = useWishlist()
  const { showToast } = useToast()
  const navigate = useNavigate()
  const location = useLocation()
  const saved = isSaved(productId)

  async function click(event: MouseEvent<HTMLButtonElement>) {
    event.preventDefault()
    event.stopPropagation()
    if (!user) {
      navigate(`/dang-nhap?next=${encodeURIComponent(location.pathname + location.search)}`)
      return
    }
    try { await toggle(productId) }
    catch (error) { showToast(error instanceof Error ? error.message : 'Không thể cập nhật yêu thích.') }
  }

  return <button type="button" aria-label={saved ? 'Bỏ khỏi yêu thích' : 'Thêm vào yêu thích'} aria-pressed={saved} disabled={loading && !!user} onClick={(event) => void click(event)} className={`inline-flex items-center justify-center disabled:opacity-50 ${className}`}>
    <Icon name="favorite" size={20} filled={saved} />
  </button>
}
