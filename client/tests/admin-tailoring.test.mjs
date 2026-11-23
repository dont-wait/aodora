import { test } from 'node:test'
import assert from 'node:assert/strict'
import { adminTailoring, AdminTailoringApiError, adminTailoringSearch, readAdminTailoringFilters, tailoringStatusLabel } from '../src/features/admin/tailoring.ts'
import {
  adminTailoringActionButtons,
  adminTailoringConfirmation,
  buildTailoringPatch,
  confirmAdminMeasurements,
  createAdminCustomOrder,
  fromLocalInput,
  initialCustomOrder,
  initialTailoringEdit,
  loadAdminTailoring,
  loadAdminTailorings,
  orderablePhase,
  performAdminTailoringAction,
  saveAdminTailoring,
  toLocalInput,
  validateCustomOrder,
} from '../src/features/admin/tailoringActions.ts'

const tailoring = (over = {}) => ({
  id: 'tr-1', customer_id: null, is_guest: true, full_name: 'Khách mẫu', phone: '0900000001', preferred_contact: null, category_id: null,
  product_id: null, product_slug: null, measurements: null, notes: null, status: 'submitted', appointment_at: null, assigned_staff: null,
  created_at: '2026-11-17T00:00:00Z', updated_at: '2026-11-17T01:00:00Z', allowed_actions: ['update', 'update_measurements', 'contact', 'cancel'], ...over,
})
const raw = (over = {}) => JSON.stringify({ data: tailoring(over) })
const response = (body, status = 200) => new Response(typeof body === 'string' ? body : JSON.stringify(body), { status })
const err = (status, code, field = '') => response({ error: { code, field, message: `lỗi ${code}` } }, status)
async function withFetch(fn, run) {
  const old = globalThis.fetch
  globalThis.fetch = fn
  try {
    await run()
  } finally {
    globalThis.fetch = old
  }
}
const calls = []
const recording = (handler) => async (url, init) => {
  calls.push({ url, init })
  return handler(url, init, calls.length)
}

test('lọc/phân trang: giá trị lạ hoặc lặp không đẩy xuống API', () => {
  assert.deepEqual(readAdminTailoringFilters(new URLSearchParams('status=bogus&page=abc&page_size=9999')), { status: '', page: 1, page_size: 100 })
  assert.equal(readAdminTailoringFilters(new URLSearchParams('status=contacted&status=cancelled')).status, '')
  assert.equal(readAdminTailoringFilters(new URLSearchParams('status=measurement_confirmed')).status, 'measurement_confirmed')
  assert.equal(adminTailoringSearch({ status: 'nope', page: 3, page_size: 20 }).toString(), 'page=3&page_size=20')
  assert.equal(adminTailoringSearch({ status: 'completed', page: 1, page_size: 50 }).toString(), 'page=1&page_size=50&status=completed')
  assert.equal(tailoringStatusLabel('consultation_scheduled'), 'Đã hẹn tư vấn')
})

test('API: route/method/body, credentials include, không header guest, id được mã hóa', async () => {
  calls.length = 0
  await withFetch(recording(() => response(raw({ status: 'contacted' }))), async () => {
    await adminTailoring.get('a/b c')
    await adminTailoring.update('tr-1', { notes: null, appointment_at: null, measurements: { waist_cm: 65.25, neck_cm: null } })
    await adminTailoring.setStatus('tr-1', 'contacted')
    await adminTailoring.confirmMeasurements('tr-1')
  })
  assert.deepEqual(calls.map((c) => [c.init.method, c.url]), [
    ['GET', '/api/v1/admin/tailoring-requests/a%2Fb%20c'],
    ['PATCH', '/api/v1/admin/tailoring-requests/tr-1'],
    ['PATCH', '/api/v1/admin/tailoring-requests/tr-1/status'],
    ['POST', '/api/v1/admin/tailoring-requests/tr-1/confirm-measurements'],
  ])
  for (const c of calls) {
    assert.equal(c.init.credentials, 'include')
    assert.ok(!('X-Order-Access-Token' in c.init.headers) && !('Idempotency-Key' in c.init.headers))
  }
  assert.deepEqual(JSON.parse(calls[1].init.body), { notes: null, appointment_at: null, measurements: { waist_cm: 65.25, neck_cm: null } })
  assert.ok(!('status' in JSON.parse(calls[1].init.body)))
  assert.deepEqual(JSON.parse(calls[2].init.body), { status: 'contacted' })
  assert.deepEqual(JSON.parse(calls[3].init.body), { confirmed: true })
})

