import { test } from 'node:test'
import assert from 'node:assert/strict'
import { randomBytes } from 'node:crypto'
import { ordersApi, OrderApiError } from '../src/features/orders/api.ts'
import { parseExactJSON, formatOrderMoney } from '../src/features/orders/money.ts'
import {
  createGuestToken,
  makeAttempt,
  saveAttempt,
  readAttempt,
  attemptQuoteExpired,
  clearAttempt,
  saveReceipt,
  saveOrderToken,
  readReceipt,
  readOrderToken,
  matchesIdentity,
  subtractPurchased,
  isDefinitiveOrderError,
  lookupAttempt,
  sendAttempt,
  quoteDifferences,
} from '../src/features/orders/checkout.ts'
import { completeCheckout } from '../src/features/orders/completion.ts'
import { consumeAccountPurchase } from '../src/features/orders/cartCleanup.ts'
import { loadCheckoutResult, loadOrderDetail } from '../src/features/orders/result.ts'
import { loadMyOrders, orderPageCount, readOrdersPage, startOrderRead } from '../src/features/orders/read.ts'
import { formatOrderDate, orderDeliveryAddress, orderLoginPath, orderPaymentLabel, orderStatusLabels } from '../src/features/orders/presentation.ts'

function memoryStorage() {
  const data = new Map()
  return { getItem: (key) => data.get(key) ?? null, setItem: (key, value) => data.set(key, value), removeItem: (key) => data.delete(key) }
}
const body = () => ({
  payment_method: 'cash_at_counter',
  shipping_address: { recipient: 'Khách mẫu', phone: '0900000000' },
  items: [{ product_id: 'p', sku: 'A', quantity: 2 }],
  quote_token: 'quote-from-test-server',
})
const snapshot = () => [{ product_id: 'p', sku: 'A', quantity: 2, name: 'Áo mẫu' }]
const order = () => ({
  id: 'order-1',
  order_number: 'AD-TEST',
  status: 'pending',
  payment_status: 'unpaid',
  payment_method: 'cash_at_counter',
  subtotal: '500000',
  shipping_fee: '0',
  total: '500000',
  shipping_address: body().shipping_address,
  items: [{ ...snapshot()[0], product_name: 'Áo mẫu', unit_price: '250000' }],
  created_at: '2026-11-17T00:00:00Z',
})
const response = (data, status = 200) => new Response(JSON.stringify(data), { status })
const authResponse = (id = null) => (id ? response({ data: { id, role: 'customer' } }) : response({ error: { message: 'Cần đăng nhập' } }, 401))
async function withFetch(fn, run) {
  const old = globalThis.fetch
  globalThis.fetch = fn
  try {
    await run()
  } finally {
    globalThis.fetch = old
  }
}
async function withStorage(run) {
  const old = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: memoryStorage() })
  try {
    await run(globalThis.localStorage)
  } finally {
    if (old) Object.defineProperty(globalThis, 'localStorage', old)
    else delete globalThis.localStorage
  }
}

test('guest token: 32 byte ngẫu nhiên → 43 ký tự base64url canonical không padding', () => {
  const bytes = randomBytes(32)
  let length = 0
  const token = createGuestToken({
    getRandomValues: (target) => {
      length = target.length
      target.set(bytes)
      return target
    },
  })
  assert.equal(length, 32)
  assert.equal(token.length, 43)
  assert.ok(/^[A-Za-z0-9_-]{43}$/.test(token))
  assert.ok(Buffer.from(token, 'base64url').equals(bytes), 'decode recovers the original bytes')
  assert.ok(Buffer.from(token, 'base64url').toString('base64url') === token, 'canonical encoding')
  assert.ok(createGuestToken() !== createGuestToken(), 'separate checkouts have separate secrets')
})

test('tiền JSON lớn và decimal giữ nguyên qua parse API và hiển thị, không qua float', async () => {
  const exact = '900719925474099312345678901234.56'
  assert.equal(parseExactJSON(`{"total":${exact}}`).total, exact)
  assert.equal(formatOrderMoney(exact), '900.719.925.474.099.312.345.678.901.234,56 ₫')
  assert.equal(formatOrderMoney('0.100000000000000001'), '0,100000000000000001 ₫')
  assert.equal(formatOrderMoney('0'), '0 ₫')
  assert.equal(formatOrderMoney('bad'), '—')
  await withFetch(
    async () =>
      new Response(
        `{"data":{"quote_token":"q","expires_at":"2026-11-17T00:15:00Z","subtotal":${exact},"shipping_fee":0,"total":${exact},"items":[{"product_id":"p","sku":"A","quantity":2,"unit_price":0.100000000000000001}]}}`,
      ),
    async () => {
      const q = await ordersApi.quote(body())
      assert.equal(q.total, exact)
      assert.equal(q.items[0].unit_price, '0.100000000000000001')
      assert.equal(q.items[0].quantity, 2)
      assert.equal(q.shipping_fee, '0')
    },
  )
})

