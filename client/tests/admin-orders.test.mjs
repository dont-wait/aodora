import { test } from 'node:test'
import assert from 'node:assert/strict'
import { adminOrders, AdminOrderApiError, adminOrderSearch, readAdminOrderFilters } from '../src/features/admin/orders.ts'
import {
  adminOrderActionGroups,
  adminOrderConfirmation,
  inventoryRestockLabel,
  loadAdminOrder,
  loadAdminOrders,
  performAdminOrderAction,
} from '../src/features/admin/orderActions.ts'
import { formatOrderMoney } from '../src/features/orders/money.ts'
import { orderLoginPath } from '../src/features/orders/presentation.ts'

const order = () => ({
  id: 'order-1',
  order_number: 'AD-TEST',
  status: 'confirmed',
  payment_method: 'cash_at_counter',
  payment_status: 'unpaid',
  subtotal: '500000',
  shipping_fee: '0',
  total: '500000',
  shipping_address: { recipient: 'Khách mẫu', phone: '0900000000' },
  items: [{ product_id: 'product-1', product_name: 'Áo mẫu', sku: 'A', quantity: 2, unit_price: '250000', fulfillment_mode: 'ready' }],
  created_at: '2026-11-17T00:00:00Z',
  updated_at: '2026-11-17T01:00:00Z',
  is_guest: true,
  allowed_actions: ['confirm', 'ship', 'deliver', 'cancel', 'restock', 'mark_paid', 'mark_refunded'],
})
const filter = () => ({ status: '', payment_status: '', page: 1, page_size: 20 })
const response = (payload, status = 200) => new Response(JSON.stringify(payload), { status })
async function withFetch(fn, run) {
  const old = globalThis.fetch
  globalThis.fetch = fn
  try {
    await run()
  } finally {
    globalThis.fetch = old
  }
}

const actionCases = [
  ['confirm', 'PATCH', '/status', { status: 'confirmed' }],
  ['ship', 'PATCH', '/status', { status: 'shipping' }],
  ['deliver', 'PATCH', '/status', { status: 'delivered' }],
  ['cancel', 'POST', '/cancel', undefined],
  ['restock', 'POST', '/restock', undefined],
  ['mark_paid', 'PATCH', '/payment', { payment_status: 'paid' }],
  ['mark_refunded', 'PATCH', '/payment', { payment_status: 'refunded' }],
]

test('mọi admin action gửi đúng method/path/body, include cookie và không guest header', async () => {
  for (const [action, method, path, body] of actionCases) {
    const calls = []
    const saved = { ...order(), status: 'delivered', payment_status: 'unpaid', allowed_actions: [] }
    await withFetch(
      async (url, options) => {
        calls.push({ url, ...options })
        return response({ data: saved })
      },
      async () => {
        const result = await performAdminOrderAction({ ...order(), id: 'order/id with space' }, action)
        assert.equal(result.kind, 'saved')
        assert.equal(result.order.status, saved.status)
        assert.equal(result.order.payment_status, 'unpaid')
        assert.deepEqual(result.order.allowed_actions, [])
      },
    )
    assert.equal(calls.length, 1)
    const call = calls[0]
    assert.ok(call.url.endsWith(`/api/v1/admin/orders/order%2Fid%20with%20space${path}`))
    assert.equal(call.method, method)
    assert.equal(call.credentials, 'include')
    assert.ok(!('X-Order-Access-Token' in call.headers))
    if (body === undefined) {
      assert.equal(call.body, undefined)
      assert.ok(!('Content-Type' in call.headers))
    } else {
      assert.deepEqual(JSON.parse(call.body), body)
      assert.equal(call.headers['Content-Type'], 'application/json')
    }
  }
})

test('admin list/get đúng endpoint, metadata phân trang và không gắn token guest', async () => {
  const calls = []
  await withFetch(
    async (url, options) => {
      calls.push({ url, ...options })
      return url.includes('?')
        ? response({ data: [order()], total: 81, page: 2, page_size: 50 })
        : response({
            data: {
              ...order(),
              customer_id: 'customer-1',
              is_guest: false,
              cancelled_at: '2026-11-17T02:00:00Z',
              inventory_restock: 'awaiting_return',
              paid_at: '2026-11-17T01:00:00Z',
            },
          })
    },
    async () => {
      const page = await adminOrders.list({ status: 'shipping', payment_status: 'paid', page: 2, page_size: 50 })
      assert.equal(page.total, 81)
      assert.equal(page.page, 2)
      assert.equal(page.page_size, 50)
      assert.equal(page.data[0].items[0].quantity, 2)
      const data = await adminOrders.get('order-1')
      assert.equal(data.is_guest, false)
      assert.equal(data.customer_id, 'customer-1')
      assert.equal(data.inventory_restock, 'awaiting_return')
      assert.equal(data.cancelled_at, '2026-11-17T02:00:00Z')
      assert.equal(data.paid_at, '2026-11-17T01:00:00Z')
    },
  )
  const url = new URL(calls[0].url, 'https://shop.example')
  assert.equal(url.pathname, '/api/v1/admin/orders')
  assert.equal(url.searchParams.get('status'), 'shipping')
  assert.equal(url.searchParams.get('payment_status'), 'paid')
  for (const call of calls) {
    assert.equal(call.method, 'GET')
    assert.equal(call.credentials, 'include')
    assert.ok(!('X-Order-Access-Token' in call.headers))
  }
})