test('list: URL từ bộ lọc an toàn, tổng/trang là số, số đo giữ chuỗi chính xác', async () => {
  calls.length = 0
  const body = '{"data":[{"id":"tr-1","customer_id":null,"is_guest":true,"full_name":"A","phone":"1","status":"submitted","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","measurements":{"waist_cm":65.250,"body":"1m62"},"allowed_actions":["update"]}],"total":41,"page":2,"page_size":20}'
  await withFetch(recording(() => response(body)), async () => {
    const page = await adminTailoring.list({ status: 'bogus', page: 2, page_size: 20 })
    assert.equal(page.total, 41)
    assert.equal(page.page, 2)
    assert.equal(page.data[0].measurements.waist_cm, '65.250')
    assert.equal(page.data[0].measurements.body, '1m62')
    assert.equal(page.data[0].product_slug, null)
  })
  assert.equal(calls[0].url, '/api/v1/admin/tailoring-requests?page=2&page_size=20')
})

test('lỗi giữ status/code/field; phản hồi thiếu/hỏng bị từ chối', async () => {
  for (const [status, code, field] of [[403, 'ADMIN_REQUIRED', ''], [401, 'AUTH_REQUIRED', ''], [404, 'NOT_FOUND', ''], [409, 'MEASUREMENT_LOCKED', 'measurements'], [503, 'TAILORING_UNAVAILABLE', '']]) {
    await withFetch(async () => err(status, code, field), async () => {
      await assert.rejects(adminTailoring.get('x'), (e) => e instanceof AdminTailoringApiError && e.status === status && e.code === code && e.field === field)
    })
  }
  for (const bad of ['not json', JSON.stringify({ data: tailoring({ allowed_actions: undefined }) }), JSON.stringify({ data: { id: 'x' } }), '{}']) {
    await withFetch(async () => response(bad), async () => {
      await assert.rejects(adminTailoring.get('x'), (e) => e instanceof AdminTailoringApiError && e.code === 'INVALID_RESPONSE')
    })
  }
})

test('nút trạng thái chỉ sinh từ allowed_actions; confirm chỉ khi server cho phép', () => {
  const none = adminTailoringActionButtons(tailoring({ allowed_actions: ['update'] }))
  assert.deepEqual([none.status.length, none.canConfirmMeasurements, none.canEditMeasurements], [0, false, false])
  const some = adminTailoringActionButtons(tailoring({ allowed_actions: ['update', 'update_measurements', 'schedule', 'confirm_measurements', 'cancel'] }))
  assert.deepEqual(some.status.map((a) => [a.id, a.target]), [['schedule', 'consultation_scheduled'], ['cancel', 'cancelled']])
  assert.equal(some.canConfirmMeasurements, true)
  assert.equal(some.canEditMeasurements, true)
  const all = adminTailoringActionButtons(tailoring({ allowed_actions: ['contact', 'schedule', 'start', 'complete', 'cancel'] }))
  assert.deepEqual(all.status.map((a) => a.target), ['contacted', 'consultation_scheduled', 'in_progress', 'completed', 'cancelled'])
  assert.match(adminTailoringConfirmation(tailoring(), 'cancel'), /không hoàn tác/)
})

test('thao tác trạng thái: gọi đúng endpoint; không được phép thì không gọi mà tải lại', async () => {
  calls.length = 0
  await withFetch(recording((url) => (url.endsWith('/status') ? response(raw({ status: 'contacted' })) : response(raw({ allowed_actions: ['update'] })))), async () => {
    const saved = await performAdminTailoringAction(tailoring(), 'contact')
    assert.equal(saved.kind, 'saved')
    assert.equal(saved.tailoring.status, 'contacted')
    const blocked = await performAdminTailoringAction(tailoring({ allowed_actions: ['update'] }), 'complete')
    assert.equal(blocked.kind, 'conflict')
  })
  assert.deepEqual(calls.map((c) => c.init.method + ' ' + c.url.split('tr-1')[1]), ['PATCH /status', 'GET '])
})

