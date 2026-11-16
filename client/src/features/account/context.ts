import { createContext } from 'react'
import type { WishlistItem } from './api'

export type WishlistContextValue = {
  items: WishlistItem[]
  loading: boolean
  error: string
  refresh: () => Promise<void>
  isSaved: (productId: string) => boolean
  toggle: (productId: string) => Promise<void>
}

export const WishlistContext = createContext<WishlistContextValue | null>(null)