test('parse exact không thay số trong chuỗi, escape hoặc ghi chú; từ chối JSON hỏng', () => {
  const value = { note: 'Áo "123" \\ đường 55', sku: '12-345', count: 3, enabled: true, empty: null }
  assert.deepEqual(parseExactJSON(JSON.stringify(value)), { ...value, count: '3' })
  assert.throws(() => parseExactJSON('{"total":1.}'))
})

test('API đủ 5 endpoint, include cookie, key/body nguyên vẹn và secret chỉ ở header', async () => {
  const calls = []
  const pending = makeAttempt(body(), snapshot(), null)
  await withFetch(
    async (url, options) => {
      calls.push({ url, ...options })
      if (url.includes('/quote'))
        return response({
          data: { quote_token: 'q', expires_at: '2026-11-17T00:15:00Z', subtotal: 5, shipping_fee: 0, total: 5, items: [{ ...snapshot()[0], unit_price: 5 }] },
        })
      if (url.includes('?page=')) return response({ data: [order()], total: 1, page: 2, page_size: 10 })
      return response({ data: order() }, options.method === 'POST' ? 201 : 200)
    },
    async () => {
      await ordersApi.quote(body())
      await ordersApi.create(pending)
      await ordersApi.get('id/with slash', pending.credential)
      await ordersApi.attempt(pending)
      const page = await ordersApi.list(2, 10)
      assert.equal(page.page, 2)
      assert.equal(page.total, 1)
      assert.equal(page.data[0].total, '500000')
    },
  )
  assert.equal(calls.length, 5)
  for (const call of calls) {
    assert.equal(call.credentials, 'include')
    assert.ok(!call.url.includes(pending.credential.token), 'secret never appears in URL')
  }
  assert.equal(calls[1].headers['Idempotency-Key'], pending.request_id)
  assert.equal(calls[1].body, pending.body)
  assert.ok(calls[1].headers['X-Order-Access-Token'] === pending.credential.token, 'correct guest header')
  assert.ok(calls[2].url.endsWith('/id%2Fwith%20slash'))
  assert.ok(calls[3].url.includes('/attempts/'))
  assert.ok(calls[4].url.endsWith('?page=2&page_size=10'))
})

test('tài khoản tuyệt đối không gửi guest header', async () => {
  const pending = makeAttempt(body(), snapshot(), 'a')
  const calls = []
  await withFetch(
    async (_url, options) => {
      calls.push(options)
      return response({ data: order() })
    },
    async () => {
      await ordersApi.create(pending)
      await ordersApi.get('id', { kind: 'account', customerId: 'a', token: createGuestToken() })
      await ordersApi.attempt(pending)
    },
  )
  assert.ok(calls.every((call) => !('X-Order-Access-Token' in call.headers)))
})

test('lỗi API giữ đủ status/code/field; HTML 503 và 202 không coi là thành công', async () => {
  await withFetch(
    async () => response({ error: { code: 'INVALID_FIELD', field: 'shipping_address.city', message: 'Thiếu tỉnh thành' } }, 400),
    async () => {
      await assert.rejects(
        ordersApi.quote(body()),
        (e) => e instanceof OrderApiError && e.status === 400 && e.code === 'INVALID_FIELD' && e.field === 'shipping_address.city',
      )
    },
  )
  await withFetch(
    async () => new Response('<html>unavailable</html>', { status: 503 }),
    async () => {
      await assert.rejects(ordersApi.quote(body()), (e) => e instanceof OrderApiError && e.status === 503)
    },
  )
  await withFetch(
    async () => response({ data: order() }, 202),
    async () => {
      await assert.rejects(ordersApi.create(makeAttempt(body(), snapshot(), null)), (e) => e instanceof OrderApiError && e.code === 'INVALID_RESPONSE')
    },
  )
})

