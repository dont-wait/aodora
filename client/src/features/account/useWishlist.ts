import { useContext } from 'react'
import { WishlistContext } from './context'

export function useWishlist() {
  const value = useContext(WishlistContext)
  if (!value) throw new Error('useWishlist phải được dùng bên trong WishlistProvider')
  return value
}
