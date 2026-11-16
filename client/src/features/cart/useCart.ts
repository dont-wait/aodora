import { useContext } from 'react'
import { CartContext } from './context'

export function useCart() {
  const value = useContext(CartContext)
  if (!value) throw new Error('useCart phải được dùng bên trong CartProvider')
  return value
}