test('snapshot chỉ trừ đúng SKU/lượng đã mua, giữ dòng và lượng thêm sau khi gửi', () => {
  const current = [
    { product_id: 'p', sku: 'A', quantity: 5, name: 'Áo A' },
    { product_id: 'p', sku: 'B', quantity: 2 },
    { product_id: 'new', sku: 'C', quantity: 3 },
  ]
  assert.deepEqual(subtractPurchased(current, snapshot()), [{ ...current[0], quantity: 3 }, current[1], current[2]])
  assert.equal(current[0].quantity, 5)
  assert.deepEqual(subtractPurchased([{ product_id: 'p', sku: 'A', quantity: 1 }], snapshot()), [])
  assert.deepEqual(subtractPurchased(current, [...snapshot(), ...snapshot()]), [{ ...current[0], quantity: 1 }, current[1], current[2]])
  assert.deepEqual(subtractPurchased(current.slice(1), snapshot()), current.slice(1))
})

test('account cleanup get giỏ mới nhất rồi update/remove, gồm dòng chuyển sang issues', async () => {
  const line = (sku, quantity) => ({
    product_id: 'p',
    product_slug: 'ao',
    sku,
    quantity,
    color: 'Đỏ',
    size: 'M',
    fulfillment_mode: 'ready',
    unit_price: 10,
    line_total: 10,
    stock: 20,
    available: true,
  })
  let current = { items: [line('A', 5), line('B', 2), line('NEW', 4)], issues: [{ ...line('SOLD', 1), available: false }], item_count: 12, subtotal: 0 }
  const calls = []
  const api = {
    get: async () => {
      calls.push('get')
      return structuredClone(current)
    },
    update: async (item) => {
      calls.push(['update', item.sku, item.quantity])
      current.items = current.items.map((old) => (old.sku === item.sku ? { ...old, quantity: item.quantity } : old))
      return current
    },
    remove: async (_id, sku) => {
      calls.push(['remove', sku])
      current.items = current.items.filter((item) => item.sku !== sku)
      current.issues = current.issues.filter((item) => item.sku !== sku)
      return current
    },
  }
  await consumeAccountPurchase(
    [...snapshot(), { product_id: 'p', sku: 'B', quantity: 2 }, { product_id: 'p', sku: 'SOLD', quantity: 1 }],
    api,
    () => {},
    () => {},
  )
  assert.deepEqual(calls, ['get', ['update', 'A', 3], ['remove', 'B'], ['remove', 'SOLD']])
  assert.deepEqual(
    current.items.map((item) => [item.sku, item.quantity]),
    [
      ['A', 3],
      ['NEW', 4],
    ],
  )
  assert.equal(current.issues.length, 0)
})

test('lưu/khôi phục attempt qua reload giữ UUID, body, credential và snapshot', () => {
  const storage = memoryStorage()
  const pending = makeAttempt(body(), snapshot(), null)
  saveAttempt(pending, storage)
  const restored = readAttempt(storage)
  assert.equal(restored.request_id, pending.request_id)
  assert.equal(restored.body, pending.body)
  assert.ok(restored.credential.token === pending.credential.token, 'same credential after reload')
  assert.deepEqual(restored.snapshot, pending.snapshot)
  assert.throws(() => saveAttempt(makeAttempt(body(), snapshot(), null), storage))
  clearAttempt('different-request', storage)
  assert.ok(readAttempt(storage))
  clearAttempt(pending.request_id, storage)
  assert.equal(readAttempt(storage), null)
})

test('storage hỏng bị chặn, storage đầy không thể lưu attempt trước khi gửi', () => {
  const storage = memoryStorage()
  storage.setItem('aodora.orders.attempt.v1', '{broken')
  assert.throws(() => readAttempt(storage), /Không đọc được/)
  assert.throws(
    () =>
      saveAttempt(makeAttempt(body(), snapshot(), null), {
        ...memoryStorage(),
        setItem: () => {
          throw new Error('storage full')
        },
      }),
    /storage full/,
  )
})

test('mất mạng + reload + lookup 404: retry cùng key/body/token, không tạo attempt mới', async () => {
  const storage = memoryStorage()
  const pending = makeAttempt(body(), snapshot(), null)
  saveAttempt(pending, storage)
  const posts = []
  let first = true
  await withFetch(
    async (url, options) => {
      if (url.endsWith('/auth/me')) return authResponse()
      if (url.includes('/attempts/')) return response({ error: { code: 'NOT_FOUND' } }, 404)
      posts.push(options)
      if (first) {
        first = false
        throw new TypeError('network disconnected')
      }
      return response({ data: order() })
    },
    async () => {
      await assert.rejects(sendAttempt(pending, null))
      const restored = readAttempt(storage)
      assert.equal(await lookupAttempt(restored, null), null)
      assert.equal((await sendAttempt(restored, null)).id, 'order-1')
    },
  )
  assert.equal(posts.length, 2)
  assert.equal(posts[0].body, posts[1].body)
  assert.equal(posts[0].headers['Idempotency-Key'], posts[1].headers['Idempotency-Key'])
  assert.ok(posts[0].headers['X-Order-Access-Token'] === posts[1].headers['X-Order-Access-Token'], 'same secret on retry')
})

