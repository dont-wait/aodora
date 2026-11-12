import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createCartSelection, registerCartAddHandler, addSelectedProduct } from '../src/features/catalog/selection.ts'
import { getCategories, getProduct, listProducts, CatalogApiError } from '../src/features/catalog/api.ts'
import { parseVNDInput, formatVNDInput } from '../src/features/catalog/prices.ts'

const product = {
  id: '507f1f77bcf86cd799439011', slug: 'ao-mau', stock: 6, sku: '', ready_available: true,
  variants: [
    { sku: 'RED-M', color: 'Đỏ', size: 'M', stock: 4, available: true },
    { sku: 'BLUE-L', color: 'Xanh', size: 'L', stock: 2, available: true },
    { sku: 'BLUE-M', color: 'Xanh', size: 'M', stock: 0, available: false },
  ],
}

test('chuyển đúng SKU, màu, cỡ và số lượng sang KR-03', async () => {
  const selection = createCartSelection(product, 'Đỏ', 'M', 2)
  assert.deepEqual(selection, {
    product_id: product.id, product_slug: product.slug,
    sku: 'RED-M', color: 'Đỏ', size: 'M', quantity: 2, fulfillment_mode: 'ready',
  })
  let received
  const unregister = registerCartAddHandler(async (data) => { received = data })
  await addSelectedProduct(selection)
  assert.deepEqual(received, selection)
  unregister()
})

test('không ghép sai màu/cỡ, không chấp nhận hàng hết hoặc số lượng vượt tồn', () => {
  assert.throws(() => createCartSelection(product, 'Đỏ', 'L', 1))
  assert.throws(() => createCartSelection(product, 'Xanh', 'M', 1))
  assert.throws(() => createCartSelection(product, 'Xanh', 'L', 3))
  assert.throws(() => createCartSelection(product, 'Đỏ', 'M', 0))
})

test('chưa có KR-03 thì không báo thêm giỏ thành công', async () => {
  await assert.rejects(addSelectedProduct(createCartSelection(product, 'Đỏ', 'M', 1)), /chưa được tích hợp/)
})

test('đọc sản phẩm theo slug và truyền bộ lọc đến API thực', async () => {
  const previous = globalThis.fetch
  const called = []
  globalThis.fetch = async (url) => {
    called.push(url)
    if (url.includes('/categories')) return { ok: true, status: 200, json: async () => ({ data: [{ id: 'cat-1', slug: 'co-phuc', name: 'Cổ phục' }] }) }
    if (url.includes('/products/')) return { ok: true, status: 200, json: async () => ({ data: product }) }
    return { ok: true, status: 200, json: async () => ({ data: [], total: 0, page: 2, page_size: 12, total_pages: 0 }) }
  }
  try {
    assert.equal((await getCategories())[0].slug, 'co-phuc')
    assert.equal((await getProduct('ao-mau')).slug, 'ao-mau')
    assert.equal((await listProducts({ search: 'lụa tơ', category: 'co-phuc', page: 2 })).page, 2)
    assert.ok(called[2].includes('search=l%E1%BB%A5a+t%C6%A1'))
    assert.ok(called[2].includes('category=co-phuc'))
    assert.ok(called[2].includes('page=2'))
  } finally { globalThis.fetch = previous }
})

test('API 404 không trả sản phẩm khác thay cho slug bị thiếu', async () => {
  const previous = globalThis.fetch
  globalThis.fetch = async () => ({ ok: false, status: 404, json: async () => ({ error: { code: 'PRODUCT_NOT_FOUND', message: 'Không tìm thấy mẫu áo.' } }) })
  try {
    await assert.rejects(getProduct('khong-ton-tai'), (err) => err instanceof CatalogApiError && err.status === 404 && err.code === 'PRODUCT_NOT_FOUND')
  } finally { globalThis.fetch = previous }
})

test('định dạng tiền VNĐ không tự nhân 1000 và không chấp nhận dữ liệu sai', () => {
  assert.equal(parseVNDInput('400.000'), '400000')
  assert.equal(parseVNDInput('500,000'), '500000')
  assert.equal(parseVNDInput('400'), '400')
  assert.equal(formatVNDInput('400000'), '400.000')
  assert.equal(parseVNDInput('400k'), null)
  assert.equal(parseVNDInput('4.00'), null)
  assert.equal(parseVNDInput('-100'), null)
  assert.equal(parseVNDInput(''), '')
})
