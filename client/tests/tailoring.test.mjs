import { test } from 'node:test'
import assert from 'node:assert/strict'
import { tailoringApi, TailoringApiError } from '../src/features/tailoring/api.ts'
import {
  buildTailoringRequest,
  measurementNumbers,
  parseMeasurement,
  emptyTailoringForm,
  formFieldForServerField,
  productRefFromSearch,
  tailoringSuccessMessage,
  validateTailoringForm,
} from '../src/features/tailoring/form.ts'

const json = (data, status = 200) => new Response(JSON.stringify(data), { status })
async function withFetch(fn, run) {
  const old = globalThis.fetch
  globalThis.fetch = fn
  try {
    await run()
  } finally {
    globalThis.fetch = old
  }
}
const form = (over = {}) => ({ ...emptyTailoringForm, name: ' Nguyễn Thị Hà ', phone: ' 0912345678 ', ...over })

test('submit: POST đúng route, JSON, credentials include, không header guest/idempotency', async () => {
  let call
  await withFetch(async (url, init) => { call = { url, init }; return json({ data: { id: '6d0000000000000000000001', status: 'submitted' } }, 201) }, async () => {
    const receipt = await tailoringApi.submit({ full_name: 'A', phone: '0900000001' })
    assert.deepEqual(receipt, { id: '6d0000000000000000000001', status: 'submitted' })
  })
  assert.equal(call.url, '/api/v1/tailoring-requests')
  assert.equal(call.init.method, 'POST')
  assert.equal(call.init.credentials, 'include')
  assert.equal(call.init.headers['Content-Type'], 'application/json')
  assert.deepEqual(Object.keys(call.init.headers).sort(), ['Accept', 'Content-Type'])
  assert.deepEqual(JSON.parse(call.init.body), { full_name: 'A', phone: '0900000001' })
})

test('submit: lỗi giữ status/code/field/message (400, 401, 503)', async () => {
  for (const [status, code, field] of [[400, 'INVALID_FIELD', 'product_slug'], [401, 'AUTH_REQUIRED', ''], [503, 'TAILORING_UNAVAILABLE', '']]) {
    await withFetch(async () => json({ error: { code, field, message: `lỗi ${code}` } }, status), async () => {
      await assert.rejects(tailoringApi.submit({ full_name: 'A', phone: '0900000001' }), (e) => {
        assert.ok(e instanceof TailoringApiError)
        assert.deepEqual([e.status, e.code, e.field, e.message], [status, code, field, `lỗi ${code}`])
        return true
      })
    })
  }
})

test('submit: chỉ 201 hợp lệ mới là thành công; 200, thiếu id, body hỏng đều bị từ chối', async () => {
  for (const res of [
    () => json({ data: { id: 'x', status: 'submitted' } }, 200),
    () => json({ data: { status: 'submitted' } }, 201),
    () => json({ data: { id: '', status: 'submitted' } }, 201),
    () => json({}, 201),
    () => new Response('not json', { status: 201 }),
  ]) {
    await withFetch(async () => res(), async () => {
      await assert.rejects(tailoringApi.submit({ full_name: 'A', phone: '0900000001' }), (e) => e instanceof TailoringApiError && e.code === 'INVALID_RESPONSE')
    })
  }
})

test('submit: lỗi mạng được ném ra, không thành công giả', async () => {
  await withFetch(async () => { throw new TypeError('network') }, async () => {
    await assert.rejects(tailoringApi.submit({ full_name: 'A', phone: '0900000001' }), TypeError)
  })
})

test('buildTailoringRequest: trim, số đo trống KHÔNG gửi (không bao giờ là 0), body là mô tả', () => {
  const minimal = buildTailoringRequest(form())
  assert.deepEqual(minimal, { full_name: 'Nguyễn Thị Hà', phone: '0912345678' })
  assert.ok(!('measurements' in minimal) && !('notes' in minimal) && !('preferred_contact' in minimal))

  const body = buildTailoringRequest(form({ waist: '65,25', bust: '86', body: ' 1m62 / 50kg ', notes: ' tà rộng ', contact: 'zalo' }), { id: '680000000000000000000001', slug: 'ao-dai' })
  assert.deepEqual(body, {
    full_name: 'Nguyễn Thị Hà', phone: '0912345678', preferred_contact: 'zalo', product_id: '680000000000000000000001', product_slug: 'ao-dai',
    measurements: { waist_cm: 65.25, bust_cm: 86, body: '1m62 / 50kg' }, notes: 'tà rộng',
  })
  assert.ok(!JSON.stringify(body).includes(':0,') && !('customer_id' in body) && !('status' in body))
})

test('buildTailoringRequest: chỉ body (mô tả) vẫn gửi measurements.body, số 0 không lọt qua', () => {
  assert.deepEqual(buildTailoringRequest(form({ body: '1m60' })).measurements, { body: '1m60' })
  assert.equal(buildTailoringRequest(form({ waist: '0' })).measurements, undefined)
})