test('lookup có đơn thì dùng đơn; 401/503 không bị hiểu là chưa có đơn', async () => {
  const pending = makeAttempt(body(), snapshot(), null)
  let posts = 0
  await withFetch(
    async (url, options) => {
      if (url.endsWith('/auth/me')) return authResponse()
      if (options.method === 'POST') posts++
      return response({ data: order() })
    },
    async () => {
      assert.equal((await lookupAttempt(pending, null)).order_number, 'AD-TEST')
    },
  )
  assert.equal(posts, 0)
  for (const status of [401, 503])
    await withFetch(
      async (url) => (url.endsWith('/auth/me') ? authResponse() : response({ error: { code: 'ORDER_UNAVAILABLE' } }, status)),
      async () => {
        await assert.rejects(lookupAttempt(pending, null), (e) => e instanceof OrderApiError && e.status === status)
      },
    )
})

test('đổi identity local hoặc cookie bên server: không gửi đơn/lookup theo sai scope', async () => {
  const guest = makeAttempt(body(), snapshot(), null)
  const account = makeAttempt(body(), snapshot(), 'a')
  let calls = 0
  assert.equal(matchesIdentity(guest.credential, 'a'), false)
  assert.equal(matchesIdentity(account.credential, 'b'), false)
  await withFetch(
    async () => {
      calls++
      throw new Error('unexpected')
    },
    async () => {
      await assert.rejects(lookupAttempt(guest, 'a'))
      await assert.rejects(sendAttempt(account, 'b'))
    },
  )
  assert.equal(calls, 0)
  await withFetch(
    async (url) => {
      assert.ok(url.endsWith('/auth/me'))
      calls++
      return authResponse('b')
    },
    async () => {
      await assert.rejects(lookupAttempt(account, 'a'))
      await assert.rejects(sendAttempt(guest, null))
    },
  )
  assert.equal(calls, 2)
})

test('chỉ 4xx nghiệp vụ kết thúc attempt; auth/404/5xx/network/mixed credential giữ lại', () => {
  for (const code of ['QUOTE_CHANGED', 'QUOTE_EXPIRED', 'OUT_OF_STOCK', 'IDEMPOTENCY_CONFLICT', 'INVALID_BODY', 'INVALID_FIELD'])
    assert.equal(isDefinitiveOrderError(new OrderApiError(409, code, '', 'lỗi')), true)
  for (const status of [401, 403, 404, 503]) assert.equal(isDefinitiveOrderError(new OrderApiError(status, 'NOT_FOUND', '', 'lỗi')), false)
  assert.equal(isDefinitiveOrderError(new OrderApiError(400, 'INVALID_FIELD', 'X-Order-Access-Token', 'phiên đổi')), false)
  assert.equal(isDefinitiveOrderError(new TypeError('network')), false)
})

test('token guest theo order id và receipt bền qua reload; account không sinh guest token', () => {
  const storage = memoryStorage()
  const pending = makeAttempt(body(), snapshot(), null)
  saveOrderToken(order().id, pending.credential.token, storage)
  saveReceipt({ order_id: order().id, order_number: order().order_number, owner: null, request_id: pending.request_id, cleanup: 'done' }, storage)
  assert.ok(readOrderToken('order-1', storage) === pending.credential.token, 'per-order token persists')
  assert.equal(readOrderToken('other-order', storage), null)
  assert.equal(readReceipt('order-1', storage).order_number, 'AD-TEST')
  const account = makeAttempt(body(), snapshot(), 'a')
  saveReceipt({ order_id: 'account-order', order_number: order().order_number, owner: 'a', request_id: account.request_id, cleanup: 'done' }, storage)
  assert.equal(readOrderToken('account-order', storage), null)
})

test('success/recovery dọn giỏ một lần; lỗi giỏ không biến đơn đã lưu thành thất bại', async () => {
  await withStorage(async (storage) => {
    const pending = makeAttempt(body(), snapshot(), null)
    saveAttempt(pending, storage)
    let cleanups = 0
    assert.equal(
      (
        await completeCheckout(order(), pending, async (lines, owner) => {
          cleanups++
          assert.deepEqual(lines, pending.snapshot)
          assert.equal(owner, null)
        })
      ).cleanup,
      'done',
    )
    assert.equal(readAttempt(storage), null)
    await completeCheckout(order(), pending, async () => {
      cleanups++
    })
    assert.equal(cleanups, 1)
    assert.ok(readOrderToken('order-1', storage))
  })
  await withStorage(async (storage) => {
    const pending = makeAttempt(body(), snapshot(), 'a')
    saveAttempt(pending, storage)
    assert.equal(
      (
        await completeCheckout(order(), pending, async () => {
          throw new TypeError('cart offline')
        })
      ).cleanup,
      'failed',
    )
    assert.equal(readReceipt('order-1', storage).order_id, 'order-1')
    assert.equal(readAttempt(storage), null)
    await completeCheckout(order(), pending, async () => assert.fail('must not subtract twice'))
  })
})