test('409 ⇒ tự GET lại; GET lỗi thì KHÔNG dùng dữ liệu cũ; 400/503 phân loại đúng', async () => {
  await withFetch(async (url, init) => (init.method === 'GET' ? response(raw({ status: 'cancelled', allowed_actions: ['update'] })) : err(409, 'TAILORING_STATE_CONFLICT')), async () => {
    const r = await performAdminTailoringAction(tailoring(), 'contact')
    assert.equal(r.kind, 'conflict')
    assert.equal(r.tailoring.status, 'cancelled')
    assert.match(r.message, /tải lại/)
  })
  await withFetch(async (url, init) => (init.method === 'GET' ? err(503, 'TAILORING_UNAVAILABLE') : err(409, 'TAILORING_STATE_CONFLICT')), async () => {
    const r = await performAdminTailoringAction(tailoring(), 'contact')
    assert.equal(r.kind, 'error')
    assert.ok(!('tailoring' in r))
  })
  await withFetch(async (url, init) => (init.method === 'GET' ? err(404, 'NOT_FOUND') : err(409, 'TAILORING_STATE_CONFLICT')), async () => {
    assert.equal((await performAdminTailoringAction(tailoring(), 'contact')).kind, 'not-found')
  })
  await withFetch(async () => err(400, 'INVALID_FIELD', 'notes'), async () => {
    const r = await saveAdminTailoring(tailoring(), { notes: 'x' })
    assert.deepEqual([r.kind, r.field], ['invalid', 'notes'])
  })
  await withFetch(async () => err(503, 'TAILORING_UNAVAILABLE'), async () => {
    assert.equal((await saveAdminTailoring(tailoring(), { notes: 'x' })).kind, 'error')
  })
  await withFetch(async () => { throw new TypeError('network') }, async () => {
    assert.equal((await saveAdminTailoring(tailoring(), { notes: 'x' })).kind, 'error')
  })
  for (const [status, kind] of [[401, 'unauthenticated'], [403, 'forbidden']]) {
    await withFetch(async () => err(status, 'X'), async () => assert.equal((await saveAdminTailoring(tailoring(), { notes: 'x' })).kind, kind))
  }
})

test('confirm-measurements: không có quyền thì không gọi endpoint; có quyền gọi POST', async () => {
  calls.length = 0
  await withFetch(recording((url, init) => (init.method === 'GET' ? response(raw()) : response(raw({ status: 'measurement_confirmed' })))), async () => {
    assert.equal((await confirmAdminMeasurements(tailoring())).kind, 'conflict')
    const r = await confirmAdminMeasurements(tailoring({ allowed_actions: ['confirm_measurements'] }))
    assert.equal(r.kind, 'saved')
  })
  assert.deepEqual(calls.map((c) => c.init.method), ['GET', 'POST'])
})

test('buildTailoringPatch: chỉ trường đổi; trống ⇒ không gửi; xóa riêng ⇒ null; không bao giờ có status/0', () => {
  const t = tailoring({ notes: 'cũ', assigned_staff: 'Lan', preferred_contact: 'zalo', appointment_at: '2026-12-01T02:00:00Z', measurements: { waist_cm: '65', neck_cm: '34', body: '1m60' } })
  const base = initialTailoringEdit(t)
  assert.deepEqual(buildTailoringPatch(t, base, true).patch, {})
  const edited = { ...base, notes: ' mới ', assigned_staff: '', preferred_contact: '', appointment: '',
    measurements: { ...base.measurements, waist_cm: '65,5', bust_cm: '' }, cleared: { ...base.cleared, neck_cm: true, body: true } }
  const { patch, errors } = buildTailoringPatch(t, edited, true)
  assert.deepEqual(errors, {})
  assert.deepEqual(patch, { notes: 'mới', assigned_staff: null, preferred_contact: null, appointment_at: null, measurements: { waist_cm: 65.5, neck_cm: null, body: null } })
  const json = JSON.stringify(patch)
  assert.ok(!json.includes('status') && !json.includes('customer_id') && !json.includes('bust_cm') && !/:0[,}]/.test(json))
  // locked: measurement edits are never sent
  assert.equal(buildTailoringPatch(t, edited, false).patch.measurements, undefined)
  // invalid measurement blocks sending
  const bad = buildTailoringPatch(t, { ...base, measurements: { ...base.measurements, hips_cm: 'abc', sleeve_cm: '-1', waist_cm: '0' } }, true)
  assert.deepEqual(Object.keys(bad.errors).sort(), ['hips_cm', 'sleeve_cm', 'waist_cm'])
})

