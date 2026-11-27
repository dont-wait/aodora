import { test } from 'node:test'
import assert from 'node:assert/strict'
import { cartLineKey, readUnselected, selectedLines, selectedSubtotal, writeUnselected } from '../src/features/cart/selection.ts'
import { subtractPurchased } from '../src/features/orders/checkout.ts'

const memory = () => {
  const data = new Map()
  return { getItem: (k) => (data.has(k) ? data.get(k) : null), setItem: (k, v) => data.set(k, String(v)), removeItem: (k) => data.delete(k), data }
}
const line = (product_id, sku, quantity = 1, line_total = 100) => ({ product_id, sku, quantity, line_total })

test('lines are selected by default; unticked keys persist and reload', () => {
  const storage = memory()
  const items = [line('a', 'A-M', 1, 300), line('b', 'B-M', 2, 500), line('c', 'C-M', 1, 200)]
  assert.deepEqual(selectedLines(items, readUnselected(storage)), items)
  writeUnselected(new Set([cartLineKey(items[1])]), storage)
  const unselected = readUnselected(storage)
  assert.deepEqual(selectedLines(items, unselected).map((item) => item.sku), ['A-M', 'C-M'])
  assert.equal(selectedSubtotal(selectedLines(items, unselected)), 500)
  writeUnselected(new Set(), storage)
  assert.equal(storage.data.size, 0)
})

test('corrupt selection storage falls back to everything selected', () => {
  const storage = memory()
  storage.setItem('aodora.cart.unselected.v1', '{not json')
  assert.equal(readUnselected(storage).size, 0)
  storage.setItem('aodora.cart.unselected.v1', JSON.stringify([1, null, '["a","A-M"]']))
  assert.deepEqual([...readUnselected(storage)], ['["a","A-M"]'])
})

test('checking out the selected lines keeps the unselected ones in the cart', () => {
  const cart = [line('a', 'A-M', 2), line('b', 'B-M', 1)]
  const bought = selectedLines(cart, new Set([cartLineKey(cart[1])])).map(({ product_id, sku, quantity }) => ({ product_id, sku, quantity }))
  assert.deepEqual(subtractPurchased(cart, bought).map((item) => [item.sku, item.quantity]), [['B-M', 1]])
})
