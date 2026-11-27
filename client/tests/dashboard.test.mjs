import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  defaultGranularity, formatBucketLabel, formatChange, formatPercent, formatVND, formatVNDShort, isoDate, niceScale, presetRange, rangeDays, scaleLinear, statusLabel, vietnamToday,
} from '../src/features/admin/dashboard.ts'
import {
  canCommit, commitLabel, errorRows, fileProblem, parseImportReport, templateCsv, templateJson, toCsv, IMPORT_MAX_BYTES,
} from '../src/features/admin/import.ts'

test('formatVNDShort uses k / tr / tỷ with a decimal comma', () => {
  assert.equal(formatVNDShort(0), '0')
  assert.equal(formatVNDShort(950), '950')
  assert.equal(formatVNDShort(1000), '1k')
  assert.equal(formatVNDShort(850000), '850k')
  assert.equal(formatVNDShort(1500000), '1,5 tr')
  assert.equal(formatVNDShort(2000000), '2 tr')
  assert.equal(formatVNDShort(1250000000), '1,3 tỷ')
  assert.equal(formatVNDShort(-2500000), '-2,5 tr')
})

test('formatVND, formatChange, formatPercent', () => {
  assert.equal(formatVND(1500000), '1.500.000 ₫')
  assert.equal(formatVND(999), '999 ₫')
  assert.equal(formatChange(12.34), '+12,3%')
  assert.equal(formatChange(-4), '-4,0%')
  assert.equal(formatChange(0), '0,0%')
  assert.equal(formatChange(null), '—')
  assert.equal(formatChange(Number.NaN), '—')
  assert.equal(formatPercent(0.2), '20,0%')
  assert.equal(formatPercent(0.0123), '1,2%')
})

test('niceScale: round maximum and evenly spaced ticks from 0', () => {
  assert.deepEqual(niceScale(0), { max: 4, ticks: [0, 1, 2, 3, 4] })
  assert.deepEqual(niceScale(87), { max: 100, ticks: [0, 25, 50, 75, 100] })
  assert.deepEqual(niceScale(1_234_000), { max: 1_500_000, ticks: [0, 500000, 1000000, 1500000] })
  const { max, ticks } = niceScale(4200)
  assert.ok(max >= 4200 && ticks[0] === 0 && ticks.at(-1) === max)
  for (const v of [1, 7, 99, 100, 101, 5555, 12_345_678]) {
    const s = niceScale(v)
    assert.ok(s.max >= v, 'max covers data for ' + v)
    assert.ok(s.ticks.length >= 3 && s.ticks.length <= 7, 'tick count for ' + v + ': ' + s.ticks.length)
  }
})

test('scaleLinear maps domain to range (also inverted for SVG y)', () => {
  const x = scaleLinear(0, 10, 100, 200)
  assert.equal(x(0), 100)
  assert.equal(x(5), 150)
  assert.equal(x(10), 200)
  const y = scaleLinear(0, 100, 300, 20)
  assert.equal(y(0), 300)
  assert.equal(y(100), 20)
  assert.equal(scaleLinear(5, 5, 0, 10)(5), 0)
})

test('date presets use the Vietnam calendar day', () => {
  // 2026-03-10 18:30 UTC is already 2026-03-11 01:30 in Vietnam.
  const now = new Date(Date.UTC(2026, 2, 10, 18, 30))
  assert.equal(isoDate(vietnamToday(now)), '2026-03-11')
  assert.deepEqual(presetRange('7d', now), { from: '2026-03-05', to: '2026-03-11' })
  assert.deepEqual(presetRange('30d', now), { from: '2026-02-10', to: '2026-03-11' })
  assert.deepEqual(presetRange('ytd', now), { from: '2026-01-01', to: '2026-03-11' })
  assert.equal(rangeDays('2026-03-05', '2026-03-11'), 7)
  assert.equal(rangeDays('2026-03-05', '2026-03-05'), 1)
  assert.ok(Number.isNaN(rangeDays('x', '2026-03-05')))
  assert.equal(defaultGranularity('7d'), 'day')
  assert.equal(defaultGranularity('90d'), 'week')
  assert.equal(defaultGranularity('ytd'), 'month')
})