test('datetime-local ⇄ RFC3339 khứ hồi; giá trị sai bị chặn; trống xóa lịch hẹn', () => {
  const iso = '2026-12-01T02:30:00.000Z'
  assert.equal(fromLocalInput(toLocalInput(iso)), iso)
  assert.equal(toLocalInput(null), '')
  assert.equal(fromLocalInput('not a date'), null)
  assert.equal(fromLocalInput('2026-13-45T99:99'), null)
  const t = tailoring({ appointment_at: iso })
  const v = initialTailoringEdit(t)
  assert.deepEqual(buildTailoringPatch(t, { ...v, appointment: '' }, true).patch, { appointment_at: null })
  assert.ok(buildTailoringPatch(t, { ...v, appointment: '2026-13-45T99:99' }, true).errors.appointment)
})

const orderValues = (over = {}) => ({ ...initialCustomOrder(tailoring({ product_id: 'p'.repeat(24) })), unit_price: '1500000.50', payment_method: 'cash_at_counter', ...over })
const withProduct = tailoring({ product_id: '680000000000000000000001', status: 'measurement_confirmed' })

test('đơn custom: unit_price giữ nguyên chuỗi, không float; điều kiện hiển thị panel', () => {
  assert.equal(orderablePhase('measurement_confirmed'), true)
  assert.equal(orderablePhase('in_progress'), true)
  assert.equal(orderablePhase('contacted'), false)
  const { body, errors } = validateCustomOrder(withProduct, orderValues())
  assert.deepEqual(errors, {})
  assert.equal(body.unit_price, '1500000.50')
  assert.equal(typeof body.unit_price, 'string')
  assert.ok(JSON.stringify(body).includes('"unit_price":"1500000.50"'))
  assert.deepEqual(body.shipping_address, { recipient: 'Khách mẫu', phone: '0900000001' })
  assert.ok(!('product_id' in body))
  assert.equal(validateCustomOrder(withProduct, orderValues({ unit_price: '12345678901234567890.123456789' })).body.unit_price, '12345678901234567890.123456789')
})

test('đơn custom: kiểm tra cục bộ (giá, số lượng, COD cần địa chỉ, mẫu chỉ khi chưa có)', () => {
  for (const bad of ['', '0', '0.00', '-5', 'abc', '1e6', '1,5', '1.']) {
    assert.ok(validateCustomOrder(withProduct, orderValues({ unit_price: bad })).errors.unit_price, `giá "${bad}" phải bị chặn`)
  }
  assert.ok(validateCustomOrder(withProduct, orderValues({ quantity: '0' })).errors.quantity)
  assert.ok(validateCustomOrder(withProduct, orderValues({ quantity: '1.5' })).errors.quantity)
  assert.ok(validateCustomOrder(withProduct, orderValues({ payment_method: '' })).errors.payment_method)
  const cod = validateCustomOrder(withProduct, orderValues({ payment_method: 'cod' }))
  assert.deepEqual(Object.keys(cod.errors).sort(), ['city', 'line1'])
  assert.equal(cod.body, undefined)
  const ok = validateCustomOrder(withProduct, orderValues({ payment_method: 'cod', line1: ' 1 Lê Lợi ', city: 'Huế' }))
  assert.deepEqual(ok.body.shipping_address, { recipient: 'Khách mẫu', phone: '0900000001', line1: '1 Lê Lợi', city: 'Huế' })
  const noProduct = tailoring({ product_id: null })
  assert.ok(validateCustomOrder(noProduct, orderValues()).errors.product_id)
  assert.equal(validateCustomOrder(noProduct, orderValues({ product_id: '680000000000000000000001' })).body.product_id, '680000000000000000000001')
  assert.ok(validateCustomOrder(withProduct, orderValues({ recipient: ' ' })).errors.recipient)
})