test('validateTailoringForm: bắt buộc tên + điện thoại, số đo âm/0/chữ/quá lớn bị chặn, trống thì hợp lệ', () => {
  assert.deepEqual(Object.keys(validateTailoringForm(emptyTailoringForm)).sort(), ['name', 'phone'])
  assert.deepEqual(Object.keys(validateTailoringForm(form({ name: '   ' }))), ['name'])
  assert.deepEqual(validateTailoringForm(form()), {})
  for (const bad of ['-1', '0', 'abc', '501', '1e2', '1.2.3', '0.0']) {
    assert.ok(validateTailoringForm(form({ waist: bad })).waist, `waist=${bad} phải bị chặn`)
  }
  for (const ok of ['0.5', '65', '65,5', '500']) assert.deepEqual(validateTailoringForm(form({ waist: ok })), {})
})

test('productRefFromSearch: đọc product_id/product_slug từ query của PurchasePanel, không có thì rỗng', () => {
  assert.deepEqual(productRefFromSearch('?product_id=680000000000000000000001&product_slug=ao-dai&mode=custom'), { id: '680000000000000000000001', slug: 'ao-dai' })
  assert.deepEqual(productRefFromSearch(''), {})
  assert.deepEqual(productRefFromSearch('?mode=custom&product_slug=%20'), {})
})

test('formFieldForServerField: field server ánh xạ về ô nhập', () => {
  assert.equal(formFieldForServerField('full_name'), 'name')
  assert.equal(formFieldForServerField('garment_length_cm'), 'length')
  assert.equal(formFieldForServerField('neck_cm'), 'neck')
  assert.equal(formFieldForServerField('product_slug'), 'product')
  assert.equal(formFieldForServerField('body'), 'body')
})

test('thông báo thành công nói rõ chưa phải đặt đơn, không dùng cụm "đặt hàng thành công"', () => {
  assert.match(tailoringSuccessMessage, /chưa phải đặt đơn/)
  assert.doesNotMatch(tailoringSuccessMessage, /đặt hàng thành công/i)
})

test('parseMeasurement dùng chung: dấu phẩy ra đúng số, NaN/âm/0/chữ không lọt', () => {
  assert.equal(parseMeasurement('65,5'), 65.5)
  assert.equal(parseMeasurement('65.5'), 65.5)
  assert.equal(parseMeasurement('34'), 34)
  for (const bad of ['', ' ', 'abc', '-3', '0', '0,0', 'NaN', 'Infinity', '1e2', '501', '1,2,3', '6 5']) {
    assert.equal(parseMeasurement(bad), null, `"${bad}" phải bị từ chối`)
  }
})

test('measurementNumbers (hồ sơ KR-02): ô trống bị bỏ, dấu phẩy đúng, JSON không có null/NaN/0', () => {
  const nums = measurementNumbers(form({ waist: '65,5', neck: '', bust: ' 86 ', hips: 'abc', sleeve: '-3', shoulder: '0' }))
  assert.deepEqual(nums, { waist_cm: 65.5, bust_cm: 86 })
  const text = JSON.stringify({ ...nums, body: undefined })
  assert.ok(!/null|NaN|:0[,}]/.test(text), text)
  assert.deepEqual(measurementNumbers(form()), {})
})

test('hồ sơ KR-02 và yêu cầu may đo cho cùng kết quả số đo', () => {
  const f = form({ waist: '65,25', bust: '86', length: '138,5' })
  assert.deepEqual(measurementNumbers(f), { waist_cm: 65.25, bust_cm: 86, garment_length_cm: 138.5 })
  const { body, ...rest } = buildTailoringRequest(f).measurements
  assert.deepEqual(rest, measurementNumbers(f))
  assert.equal(body, undefined)
})

import { buildConsultBody, buildConsultNotes, validateConsult, consultFieldForServerField, consultSuccessMessage } from '../src/features/tailoring/consult.ts'

test('consult notes: "Nhãn: giá trị" lines, blank values dropped and trimmed', () => {
  assert.equal(buildConsultNotes([['Thành phố', 'Hà Nội'], ['Ghi chú', '   '], ['Ngày', ' 2026-12-01 ']]), 'Thành phố: Hà Nội\nNgày: 2026-12-01')
  assert.equal(buildConsultNotes([['A', ''], ['B', ' ']]), '')
})

test('consult body: trims name/phone, omits notes when empty, never sends server-owned fields', () => {
  assert.deepEqual(buildConsultBody({ name: ' An ', phone: ' 0900000001 ', extras: [['Ghi chú', '']] }), { full_name: 'An', phone: '0900000001' })
  const body = buildConsultBody({ name: 'An', phone: '0900000001', extras: [['Khu vực', 'Hà Nội'], ['Địa chỉ', '1 Phố X']] })
  assert.equal(body.notes, 'Khu vực: Hà Nội\nĐịa chỉ: 1 Phố X')
  assert.deepEqual(Object.keys(body).sort(), ['full_name', 'notes', 'phone'])
})

test('consult validation and server field mapping', () => {
  assert.deepEqual(Object.keys(validateConsult({ name: ' ', phone: '' })).sort(), ['name', 'phone'])
  assert.deepEqual(validateConsult({ name: 'An', phone: '0900000001' }), {})
  assert.equal(consultFieldForServerField('full_name'), 'name')
  assert.equal(consultFieldForServerField('phone'), 'phone')
  assert.equal(consultFieldForServerField('notes'), '')
  assert.match(consultSuccessMessage, /chưa phải đặt đơn/)
})
