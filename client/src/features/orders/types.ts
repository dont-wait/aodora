export type Money = string
export type PaymentMethod = 'cod' | 'cash_at_counter'
export type OrderCredential = { kind: 'account'; customerId: string } | { kind: 'guest'; token: string }
export type PurchaseLine = { product_id: string; sku: string; quantity: number; name?: string }
export type ShippingAddress = { recipient: string; phone: string; line1?: string; ward?: string; /** chỉ đọc dữ liệu trước 07/2025 */ district?: string; city?: string }
export type CheckoutBody = { payment_method: PaymentMethod; shipping_address: ShippingAddress; items: PurchaseLine[]; note?: string; quote_token?: string }
export type QuoteItem = PurchaseLine & { unit_price: Money }
export type Quote = { quote_token: string; expires_at: string; subtotal: Money; shipping_fee: Money; total: Money; items: QuoteItem[] }
export type Order = {
  id: string
  order_number: string
  status: string
  payment_method: string
  payment_status: string
  subtotal: Money
  shipping_fee: Money
  total: Money
  shipping_address: ShippingAddress
  items: (QuoteItem & { product_name: string; color?: string; size?: string })[]
  note?: string
  created_at: string
}
export type OrderPage = { data: Order[]; total: number; page: number; page_size: number }
export type CheckoutAttempt = {
  version: 1
  request_id: string
  credential: OrderCredential
  body: string
  snapshot: PurchaseLine[]
  /** When the quote inside body stops being accepted; absent on attempts saved before this field existed. */
  quote_expires_at?: string
}
export type OrderReceipt = {
  order_id: string
  order_number: string
  request_id: string
  owner: string | null
  cleanup: 'pending' | 'started' | 'done' | 'failed'
}