test('bucket labels and status labels', () => {
  assert.equal(formatBucketLabel('2026-03-02', 'day'), '02/03')
  assert.equal(formatBucketLabel('2026-03-02', 'month'), '03/2026')
  assert.equal(statusLabel('delivered'), 'Đã giao')
  assert.equal(statusLabel('cod'), 'Giao tận nơi (COD)')
  assert.equal(statusLabel('weird'), 'weird')
  assert.equal(statusLabel(''), 'Khác')
})

test('parseImportReport tolerates nulls and unknown shapes', () => {
  const r = parseImportReport({ mode: 'dry_run', created: 2, updated: 1, skipped: 0, ignored_stock: 3, rows: 4, committed: false,
    errors: [{ row: 3, field: 'size', message: 'Sai cỡ' }, null], notes: ['a'], items: [{ row: 2, slug: 'ao', action: 'update', detail: 'x' }, { row: 3, slug: 'b', action: 'weird' }] })
  assert.equal(r.created, 2)
  assert.deepEqual(r.errors, [{ row: 3, field: 'size', message: 'Sai cỡ' }])
  assert.equal(r.items[0].action, 'update')
  assert.equal(r.items[1].action, 'create')
  const empty = parseImportReport(null)
  assert.deepEqual([empty.errors, empty.notes, empty.items, empty.rows, empty.committed], [[], [], [], 0, false])
  const nulls = parseImportReport({ errors: null, items: null, notes: null })
  assert.deepEqual([nulls.errors, nulls.items, nulls.notes], [[], [], []])
})

test('canCommit only for an error-free dry run that writes something', () => {
  const base = parseImportReport({ mode: 'dry_run', created: 1, rows: 3 })
  assert.equal(canCommit(base), true)
  assert.equal(canCommit({ ...base, errors: [{ row: 2, field: 'x', message: 'y' }] }), false)
  assert.equal(canCommit({ ...base, created: 0, updated: 0, skipped: 3 }), false)
  assert.equal(canCommit({ ...base, mode: 'commit' }), false)
  assert.equal(canCommit(null), false)
  assert.equal(commitLabel(base), 'Nhập 3 dòng')
  assert.deepEqual([...errorRows({ ...base, errors: [{ row: 5, field: 'a', message: 'b' }, { row: 5, field: 'c', message: 'd' }] })], [5])
})

test('fileProblem: extension, size, empty', () => {
  assert.equal(fileProblem({ name: 'a.csv', size: 10 }), '')
  assert.equal(fileProblem({ name: 'a.JSON', size: 10 }), '')
  assert.match(fileProblem({ name: 'a.xlsx', size: 10 }), /csv/)
  assert.match(fileProblem({ name: 'a.csv', size: IMPORT_MAX_BYTES + 1 }), /lớn/)
  assert.match(fileProblem({ name: 'a.csv', size: 0 }), /trống/)
})

test('templates: BOM, header, quoting; JSON round trips', () => {
  const csv = templateCsv('products')
  assert.equal(csv.charCodeAt(0), 0xfeff)
  assert.ok(csv.slice(1).startsWith('product_slug,name,category_slug,description,material,price,compare_at_price,is_active,image_url,gallery_urls,color_name,color_hex,size,sku,stock'))
  assert.equal(toCsv([['a,b', 'c"d', 'e']]), '"a,b","c""d",e\r\n')
  const json = JSON.parse(templateJson('categories'))
  assert.equal(json[0].slug, 'ao-dai')
  assert.equal(json[1].parent_slug, 'ao-dai')
  assert.deepEqual(Object.keys(json[0]), ['name', 'slug', 'description', 'parent_slug', 'sort_order', 'is_active', 'image_url', 'image_alt'])
})