test('chưa lưu được receipt/quyền guest thì không xóa attempt và không dọn giỏ', async () => {
  await withStorage(async (storage) => {
    const pending = makeAttempt(body(), snapshot(), null)
    saveAttempt(pending, storage)
    storage.setItem = () => {
      throw new Error('storage full')
    }
    await assert.rejects(completeCheckout(order(), pending, async () => assert.fail('must persist first')))
    assert.ok(readAttempt(storage))
  })
})

test('quote đổi giá/phí/số lượng hiện khác biệt exact, không tính tổng bằng float', () => {
  const before = { subtotal: '10.01', shipping_fee: '0', total: '10.01', items: [{ product_id: 'p', sku: 'A', quantity: 1, unit_price: '10.01' }] }
  const after = {
    subtotal: '20.000000000000000002',
    shipping_fee: '1.000000000000000001',
    total: '21.000000000000000003',
    items: [{ product_id: 'p', sku: 'A', quantity: 2, unit_price: '10.000000000000000001' }],
  }
  assert.deepEqual(quoteDifferences(null, after), [])
  assert.deepEqual(quoteDifferences(before, before), [])
  const changes = quoteDifferences(before, after)
  assert.ok(changes.some((text) => text.includes('21,000000000000000003')))
  assert.ok(changes.some((text) => text.includes('số lượng 1 → 2')))
  assert.ok(changes.some((text) => text.includes('10,000000000000000001')))
})

test('response 200 thiếu quote/đơn không coi là xác nhận thành công', async () => {
  await withFetch(
    async () => response({ data: {} }),
    async () => {
      await assert.rejects(ordersApi.quote(body()))
      await assert.rejects(ordersApi.create(makeAttempt(body(), snapshot(), null)))
    },
  )
  const storage = memoryStorage()
  storage.setItem('aodora.orders.receipt.v1.bad', JSON.stringify({ order: { id: 'bad', items: [] } }))
  assert.equal(readReceipt('bad', storage), null)
})

test('receipt chỉ ghi metadata allowlist, không lưu PII/order/credential dù caller có field thừa', async () => {
  await withStorage(async (storage) => {
    const pending = makeAttempt(body(), snapshot(), null)
    await completeCheckout({ ...order(), note: 'private-note' }, pending, async () => {})
    const key = 'aodora.orders.receipt.v1.order-1'
    const receipt = readReceipt('order-1', storage)
    assert.deepEqual(Object.keys(JSON.parse(storage.getItem(key))).sort(), ['cleanup', 'order_id', 'order_number', 'owner', 'request_id'])
    for (const forbidden of ['shipping_address', 'phone', 'recipient', 'items', 'note', 'credential', 'token', 'private-note'])
      assert.ok(!storage.getItem(key).includes(forbidden), `no ${forbidden} in receipt`)
    saveReceipt({ ...receipt, order: order(), shipping_address: order().shipping_address, phone: 'private-phone' }, storage)
    assert.deepEqual(JSON.parse(storage.getItem(key)), receipt)
    assert.ok(readOrderToken('order-1', storage) === pending.credential.token, 'guest credential exists only in its separate key')
  })
})

test('receipt cũ được migrate tại chỗ, giữ cleanup/id/token riêng và loại PII khỏi storage', () => {
  const storage = memoryStorage()
  const pending = makeAttempt(body(), snapshot(), null)
  const key = 'aodora.orders.receipt.v1.order-1'
  storage.setItem(
    key,
    JSON.stringify({ order: { ...order(), note: 'legacy-private-note' }, credential: pending.credential, request_id: pending.request_id, cleanup: 'failed' }),
  )
  const metadata = readReceipt('order-1', storage)
  assert.deepEqual(metadata, { order_id: 'order-1', order_number: 'AD-TEST', request_id: pending.request_id, owner: null, cleanup: 'failed' })
  assert.deepEqual(JSON.parse(storage.getItem(key)), metadata)
  for (const forbidden of ['shipping_address', 'phone', 'recipient', 'items', 'legacy-private-note', 'credential'])
    assert.ok(!storage.getItem(key).includes(forbidden))
  assert.ok(readOrderToken('order-1', storage) === pending.credential.token, 'legacy guest token migrated separately')
})

