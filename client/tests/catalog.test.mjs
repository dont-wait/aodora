import { test } from 'node:test'
import assert from 'node:assert/strict'
import { createCartSelection, registerCartAddHandler, addSelectedProduct } from '../src/features/catalog/selection.ts'
import { getCategories, getProduct, listProducts, CatalogApiError } from '../src/features/catalog/api.ts'
import { parseVNDInput, formatVNDInput } from '../src/features/catalog/prices.ts'
import { productGallery } from '../src/features/catalog/gallery.ts'
import { formatMoney, maskMoneyInput } from '../src/features/catalog/prices.ts'

test('gallery gộp URL legacy và RustFS, bỏ trùng và giữ alt của ảnh chính', () => {
  const primary = { src: 'http://localhost:9100/aodora-images/main.jpg', alt: 'Ảnh chính' }
  assert.deepEqual(productGallery({
    name: 'Áo mẫu', image: primary,
    gallery: [{ src: primary.src, alt: 'Trùng' }, { src: '/detail.jpg', alt: 'Chi tiết' }],
    images: [primary.src, '/legacy.jpg', '/detail.jpg', ''],
  }), [primary, { src: '/detail.jpg', alt: 'Chi tiết' }, { src: '/legacy.jpg', alt: 'Áo mẫu' }])
  assert.deepEqual(productGallery({ name: 'Áo mẫu', image: null, gallery: [], images: ['/a.jpg', '/b.jpg'] }),
    [{ src: '/a.jpg', alt: 'Áo mẫu' }, { src: '/b.jpg', alt: 'Áo mẫu' }])
})

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

test('money input adds thousands dots while typing and keeps the caret', () => {
  assert.equal(formatMoney(1250000), '1.250.000')
  assert.equal(formatMoney(null), '')
  // typing 4,0,0,0,0,0 one key at a time
  let text = ''
  for (const key of '400000') text = maskMoneyInput(text + key).text
  assert.equal(text, '400.000')
  // the user types a dot themselves: ignored, still the same number
  assert.deepEqual(maskMoneyInput('400.000.'), { text: '400.000', caret: 7, value: 400000 })
  // insert a 1 in front of "500.000" (caret after the typed digit)
  assert.deepEqual(maskMoneyInput('1500.000', 1), { text: '1.500.000', caret: 1, value: 1500000 })
  // delete the dot before 000 in "1.500.000" -> digits unchanged, caret stays near
  assert.equal(maskMoneyInput('1.500000', 5).value, 1500000)
  assert.deepEqual(maskMoneyInput(''), { text: '', caret: 0, value: null })
  assert.equal(maskMoneyInput('007').value, 7)
})

test('money input understands k / tr shortcuts', () => {
  assert.equal(maskMoneyInput('800k').value, 800000)
  assert.equal(maskMoneyInput('2tr').value, 2000000)
  assert.equal(maskMoneyInput('2 triệu').value, 2000000)
  assert.equal(maskMoneyInput('3m').text, '3.000.000')
})