test('tiền admin giữ nguyên JSON decimal lớn qua parseExactJSON, cả unit_price', async () => {
  const exact = '900719925474099312345678901234.56'
  const data = { ...order(), total: '__TOTAL__', items: [{ ...order().items[0], unit_price: '__PRICE__' }] }
  const wire = JSON.stringify({ data }).replace('"__TOTAL__"', exact).replace('"__PRICE__"', '0.100000000000000001')
  await withFetch(
    async () => new Response(wire),
    async () => {
      const value = await adminOrders.get('order-1')
      assert.equal(value.total, exact)
      assert.equal(value.items[0].unit_price, '0.100000000000000001')
      assert.equal(formatOrderMoney(value.total), '900.719.925.474.099.312.345.678.901.234,56 ₫')
    },
  )
})

test('nút chỉ sinh từ allowed_actions; confirmed tại quầy không tự sinh ship/mark_paid', () => {
  const groups = adminOrderActionGroups({ ...order(), allowed_actions: ['deliver', 'cancel'] })
  assert.deepEqual(
    groups.delivery.map((action) => action.id),
    ['deliver', 'cancel'],
  )
  assert.deepEqual(groups.payment, [])
  const empty = adminOrderActionGroups({ ...order(), status: 'pending', payment_status: 'unpaid', allowed_actions: [] })
  assert.deepEqual(empty, { delivery: [], payment: [] })
  // Even inconsistent state must not be used as a client state machine.
  assert.deepEqual(
    adminOrderActionGroups({ ...order(), allowed_actions: ['ship'] }).delivery.map((action) => action.id),
    ['ship'],
  )
})

test('nhóm giao nhận và nhóm tiền tách riêng, action lạ không sinh nút', () => {
  const groups = adminOrderActionGroups({ ...order(), allowed_actions: [...order().allowed_actions, 'unknown-action'] })
  assert.deepEqual(
    groups.delivery.map((action) => action.id),
    ['confirm', 'ship', 'deliver', 'cancel', 'restock'],
  )
  assert.deepEqual(
    groups.payment.map((action) => action.id),
    ['mark_paid', 'mark_refunded'],
  )
  assert.ok(groups.payment[1].label.includes('hoàn tiền thủ công'))
})

test('xác nhận nêu hệ quả hủy shipping/tiền độc lập và hoàn tiền chỉ ghi nhận thủ công', () => {
  const cancel = adminOrderConfirmation({ ...order(), status: 'shipping' }, 'cancel')
  assert.ok(cancel.includes('chưa cộng tồn'))
  assert.ok(cancel.includes('hàng chưa về kho'))
  assert.ok(cancel.includes('không đổi trạng thái thanh toán'))
  assert.ok(cancel.includes('không tự hoàn tiền'))
  const refund = adminOrderConfirmation(order(), 'mark_refunded')
  assert.ok(refund.includes('500.000 ₫'))
  assert.ok(refund.includes('thủ công'))
  assert.ok(refund.includes('không tự chuyển tiền'))
  for (const [action] of actionCases) assert.ok(adminOrderConfirmation(order(), action).length > 20)
  assert.equal(inventoryRestockLabel('restocked'), 'Đã nhập lại')
  assert.equal(inventoryRestockLabel('awaiting_return'), 'Chờ hàng về')
  assert.equal(inventoryRestockLabel('not_applicable'), 'Không áp dụng')
})

test('tất cả 409 state/payment/restock tự GET lại rồi dùng DTO/allowed_actions mới', async () => {
  for (const code of ['ORDER_STATE_CONFLICT', 'PAYMENT_STATE_CONFLICT', 'RESTOCK_UNAVAILABLE']) {
    const calls = []
    const fresh = {
      ...order(),
      status: 'cancelled',
      payment_status: 'paid',
      inventory_restock: 'awaiting_return',
      allowed_actions: ['restock', 'mark_refunded'],
    }
    await withFetch(
      async (url, options) => {
        calls.push({ url, ...options })
        return options.method === 'GET' ? response({ data: fresh }) : response({ error: { code, message: 'Xung đột' } }, 409)
      },
      async () => {
        const result = await performAdminOrderAction(order(), 'cancel')
        assert.equal(result.kind, 'conflict')
        assert.equal(result.order.status, 'cancelled')
        assert.equal(result.order.payment_status, 'paid')
        assert.deepEqual(result.order.allowed_actions, fresh.allowed_actions)
        assert.ok(result.message.includes('Đã tải lại'))
        if (code === 'RESTOCK_UNAVAILABLE') assert.ok(result.message.includes('khôi phục'))
      },
    )
    assert.equal(calls.length, 2)
    assert.equal(calls[0].method, 'POST')
    assert.equal(calls[1].method, 'GET')
    assert.ok(calls[1].url.endsWith('/api/v1/admin/orders/order-1'))
  }
})

