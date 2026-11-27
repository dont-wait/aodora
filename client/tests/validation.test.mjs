import { test } from 'node:test'
import assert from 'node:assert/strict'
import { slugify } from '../src/features/admin/slug.ts'
import {
  LIMITS, errorCount, fieldId, formatThousands, foldText, mapServerField, normalizeSku, parseThousands, validateAdjustment, validateCategory, validateNewSku, validateProduct, validateSkuRows,
} from '../src/features/admin/validation.ts'
import { checkImageDimensions, checkImageFile, defaultAlt, moveItem, uploadErrorMessage } from '../src/features/admin/upload.ts'

const img = (src = 'https://cdn.test/a.jpg', alt = 'Áo dài') => ({ src, alt })
const base = () => ({
  name: 'Áo dài lụa', slug: 'ao-dai-lua', category_id: '670000000000000000000001', description: '', material: '', price: 650000, compare_at_price: null,
  is_active: false, image: img(), gallery: [], colors: [], accessories: [],
})
const fieldsOf = (p, ctx) => Object.keys(validateProduct(p, ctx)).sort()

test('slugify: Vietnamese -> ascii kebab-case', () => {
  assert.equal(slugify('Áo dài Lụa Đỏ!'), 'ao-dai-lua-do')
  assert.equal(slugify('  Đầm  midi -- hoa nhí  '), 'dam-midi-hoa-nhi')
  assert.equal(slugify('ĐĂNG KÝ 2026'), 'dang-ky-2026')
  assert.equal(slugify('###'), '')
  assert.equal(slugify('a'.repeat(200)).length, 120)
  assert.equal(slugify('ab-' + 'c'.repeat(200), 4), 'ab-c')
})

test('price formatting and parsing', () => {
  assert.equal(formatThousands(650000), '650.000')
  assert.equal(formatThousands(1000000000), '1.000.000.000')
  assert.equal(formatThousands(999), '999')
  assert.equal(formatThousands(null), '')
  assert.equal(parseThousands('650.000'), 650000)
  assert.equal(parseThousands('650 000 đ'), 650000)
  assert.equal(parseThousands('650000'), 650000)
  assert.equal(parseThousands(''), null)
  assert.ok(Number.isNaN(parseThousands('12abc')))
  assert.ok(Number.isNaN(parseThousands('-5')))
})

test('foldText / normalizeSku match the server', () => {
  assert.equal(foldText('Đỏ  Đô'), 'do do')
  assert.equal(normalizeSku(' áo dài-đỏ m '), 'AO-DAI-DO-M')
})

test('product rules: valid product has no errors', () => {
  assert.deepEqual(validateProduct(base()), {})
})

test('product rules: each limit reports the right field path', () => {
  const long = (n) => 'a'.repeat(n)
  const cases = [
    [{ name: 'Áo' }, ['name']],
    [{ name: '   ' }, ['name']],
    [{ name: long(151) }, ['name']],
    [{ name: long(150) }, []],
    [{ slug: 'Ao Dai' }, ['slug']],
    [{ slug: 'ao--dai' }, ['slug']],
    [{ slug: long(121) }, ['slug']],
    [{ category_id: '' }, ['category_id']],
    [{ price: 999 }, ['price']],
    [{ price: 1000 }, []],
    [{ price: 1_000_000_000 }, []],
    [{ price: 1_000_000_001 }, ['price']],
    [{ price: 1500.5 }, ['price']],
    [{ compare_at_price: 650000 }, ['compare_at_price']],
    [{ compare_at_price: 700000 }, []],
    [{ description: long(5001) }, ['description']],
    [{ material: long(201) }, ['material']],
    [{ image: img('https://x/a.jpg', ' ') }, ['image.alt']],
    [{ image: img('https://x/a.jpg', long(201)) }, ['image.alt']],
    [{ image: img('', 'alt') }, ['image.src']],
  ]
  for (const [patch, want] of cases) assert.deepEqual(fieldsOf({ ...base(), ...patch }), want, JSON.stringify(patch).slice(0, 60))
})

test('gallery rules: max 12, no duplicates, alt required', () => {
  const g = (n) => Array.from({ length: n }, (_, i) => img('https://cdn.test/g' + i + '.jpg', 'alt ' + i))
  assert.deepEqual(fieldsOf({ ...base(), gallery: g(12) }), [])
  assert.deepEqual(fieldsOf({ ...base(), gallery: g(13) }), ['gallery'])
  assert.deepEqual(fieldsOf({ ...base(), gallery: [img('https://cdn.test/g.jpg', 'a'), img('https://cdn.test/g.jpg', 'b')] }), ['gallery[1].src'])
  assert.deepEqual(fieldsOf({ ...base(), gallery: [img(base().image.src, 'again')] }), ['gallery[0].src'])
  assert.deepEqual(fieldsOf({ ...base(), gallery: [img('https://cdn.test/g.jpg', '')] }), ['gallery[0].alt'])
})

