import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { searchOptions, foldName, findProvince, findWard, formatAddress, matchSavedAddress, normalizeName } from '../src/features/address/divisions.ts'
import { orderDeliveryAddress } from '../src/features/orders/presentation.ts'

const data = JSON.parse(readFileSync(new URL('../src/data/vn-divisions-2025.json', import.meta.url), 'utf8'))
const list = data.provinces

test('normalizeName strips diacritics, prefixes and extra spaces', () => {
  assert.equal(normalizeName('TP. Hồ Chí Minh'), normalizeName('Thành phố Hồ Chí Minh'))
  assert.equal(normalizeName('  Tỉnh   Đồng Nai '), 'dong nai')
  assert.equal(normalizeName('Phường Ba Đình'), 'ba dinh')
  assert.equal(normalizeName('Xã Tân Phú'), 'tan phu')
  assert.equal(normalizeName('Đặc khu Phú Quý'), 'phu quy')
})

test('dataset has 34 provinces and 3321 wards', () => {
  assert.equal(list.length, 34)
  assert.equal(list.reduce((sum, province) => sum + province.wards.length, 0), 3321)
})

test('ward names are unique inside each province after normalization (report only)', () => {
  const duplicates = []
  for (const province of list) {
    const seen = new Map()
    for (const ward of province.wards) {
      const key = normalizeName(ward.name)
      if (seen.has(key)) duplicates.push(`${province.name}: ${seen.get(key)} / ${ward.name}`)
      else seen.set(key, ward.name)
    }
  }
  if (duplicates.length) console.log('DUPLICATE_WARDS', JSON.stringify(duplicates))
  assert.ok(Array.isArray(duplicates))
})

test('findProvince / findWard match exactly after normalization', () => {
  const hcm = findProvince(list, 'TP. Hồ Chí Minh')
  assert.ok(hcm)
  assert.equal(hcm.name, 'Thành phố Hồ Chí Minh')
  assert.equal(findProvince(list, 'Hồ Chí Min'), undefined)
  assert.equal(findProvince(list, ''), undefined)
  const hanoi = findProvince(list, 'Hà Nội')
  assert.equal(findWard(hanoi, 'phường ba đình')?.name, 'Phường Ba Đình')
  assert.equal(findWard(hanoi, 'Ba Dinh')?.name, 'Phường Ba Đình')
  assert.equal(findWard(hanoi, 'Không có'), undefined)
  assert.equal(findWard(undefined, 'Ba Đình'), undefined)
})

test('ambiguous normalized names are not guessed; exact full names still resolve', () => {
  const thaiNguyen = list.find((province) => province.name === 'Tỉnh Thái Nguyên')
  assert.equal(findWard(thaiNguyen, 'Xã Văn Lang')?.name, 'Xã Văn Lang')
  assert.equal(findWard(thaiNguyen, 'Xã Văn Lăng')?.name, 'Xã Văn Lăng')
  assert.equal(findWard(thaiNguyen, 'Van Lang'), undefined)
})

test('matchSavedAddress flags legacy data', () => {
  const ok = matchSavedAddress(list, { city: 'Thành phố Hà Nội', ward: 'Phường Ba Đình' })
  assert.equal(ok.legacy, false)
  assert.equal(ok.province?.name, 'Thành phố Hà Nội')
  assert.equal(ok.ward?.name, 'Phường Ba Đình')
  assert.equal(matchSavedAddress(list, { city: 'Hà Nội', ward: 'Phường Ba Đình', district: 'Quận Ba Đình' }).legacy, true)
  const merged = matchSavedAddress(list, { city: 'Tỉnh Bình Dương', ward: 'Phường 1' })
  assert.equal(merged.legacy, true)
  assert.equal(merged.province, undefined)
  assert.equal(matchSavedAddress(list, { city: 'Hà Nội', ward: 'Phường 99999' }).legacy, true)
  assert.equal(matchSavedAddress(list, {}).legacy, false)
  assert.equal(matchSavedAddress(list, { city: 'Hà Nội', district: '  ' }).legacy, false)
})

test('formatAddress skips empty parts and keeps legacy district', () => {
  assert.equal(formatAddress({ line1: '12 Nguyễn Trãi', ward: '', city: 'Thành phố Hà Nội' }), '12 Nguyễn Trãi, Thành phố Hà Nội')
  assert.equal(formatAddress({ line1: '1 A', ward: 'P1', district: 'Quận 3', city: 'TP.HCM' }), '1 A, P1, Quận 3, TP.HCM')
  assert.equal(formatAddress({}), '')
})

test('orderDeliveryAddress uses formatAddress and only shows for COD', () => {
  const shipping_address = { recipient: 'A', phone: '1', line1: '1 A', ward: 'P1', district: 'Quận 3', city: 'HCM' }
  assert.equal(orderDeliveryAddress({ payment_method: 'cod', shipping_address }), '1 A, P1, Quận 3, HCM')
  assert.equal(orderDeliveryAddress({ payment_method: 'cash_at_counter', shipping_address }), null)
})

test('searchOptions: no diacritics, case-insensitive, substring, prefix-agnostic', () => {
  const hcm = list.find((province) => province.name === 'Thành phố Hồ Chí Minh')
  const options = hcm.wards.map((ward) => ({ value: ward.name, label: ward.name }))
  for (const query of ['tan son nhat', 'Tân Sơn Nhất', 'TAN SON NHAT', 'phuong tan son nhat', 'Phường Tân Sơn']) {
    assert.ok(searchOptions(options, query).some((option) => option.label === 'Phường Tân Sơn Nhất'), query)
  }
  assert.equal(searchOptions(options, 'zzzz không có').length, 0)
})

test('searchOptions: empty query returns all; prefix-start results come first', () => {
  const options = [{ label: 'Phường Hòa Bình' }, { label: 'Xã Bình An' }, { label: 'Phường An Bình' }, { label: 'Xã Bình Tân' }]
  assert.equal(searchOptions(options, '  ').length, 4)
  assert.deepEqual(searchOptions(options, 'binh').map((option) => option.label), ['Xã Bình An', 'Xã Bình Tân', 'Phường Hòa Bình', 'Phường An Bình'])
  assert.deepEqual(searchOptions(options, 'phuong an').map((option) => option.label), ['Phường An Bình'])
})

test('searchOptions finds provinces without diacritics; foldName keeps prefix', () => {
  const options = list.map((province) => ({ label: province.name }))
  assert.equal(searchOptions(options, 'ho chi minh')[0].label, 'Thành phố Hồ Chí Minh')
  assert.equal(searchOptions(options, 'ha noi')[0].label, 'Thành phố Hà Nội')
  assert.equal(foldName('Thành phố Đà Nẵng'), 'thanh pho da nang')
})