test('action không còn allowed không POST và lấy lại đơn thay vì tự suy luận', async () => {
  const calls = []
  await withFetch(
    async (_url, options) => {
      calls.push(options.method)
      return response({ data: { ...order(), allowed_actions: [] } })
    },
    async () => {
      assert.equal((await performAdminOrderAction({ ...order(), allowed_actions: [] }, 'ship')).kind, 'conflict')
    },
  )
  assert.deepEqual(calls, ['GET'])
})

test('409 mà GET lại lỗi không cho dùng dữ liệu/nút cũ; phân biệt 401/403/404/503', async () => {
  for (const [status, kind] of [
    [401, 'unauthenticated'],
    [403, 'forbidden'],
    [404, 'not-found'],
    [503, 'error'],
  ])
    await withFetch(
      async (_url, options) => {
        return options.method === 'GET' ? response({ error: { code: 'READ_FAILED' } }, status) : response({ error: { code: 'ORDER_STATE_CONFLICT' } }, 409)
      },
      async () => {
        const result = await performAdminOrderAction(order(), 'cancel')
        assert.equal(result.kind, kind)
        assert.ok(!('order' in result))
        if (status === 503) assert.ok(result.message.includes('Chưa tải lại'))
      },
    )
})

test('401 login, 403 trang 403, mạng/503 thử lại và không trả dữ liệu đơn cũ', async () => {
  for (const [status, kind] of [
    [401, 'unauthenticated'],
    [403, 'forbidden'],
    [503, 'error'],
  ])
    await withFetch(
      async () => response({ error: { code: 'UNAVAILABLE' } }, status),
      async () => {
        assert.equal((await loadAdminOrder('order-1')).kind, kind)
        assert.equal((await loadAdminOrders(filter())).kind, kind)
        const mutation = await performAdminOrderAction(order(), 'mark_paid')
        assert.equal(mutation.kind, kind)
        assert.ok(!('order' in mutation))
      },
    )
  await withFetch(
    async () => {
      throw new TypeError('network offline')
    },
    async () => {
      assert.equal((await loadAdminOrder('order-1')).kind, 'error')
      assert.equal((await performAdminOrderAction(order(), 'mark_paid')).kind, 'error')
    },
  )
})

test('API giữ status/code/field và không chấp nhận response ghi chưa rõ ràng', async () => {
  await withFetch(
    async () => response({ error: { code: 'INVALID_FIELD', field: 'status', message: 'Không hợp lệ' } }, 400),
    async () => {
      await assert.rejects(
        adminOrders.setStatus('id', 'confirmed'),
        (error) => error instanceof AdminOrderApiError && error.status === 400 && error.code === 'INVALID_FIELD' && error.field === 'status',
      )
    },
  )
  await withFetch(
    async () => new Response('<html>unavailable</html>', { status: 503 }),
    async () => {
      await assert.rejects(adminOrders.get('id'), (error) => error instanceof AdminOrderApiError && error.status === 503)
    },
  )
  await withFetch(
    async () => response({ data: order() }, 202),
    async () => {
      await assert.rejects(adminOrders.cancel('id'), (error) => error instanceof AdminOrderApiError && error.code === 'INVALID_RESPONSE')
    },
  )
})

test('filter/page URL hợp lệ round-trip; enum lạ/lặp/query lạ bị bỏ trước request', () => {
  const parsed = readAdminOrderFilters(new URLSearchParams('status=shipping&payment_status=paid&page=3&page_size=50&unexpected=1'))
  assert.deepEqual(parsed, { status: 'shipping', payment_status: 'paid', page: 3, page_size: 50 })
  assert.deepEqual(readAdminOrderFilters(adminOrderSearch(parsed)), parsed)
  assert.equal(adminOrderSearch(parsed).has('unexpected'), false)
  assert.deepEqual(readAdminOrderFilters(new URLSearchParams('status=bad&payment_status=bad&page=1e2&page_size=-1')), filter())
  assert.deepEqual(readAdminOrderFilters(new URLSearchParams('status=pending&status=shipping&page=0&page_size=999')), { ...filter(), page_size: 100 })
  assert.deepEqual(readAdminOrderFilters(new URLSearchParams('page=100001&page_size=1.5')), filter())
  assert.deepEqual(readAdminOrderFilters(new URLSearchParams('page=2&page=3&payment_status=paid&payment_status=unpaid')), filter())
})

test('đăng nhập giữ next tới admin orders hiện tại và filter/page', () => {
  for (const path of ['/admin/orders?status=shipping&page=2', '/admin/orders/order-1']) {
    const url = new URL(orderLoginPath(path), 'https://shop.example')
    assert.equal(url.pathname, '/dang-nhap')
    assert.equal(url.searchParams.get('next'), path)
  }
})