test('kết quả guest luôn GET qua server với token header, hiển thị trạng thái mới thay snapshot cũ', async () => {
  await withStorage(async (storage) => {
    const pending = makeAttempt(body(), snapshot(), null)
    storage.setItem(
      'aodora.orders.receipt.v1.order-1',
      JSON.stringify({ order: order(), credential: pending.credential, request_id: pending.request_id, cleanup: 'failed' }),
    )
    let calls = 0
    await withFetch(
      async (url, options) => {
        calls++
        assert.ok(url.endsWith('/api/v1/orders/order-1'))
        assert.equal(options.method, 'GET')
        assert.equal(options.credentials, 'include')
        assert.ok(options.headers['X-Order-Access-Token'] === pending.credential.token, 'per-order guest header')
        assert.ok(!url.includes(pending.credential.token), 'no secret in URL')
        return response({ data: { ...order(), status: 'delivered', payment_status: 'paid' } })
      },
      async () => {
        const result = await loadCheckoutResult('order-1', null)
        assert.equal(result.kind, 'ready')
        assert.equal(result.order.status, 'delivered')
        assert.equal(result.order.payment_status, 'paid')
        assert.equal(result.guest, true)
        assert.equal(result.receipt.cleanup, 'failed')
      },
    )
    assert.equal(calls, 1)
  })
})

test('kết quả account dùng cookie, không guest header; vẫn GET được khi không có receipt', async () => {
  await withStorage(async (storage) => {
    saveOrderToken('order-1', createGuestToken(), storage)
    await withFetch(
      async (url, options) => {
        assert.ok(url.endsWith('/api/v1/orders/order-1'))
        assert.equal(options.method, 'GET')
        assert.equal(options.credentials, 'include')
        assert.ok(!('X-Order-Access-Token' in options.headers))
        return response({ data: { ...order(), status: 'confirmed' } })
      },
      async () => {
        const result = await loadCheckoutResult('order-1', 'account-a')
        assert.equal(result.kind, 'ready')
        assert.equal(result.order.status, 'confirmed')
        assert.equal(result.receipt, null)
        assert.equal(result.guest, false)
      },
    )
  })
})

test('GET kết quả 401/404 không trả đơn/PII cũ, kể cả khi có legacy receipt', async () => {
  for (const [status, kind] of [
    [401, 'unauthenticated'],
    [404, 'not-found'],
  ])
    await withStorage(async (storage) => {
      storage.setItem(
        'aodora.orders.receipt.v1.order-1',
        JSON.stringify({ order: order(), credential: { kind: 'account', customerId: 'a' }, request_id: 'old-request', cleanup: 'done' }),
      )
      await withFetch(
        async () => response({ error: { code: status === 401 ? 'AUTH_REQUIRED' : 'NOT_FOUND' } }, status),
        async () => {
          const result = await loadCheckoutResult('order-1', 'a')
          assert.deepEqual(result, { kind })
          assert.ok(!('order' in result))
          assert.ok(!('receipt' in result))
          assert.ok(!storage.getItem('aodora.orders.receipt.v1.order-1').includes('shipping_address'))
        },
      )
    })
})

test('GET lỗi mạng/5xx trả trạng thái thử lại; retry gọi server, không fallback bản cũ', async () => {
  await withStorage(async () => {
    let calls = 0
    await withFetch(
      async () => {
        calls++
        if (calls === 1) throw new TypeError('network disconnected')
        if (calls === 2) return response({ error: { code: 'ORDER_UNAVAILABLE' } }, 503)
        return response({ data: { ...order(), status: 'delivered' } })
      },
      async () => {
        assert.deepEqual(await loadCheckoutResult('order-1', 'a'), { kind: 'error' })
        assert.deepEqual(await loadCheckoutResult('order-1', 'a'), { kind: 'error' })
        const result = await loadCheckoutResult('order-1', 'a')
        assert.equal(result.kind, 'ready')
        assert.equal(result.order.status, 'delivered')
      },
    )
    assert.equal(calls, 3)
  })
})

test('guest thiếu token riêng không dùng mã đơn/receipt cũ để cấp quyền xem', async () => {
  await withStorage(async (storage) => {
    saveReceipt({ order_id: 'order-1', order_number: 'AD-TEST', request_id: 'request', owner: null, cleanup: 'done' }, storage)
    await withFetch(
      async () => assert.fail('must not guess guest access from an order number'),
      async () => {
        assert.deepEqual(await loadCheckoutResult('order-1', null), { kind: 'not-found' })
      },
    )
  })
})

