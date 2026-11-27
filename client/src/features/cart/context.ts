import { createContext } from 'react'
import type { CartData, CartLine } from './api'
import type { PurchaseLine } from '@/features/orders/types'
import type { CartSelection } from '@/features/catalog/selection'

export type CartContextValue = {
  cart: CartData
  loading: boolean
  error: string
  itemCount: number
  /** Lines ticked for checkout (default: all). */
  selectedItems: CartLine[]
  selectedSubtotal: number
  isSelected: (line: Pick<CartLine, 'product_id' | 'sku'>) => boolean
  setSelected: (lines: Pick<CartLine, 'product_id' | 'sku'>[], selected: boolean) => void
  add: (selection: CartSelection) => Promise<void>
  update: (selection: CartSelection) => Promise<void>
  remove: (productId: string, sku: string) => Promise<void>
  refresh: () => Promise<void>
  consumePurchased: (snapshot: PurchaseLine[], ownerId: string | null) => Promise<void>
}

export const CartContext = createContext<CartContextValue | null>(null)
