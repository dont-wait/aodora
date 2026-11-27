import { test } from 'node:test'
import assert from 'node:assert/strict'
import { STANDARD_SIZES, isStandardSize, sortSizes } from '../src/features/catalog/sizes.ts'
import {
  buildVariantDrafts, cleanColors, draftsToVariants, isHexColor, parseAccessoriesJson, parseColorNames, skuPart, suggestSku, variantKey,
} from '../src/features/admin/catalog.ts'
import { builderRows, emptyBuilder } from '../src/features/admin/variantBuilder.ts'

test('standard size list and isStandardSize', () => {
  assert.deepEqual([...STANDARD_SIZES], ['XS', 'S', 'M', 'L', 'XL', 'XXL', 'Freesize'])
  for (const size of STANDARD_SIZES) assert.equal(isStandardSize(size), true)
  for (const size of ['A', 'AA', 'xl', '', 'FREESIZE', '32']) assert.equal(isStandardSize(size), false)
})

test('sortSizes: standard order first, unknown sizes keep original order at the end, input untouched', () => {
  const input = ['XL', 'M', 'Freesize', 'EO ÁO 76CM', 'XS', '32', 'S']
  assert.deepEqual(sortSizes(input), ['XS', 'S', 'M', 'XL', 'Freesize', 'EO ÁO 76CM', '32'])
  assert.deepEqual(input, ['XL', 'M', 'Freesize', 'EO ÁO 76CM', 'XS', '32', 'S'])
  assert.deepEqual(sortSizes([]), [])
  assert.deepEqual(sortSizes(['L', 'S', 'M']), ['S', 'M', 'L'])
})

test('SKU suggestion: SLUG-COLOR-SIZE, ASCII upper-case', () => {
  assert.equal(skuPart('Đỏ đô'), 'DO-DO')
  assert.equal(suggestSku('ao-dai-hoa', 'Đỏ đô', 'XL'), 'AO-DAI-HOA-DO-DO-XL')
  assert.equal(suggestSku('ao-dai', 'Xanh', ''), 'AO-DAI-XANH')
  assert.equal(suggestSku('ao-dai', 'Trắng', 'Freesize'), 'AO-DAI-TRANG-FREESIZE')
})

test('parseColorNames: comma/newline, trimmed, unique case-insensitively', () => {
  assert.deepEqual(parseColorNames(' Đỏ, xanh ,\nĐỎ,, Trắng '), ['Đỏ', 'xanh', 'Trắng'])
  assert.deepEqual(parseColorNames('  '), [])
})

test('buildVariantDrafts: colours x chosen sizes in standard order; no sizes => empty size', () => {
  const rows = buildVariantDrafts('ao-dai', ['Đỏ', 'Xanh'], ['L', 'XS'])
  assert.deepEqual(rows.map((r) => [r.color, r.size, r.sku, r.stock]), [
    ['Đỏ', 'XS', 'AO-DAI-DO-XS', '0'], ['Đỏ', 'L', 'AO-DAI-DO-L', '0'],
    ['Xanh', 'XS', 'AO-DAI-XANH-XS', '0'], ['Xanh', 'L', 'AO-DAI-XANH-L', '0'],
  ])
  const none = buildVariantDrafts('ao-dai', ['Đỏ'], [])
  assert.deepEqual(none.map((r) => [r.size, r.sku]), [['', 'AO-DAI-DO']])
  assert.deepEqual(buildVariantDrafts('ao-dai', [], ['M']), [])
})

test('buildVariantDrafts keeps hand edits when regenerated; untouched SKUs follow the slug', () => {
  const edits = { [variantKey('Đỏ', 'M')]: { sku: 'CUSTOM-1', stock: '7' } }
  const rows = buildVariantDrafts('moi', ['Đỏ', 'Xanh'], ['M'], edits)
  assert.deepEqual(rows.map((r) => [r.sku, r.stock]), [['CUSTOM-1', '7'], ['MOI-XANH-M', '0']])
})

test('builderRows: empty builder has no rows; draftsToVariants validates', () => {
  assert.deepEqual(builderRows('x', emptyBuilder), [])
  const rows = builderRows('ao', { sizes: ['S'], colors: ['Đỏ'], edits: { [variantKey('Đỏ', 'S')]: { stock: '12' } } })
  // colours that are no longer in the product's colour list are dropped
  assert.deepEqual(builderRows('ao', { sizes: ['S'], colors: ['Đỏ', 'Xanh'], edits: {} }, ['đỏ']).map((r) => r.color), ['Đỏ'])
  assert.deepEqual(draftsToVariants(rows), [{ sku: 'AO-DO-S', color: 'Đỏ', size: 'S', stock: 12 }])
  assert.throws(() => draftsToVariants([]), /ít nhất một màu/)
  assert.throws(() => draftsToVariants([{ key: 'k', color: 'Đỏ', size: 'S', sku: ' ', stock: '1' }]), /SKU/)
  assert.throws(() => draftsToVariants([{ key: 'k', color: 'Đỏ', size: 'S', sku: 'A', stock: '-1' }]), /không âm/)
  assert.throws(() => draftsToVariants([{ key: 'k', color: 'Đỏ', size: 'S', sku: 'A', stock: '' }]), /không âm/)
  assert.throws(() => draftsToVariants([{ key: 'k', color: 'Đỏ', size: 'S', sku: 'A', stock: '1.5' }]), /không âm/)
})

test('colours form: hex validation and blank rows dropped', () => {
  assert.equal(isHexColor('#A1b2C3'), true)
  assert.equal(isHexColor('A1b2C3'), false)
  assert.equal(isHexColor('#abc'), false)
  assert.deepEqual(cleanColors([{ name: ' Đỏ ', hex: '#ff0000' }, { name: '  ', hex: '#000000' }, { name: 'Trắng', hex: '' }, { name: 'Xanh' }]), [
    { name: 'Đỏ', hex: '#ff0000' }, { name: 'Trắng' }, { name: 'Xanh' },
  ])
  assert.throws(() => cleanColors([{ name: 'Đỏ', hex: 'red' }]), /#RRGGBB/)
})

test('accessories JSON: clear error messages', () => {
  assert.deepEqual(parseAccessoriesJson(''), [])
  assert.deepEqual(parseAccessoriesJson('[{"id":"a"}]'), [{ id: 'a' }])
  assert.throws(() => parseAccessoriesJson('[{'), /Phụ kiện: JSON không hợp lệ/)
  assert.throws(() => parseAccessoriesJson('{}'), /mảng JSON/)
})