test('colour rules: max 20, name 1–50, unique ignoring case/accents, hex', () => {
  assert.deepEqual(fieldsOf({ ...base(), colors: [{ name: 'Đỏ', hex: '#aabbcc' }, { name: 'Xanh' }] }), [])
  assert.deepEqual(fieldsOf({ ...base(), colors: [{ name: ' ' }] }), ['colors[0].name'])
  assert.deepEqual(fieldsOf({ ...base(), colors: [{ name: 'x'.repeat(51) }] }), ['colors[0].name'])
  assert.deepEqual(fieldsOf({ ...base(), colors: [{ name: 'Đỏ' }, { name: 'do' }] }), ['colors[1].name'])
  assert.deepEqual(fieldsOf({ ...base(), colors: [{ name: 'Đỏ' }, { name: 'Xanh', hex: '#12' }] }), ['colors[1].hex'])
  assert.deepEqual(fieldsOf({ ...base(), colors: Array.from({ length: 21 }, (_, i) => ({ name: 'M' + i })) }), ['colors'])
})

test('accessory rules', () => {
  const acc = (patch = {}) => ({ id: 'a1', name: 'Khăn', description: '', price: 0, ...patch })
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc()] }), [])
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc({ name: ' ' })] }), ['accessories[0].name'])
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc({ name: 'x'.repeat(101) })] }), ['accessories[0].name'])
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc({ description: 'x'.repeat(501) })] }), ['accessories[0].description'])
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc({ price: -1 })] }), ['accessories[0].price'])
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc({ price: 100_000_001 })] }), ['accessories[0].price'])
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc(), acc({ name: 'Nón' })] }), ['accessories[1].id'])
  assert.deepEqual(fieldsOf({ ...base(), accessories: [acc({ image: img('https://x/a.jpg', '') })] }), ['accessories[0].image.alt'])
  assert.deepEqual(fieldsOf({ ...base(), accessories: Array.from({ length: 21 }, (_, i) => acc({ id: 'a' + i })) }), ['accessories'])
})

test('"Đang bán" needs a main image, and a SKU for existing products', () => {
  assert.deepEqual(fieldsOf({ ...base(), is_active: true }, { skuCount: 0, isNew: true }), [])
  assert.deepEqual(fieldsOf({ ...base(), is_active: true, image: null }, { skuCount: 0, isNew: true }), ['is_active'])
  assert.deepEqual(fieldsOf({ ...base(), is_active: true }, { skuCount: 0, isNew: false, wasActive: false }), ['is_active'])
  assert.deepEqual(fieldsOf({ ...base(), is_active: true }, { skuCount: 2, isNew: false, wasActive: false }), [])
  // already on sale: saving other fields is never blocked
  assert.deepEqual(fieldsOf({ ...base(), is_active: true, image: null }, { skuCount: 0, isNew: false, wasActive: true }), [])
})

test('category rules', () => {
  const ok = { name: 'Áo dài', slug: 'ao-dai', description: '', sort_order: 0, image: null }
  assert.deepEqual(validateCategory(ok), {})
  assert.deepEqual(Object.keys(validateCategory({ ...ok, name: 'A' })), ['name'])
  assert.deepEqual(Object.keys(validateCategory({ ...ok, name: 'a'.repeat(101) })), ['name'])
  assert.deepEqual(Object.keys(validateCategory({ ...ok, slug: 'Áo' })), ['slug'])
  assert.deepEqual(Object.keys(validateCategory({ ...ok, sort_order: -1 })), ['sort_order'])
  assert.deepEqual(Object.keys(validateCategory({ ...ok, sort_order: 10000 })), ['sort_order'])
  assert.deepEqual(validateCategory({ ...ok, sort_order: 9999 }), {})
  assert.deepEqual(Object.keys(validateCategory({ ...ok, description: 'a'.repeat(2001) })), ['description'])
  assert.deepEqual(Object.keys(validateCategory({ ...ok, image: img('https://x/a.jpg', '') })), ['image.alt'])
})

test('SKU table rules: paths, duplicates, colours, stock', () => {
  const colors = ['Đỏ', 'Xanh']
  const row = (patch = {}) => ({ color: 'Đỏ', size: 'M', sku: 'AO-DO-M', stock: '5', ...patch })
  assert.deepEqual(validateSkuRows([row(), row({ size: 'L', sku: 'AO-DO-L' })], colors), {})
  assert.deepEqual(Object.keys(validateSkuRows([], colors)), ['variants'])
  assert.deepEqual(Object.keys(validateSkuRows([row({ sku: 'X' })], colors)), ['variants[0].sku'])
  assert.deepEqual(Object.keys(validateSkuRows([row(), row({ size: 'L' })], colors)), ['variants[1].sku'])
  assert.deepEqual(Object.keys(validateSkuRows([row(), row({ sku: 'AO-DO-M2' })], colors)), ['variants[1].color'])
  assert.deepEqual(Object.keys(validateSkuRows([row({ color: 'Vàng' })], colors)), ['variants[0].color'])
  assert.deepEqual(Object.keys(validateSkuRows([row({ size: 'A' })], colors)), ['variants[0].size'])
  for (const stock of ['', '-1', '1.5', 'x', '100001']) assert.deepEqual(Object.keys(validateSkuRows([row({ stock })], colors)), ['variants[0].stock'], stock)
  assert.deepEqual(validateSkuRows([row({ stock: '100000', sku: 'ao do m' })], colors), {})
})