test('danh sách của tôi chỉ dùng cookie; giữ phân trang server trả và tổng tiền exact', async () => {
  await withStorage(async (storage) => {
    saveOrderToken('order-1', createGuestToken(), storage)
    await withFetch(
      async (url, options) => {
        assert.ok(url.endsWith('/api/v1/orders?page=2&page_size=200'))
        assert.equal(options.method, 'GET')
        assert.equal(options.credentials, 'include')
        assert.ok(!('X-Order-Access-Token' in options.headers))
        return response({
          data: [{ ...order(), total: '900719925474099312345678901234.56', status: 'delivered', payment_status: 'unpaid' }],
          total: 201,
          page: 2,
          page_size: 100,
        })
      },
      async () => {
        const result = await loadMyOrders(2, 200)
        assert.equal(result.kind, 'ready')
        assert.equal(result.page.page, 2)
        assert.equal(result.page.page_size, 100)
        assert.equal(result.page.total, 201)
        assert.equal(orderPageCount(result.page), 3)
        assert.equal(formatOrderMoney(result.page.data[0].total), '900.719.925.474.099.312.345.678.901.234,56 ₫')
        assert.deepEqual(orderStatusLabels(result.page.data[0]), { fulfillment: 'Đã nhận áo', payment: 'Chưa thu tiền' })
      },
    )
  })
})

test('danh sách rỗng và số trang lấy từ metadata, không từ số dòng ở trang hiện tại', async () => {
  await withFetch(
    async () => response({ data: [], total: 0, page: 1, page_size: 20 }),
    async () => {
      const result = await loadMyOrders(1, 20)
      assert.equal(result.kind, 'ready')
      assert.deepEqual(result.page.data, [])
      assert.equal(result.page.total, 0)
      assert.equal(orderPageCount(result.page), 1)
    },
  )
  assert.equal(orderPageCount({ total: 41, page_size: 20 }), 3)
  assert.equal(readOrdersPage('2'), 2)
  assert.equal(readOrdersPage('100000'), 100000)
  for (const invalid of [null, '', '0', '-1', '1.5', '1e2', '100001', 'not-a-page']) assert.equal(readOrdersPage(invalid), 1)
})

test('danh sách 401 xóa kết quả để chuyển đăng nhập; network/5xx không giữ đơn cũ', async () => {
  let count = 0
  await withFetch(
    async () => {
      count++
      if (count === 1) return response({ data: [order()], total: 1, page: 1, page_size: 20 })
      if (count === 2) return response({ error: { code: 'AUTH_REQUIRED' } }, 401)
      if (count === 3) return response({ error: { code: 'ORDER_UNAVAILABLE' } }, 503)
      throw new TypeError('network disconnected')
    },
    async () => {
      assert.equal((await loadMyOrders(1, 20)).kind, 'ready')
      assert.deepEqual(await loadMyOrders(1, 20), { kind: 'unauthenticated' })
      assert.deepEqual(await loadMyOrders(1, 20), { kind: 'error' })
      assert.deepEqual(await loadMyOrders(1, 20), { kind: 'error' })
    },
  )
})

test('nhãn nhận áo và tiền độc lập cho đầy đủ enum, kể cả delivered + unpaid', () => {
  const delivery = { pending: 'Chờ xác nhận', confirmed: 'Đã xác nhận', shipping: 'Đang giao áo', delivered: 'Đã nhận áo', cancelled: 'Đã hủy' }
  const payment = { unpaid: 'Chưa thu tiền', paid: 'Đã thu đủ tiền', refunded: 'Đã hoàn tiền' }
  for (const [status, fulfillment] of Object.entries(delivery))
    for (const [payment_status, label] of Object.entries(payment)) {
      assert.deepEqual(orderStatusLabels({ status, payment_status }), { fulfillment, payment: label })
    }
})

test('chi tiết chỉ hiện địa chỉ giao khi COD, tại quầy không hiện dù response còn field địa chỉ', () => {
  const address = { recipient: 'Khách', phone: '0900000000', line1: '12 Đường A', ward: 'Phường B', district: 'Quận C', city: 'Thành phố D' }
  assert.equal(orderDeliveryAddress({ payment_method: 'cod', shipping_address: address }), '12 Đường A, Phường B, Quận C, Thành phố D')
  assert.equal(orderDeliveryAddress({ payment_method: 'cash_at_counter', shipping_address: address }), null)
  assert.equal(orderPaymentLabel('cash_at_counter'), 'Nhận và thanh toán tại quầy')
  assert.ok(orderPaymentLabel('cod').includes('COD'))
  assert.ok(formatOrderDate('2026-11-17T00:00:00Z').includes('2026'))
  assert.equal(formatOrderDate('invalid-date'), 'Chưa có ngày tạo')
})