test('tạo đơn: 201 created, 200 exists (kèm link), body gửi đúng literal', async () => {
  calls.length = 0
  const { body } = validateCustomOrder(withProduct, orderValues())
  await withFetch(recording((url, init, n) => response({ data: { id: 'order-9', order_number: 'AD-9', total: 3000001, items: [] }, created: n === 1 }, n === 1 ? 201 : 200)), async () => {
    const a = await createAdminCustomOrder(withProduct, body)
    assert.deepEqual([a.kind, a.result.orderId, a.result.orderNumber], ['created', 'order-9', 'AD-9'])
    const b = await createAdminCustomOrder(withProduct, body)
    assert.deepEqual([b.kind, b.result.orderId], ['exists', 'order-9'])
  })
  assert.equal(calls[0].url, '/api/v1/admin/tailoring-requests/tr-1/order')
  assert.equal(calls[0].init.method, 'POST')
  assert.ok(calls[0].init.body.includes('"unit_price":"1500000.50"'))
  assert.ok(!/customer_id|"status"|total|shipping_fee/.test(calls[0].init.body))
})

test('tạo đơn: CUSTOM_ORDER_EXISTS, SHIPPING_UNAVAILABLE, 400 theo field, 409 trạng thái ⇒ tải lại, mất mạng ⇒ chưa rõ', async () => {
  const { body } = validateCustomOrder(withProduct, orderValues())
  await withFetch(async () => err(409, 'CUSTOM_ORDER_EXISTS'), async () => assert.equal((await createAdminCustomOrder(withProduct, body)).kind, 'different-order-exists'))
  await withFetch(async () => err(503, 'SHIPPING_UNAVAILABLE'), async () => {
    const r = await createAdminCustomOrder(withProduct, body)
    assert.equal(r.kind, 'shipping-unavailable')
    assert.match(r.message, /chưa được cấu hình/)
  })
  await withFetch(async () => err(400, 'INVALID_FIELD', 'product_id'), async () => {
    const r = await createAdminCustomOrder(withProduct, body)
    assert.deepEqual([r.kind, r.field], ['invalid', 'product_id'])
  })
  for (const code of ['TAILORING_STATE_CONFLICT', 'MEASUREMENT_INCOMPLETE']) {
    await withFetch(async (url, init) => (init.method === 'GET' ? response(raw({ status: 'completed' })) : err(409, code)), async () => {
      const r = await createAdminCustomOrder(withProduct, body)
      assert.equal(r.kind, 'conflict')
      assert.equal(r.tailoring.status, 'completed')
    })
  }
  let attempts = 0
  await withFetch(async () => { attempts++; throw new TypeError('network') }, async () => {
    assert.equal((await createAdminCustomOrder(withProduct, body)).kind, 'unknown')
  })
  assert.equal(attempts, 1, 'không tự gửi lại khi chưa rõ kết quả')
  await withFetch(async () => err(503, 'TAILORING_UNAVAILABLE'), async () => assert.equal((await createAdminCustomOrder(withProduct, body)).kind, 'unknown'))
  await withFetch(async () => response({ data: { id: 'x' } }, 201), async () => assert.equal((await createAdminCustomOrder(withProduct, body)).kind, 'unknown'))
  for (const [status, kind] of [[401, 'unauthenticated'], [403, 'forbidden'], [404, 'not-found']]) {
    await withFetch(async () => err(status, 'X'), async () => assert.equal((await createAdminCustomOrder(withProduct, body)).kind, kind))
  }
})

test('loaders: phân loại 401/403/404/lỗi', async () => {
  for (const [status, kind] of [[401, 'unauthenticated'], [403, 'forbidden'], [404, 'not-found'], [503, 'error']]) {
    await withFetch(async () => err(status, 'X'), async () => {
      assert.equal((await loadAdminTailoring('x')).kind, kind)
      assert.equal((await loadAdminTailorings({ status: '', page: 1, page_size: 20 })).kind, kind)
    })
  }
  await withFetch(async () => response(raw()), async () => assert.equal((await loadAdminTailoring('tr-1')).kind, 'ready'))
})