test('stock adjustment never goes negative', () => {
  assert.equal(validateAdjustment('+5', 0), '')
  assert.equal(validateAdjustment('-2', 2), '')
  assert.match(validateAdjustment('-3', 2), /còn 2/)
  assert.match(validateAdjustment('0', 5), /khác 0/)
  assert.match(validateAdjustment('abc', 5), /số nguyên/)
  assert.match(validateAdjustment('1.5', 5), /số nguyên/)
  assert.match(validateAdjustment('100001', 5), /tối đa/)
  assert.equal(validateAdjustment('100000', 0), '')
})

test('new SKU form', () => {
  const existing = [{ sku: 'AO-DO-M', color: 'Đỏ', size: 'M' }]
  assert.deepEqual(validateNewSku({ sku: 'ao-do-l', color: 'Đỏ', size: 'L' }, ['Đỏ'], existing), {})
  assert.deepEqual(Object.keys(validateNewSku({ sku: 'AB', color: 'Đỏ', size: 'L' }, ['Đỏ'], existing)), ['sku'])
  assert.deepEqual(Object.keys(validateNewSku({ sku: 'ao do m', color: 'Đỏ', size: 'L' }, ['Đỏ'], existing)), ['sku'])
  assert.deepEqual(Object.keys(validateNewSku({ sku: 'AO-XANH', color: 'Xanh', size: 'L' }, ['Đỏ'], existing)), ['color'])
  assert.deepEqual(Object.keys(validateNewSku({ sku: 'AO-DO-M2', color: 'Đỏ', size: 'M' }, ['Đỏ'], existing)), ['color'])
  assert.deepEqual(Object.keys(validateNewSku({ sku: 'AO-DO-A', color: 'Đỏ', size: 'A' }, ['Đỏ'], existing)), ['size'])
})

test('error helpers: count, element ids, server field mapping', () => {
  assert.equal(errorCount({ a: 'x', b: '', c: 'y' }), 2)
  assert.equal(fieldId('colors[1].hex'), 'f-colors-1-hex')
  assert.equal(fieldId('variants[2].sku'), 'f-variants-2-sku')
  assert.equal(fieldId('price'), 'f-price')
  assert.equal(mapServerField('variants[2].sku'), 'variants[2].sku')
  assert.equal(mapServerField('images'), 'image')
  assert.equal(mapServerField(''), '')
  assert.equal(LIMITS.priceMin, 1000)
})

test('image file checks (client side)', () => {
  const file = (name, type, size = 1000) => ({ name, type, size })
  assert.equal(checkImageFile(file('a.jpg', 'image/jpeg')), '')
  assert.equal(checkImageFile(file('a.PNG', 'image/png')), '')
  assert.equal(checkImageFile(file('a.webp', '')), '')
  assert.match(checkImageFile(file('a.svg', 'image/svg+xml')), /SVG/)
  assert.match(checkImageFile(file('a.gif', 'image/gif')), /GIF/)
  assert.match(checkImageFile(file('a.heic', 'image/heic')), /HEIC/)
  assert.match(checkImageFile(file('a.pdf', 'application/pdf')), /JPG, PNG hoặc WebP/)
  assert.match(checkImageFile(file('a.jpg', 'image/jpeg', 10 * 1024 * 1024 + 1)), /tối đa 10 MB/)
  assert.equal(checkImageFile(file('a.jpg', 'image/jpeg', 10 * 1024 * 1024)), '')
  assert.match(checkImageFile(file('a.jpg', 'image/jpeg', 0)), /trống/)
  assert.equal(checkImageDimensions(400, 400), '')
  assert.equal(checkImageDimensions(8000, 400), '')
  assert.match(checkImageDimensions(399, 900), /quá nhỏ/)
  assert.match(checkImageDimensions(8001, 600), /quá lớn/)
})

test('alt defaults, reordering, upload error messages', () => {
  assert.equal(defaultAlt('Áo dài lụa', 2), 'Áo dài lụa — ảnh 2')
  assert.equal(defaultAlt('Áo dài lụa', null), 'Áo dài lụa')
  assert.equal(defaultAlt('  ', 1), 'Ảnh sản phẩm — ảnh 1')
  const list = ['a', 'b', 'c', 'd']
  assert.deepEqual(moveItem(list, 0, 2), ['b', 'c', 'a', 'd'])
  assert.deepEqual(moveItem(list, 3, 0), ['d', 'a', 'b', 'c'])
  assert.deepEqual(moveItem(list, 1, 1), list)
  assert.deepEqual(moveItem(list, -1, 2), list)
  assert.deepEqual(moveItem(list, 0, 9), list)
  assert.deepEqual(list, ['a', 'b', 'c', 'd'])
  assert.equal(uploadErrorMessage(413), 'Ảnh quá lớn, tối đa 10 MB.')
  assert.match(uploadErrorMessage(503), /Kho lưu ảnh/)
  assert.equal(uploadErrorMessage(400, 'Ảnh quá nhỏ'), 'Ảnh quá nhỏ')
})