test('đổi id chi tiết guest chỉ dùng token của id mới; thiếu token không gọi server bằng token cũ', async () => {
  await withStorage(async (storage) => {
    const first = createGuestToken()
    const second = createGuestToken()
    const calls = []
    saveOrderToken('first-order', first, storage)
    saveOrderToken('second-order', second, storage)
    await withFetch(
      async (url, options) => {
        calls.push(url)
        const isFirst = url.endsWith('/first-order')
        assert.ok(options.headers['X-Order-Access-Token'] === (isFirst ? first : second), 'token matches only the requested id')
        return response({ data: { ...order(), id: isFirst ? 'first-order' : 'second-order' } })
      },
      async () => {
        assert.equal((await loadOrderDetail('first-order', null)).order.id, 'first-order')
        assert.equal((await loadOrderDetail('second-order', null)).order.id, 'second-order')
        assert.deepEqual(await loadOrderDetail('unknown-order', null), { kind: 'not-found' })
      },
    )
    assert.equal(calls.length, 2)
  })
})

test('chi tiết tài khoản không gửi token guest của id khác; server 404 không lộ đơn cũ', async () => {
  await withStorage(async (storage) => {
    saveOrderToken('order-1', createGuestToken(), storage)
    await withFetch(
      async (_url, options) => {
        assert.equal(options.credentials, 'include')
        assert.ok(!('X-Order-Access-Token' in options.headers))
        return response({ error: { code: 'NOT_FOUND' } }, 404)
      },
      async () => {
        assert.deepEqual(await loadOrderDetail('order-1', 'account-b'), { kind: 'not-found' })
      },
    )
  })
})

test('đổi id/phiên/trang hủy nhận response cũ, response về muộn không ghi đè kết quả mới', async () => {
  let oldResolve
  let nextResolve
  const old = new Promise((resolve) => {
    oldResolve = resolve
  })
  const next = new Promise((resolve) => {
    nextResolve = resolve
  })
  const received = []
  const cancelOld = startOrderRead(
    () => old,
    (result) => received.push(result),
  )
  cancelOld()
  const cancelNext = startOrderRead(
    () => next,
    (result) => received.push(result),
  )
  nextResolve({ kind: 'ready', order: { id: 'next-id' } })
  oldResolve({ kind: 'ready', order: { id: 'old-id' } })
  await Promise.all([old, next])
  assert.deepEqual(received, [{ kind: 'ready', order: { id: 'next-id' } }])
  cancelNext()
})

test('response danh sách cũ không khôi phục PII sau khi request mới nhận 401', async () => {
  let resolveOld
  const old = new Promise((resolve) => {
    resolveOld = resolve
  })
  let visible
  const cancel = startOrderRead(
    () => old,
    (result) => {
      visible = result
    },
  )
  cancel()
  await withFetch(
    async () => response({ error: { code: 'AUTH_REQUIRED' } }, 401),
    async () => {
      visible = await loadMyOrders(2, 20)
      resolveOld({ kind: 'ready', page: { data: [order()], total: 1, page: 1, page_size: 20 } })
      await old
      assert.deepEqual(visible, { kind: 'unauthenticated' })
    },
  )
})

test('link đăng nhập giữ next đúng chi tiết hoặc danh sách và không thêm secret', () => {
  for (const path of ['/don-hang/order-1', '/tai-khoan/don-hang?page=2']) {
    const url = new URL(orderLoginPath(path), 'https://shop.example')
    assert.equal(url.pathname, '/dang-nhap')
    assert.equal(url.searchParams.get('next'), path)
    assert.deepEqual([...url.searchParams.keys()], ['next'])
  }
})

test('saved attempts remember when their quote stops being accepted', () => {
  const pending = makeAttempt(body(), snapshot(), 'a', '2026-11-28T02:00:00Z')
  assert.equal(pending.quote_expires_at, '2026-11-28T02:00:00Z')
  assert.equal(attemptQuoteExpired(pending, Date.parse('2026-11-28T01:59:00Z')), false)
  assert.equal(attemptQuoteExpired(pending, Date.parse('2026-11-28T02:00:01Z')), true)
  assert.equal(attemptQuoteExpired(makeAttempt(body(), snapshot(), 'a')), false)
})
