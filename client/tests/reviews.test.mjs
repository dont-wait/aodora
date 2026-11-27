import { test } from 'node:test'
import assert from 'node:assert/strict'
import { formatReviewImages, guestReviewApi, parseReviewImages, ReviewApiError, reviewErrorMessage } from '../src/features/review/api.ts'
import { adminReviews, adminReviewSearch, deleteAdminReview, loadAdminReviews, readAdminReviewFilters } from '../src/features/admin/reviews.ts'

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
const adminReview = (over = {}) => ({
  id: 'r1', rating: 4, content: 'Đẹp', display_name: 'Khách A', images: [], verified_purchase: true,
  created_at: '2026-11-27T00:00:00Z', updated_at: '2026-11-27T00:00:00Z', product_id: 'p1', product_name: 'Áo', product_slug: 'ao',
  source: 'guest', order_number: 'AD-1', ...over,
})

test('review images round-trip and reject lines without a description', () => {
  const images = parseReviewImages(' https://cdn.example/a.jpg | Mặt trước \n\nhttps://cdn.example/b.jpg|Sau ')
  assert.deepEqual(images, [{ src: 'https://cdn.example/a.jpg', alt: 'Mặt trước' }, { src: 'https://cdn.example/b.jpg', alt: 'Sau' }])
  assert.deepEqual(parseReviewImages(formatReviewImages(images)), images)
  assert.throws(() => parseReviewImages('https://cdn.example/a.jpg'), /URL \| mô tả/)
  assert.throws(() => parseReviewImages('https://cdn.example/a.jpg |  '), /URL \| mô tả/)
})

test('guest review calls send the order token only as a header', async () => {
  const calls = []
  await withFetch(async (url, init) => {
    calls.push({ url: String(url), init })
    return response({ data: init.method === 'PUT' ? { id: 'r1', product_id: 'p 1' } : [] })
  }, async () => {
    await guestReviewApi.list('o/1', 'tok')
    await guestReviewApi.submit('o/1', 'p 1', 'tok', { rating: 5, content: 'x', images: [] })
  })
  assert.equal(calls[0].url, '/api/v1/guest-orders/o%2F1/reviews')
  assert.equal(calls[1].url, '/api/v1/guest-orders/o%2F1/reviews/p%201')
  for (const call of calls) {
    assert.equal(call.init.headers['X-Order-Access-Token'], 'tok')
    assert.ok(!call.url.includes('tok'))
  }
  assert.equal(calls[1].init.method, 'PUT')
  assert.deepEqual(JSON.parse(calls[1].init.body), { rating: 5, content: 'x', images: [] })
})

test('review errors map to clear messages', async () => {
  assert.match(reviewErrorMessage(new ReviewApiError(403, 'order_id', 'x', 'REVIEW_NOT_ELIGIBLE')), /đơn hàng đã nhận/)
  assert.match(reviewErrorMessage(new ReviewApiError(404, 'order_id', 'x', 'ORDER_NOT_FOUND')), /không còn quyền/)
  await withFetch(async () => response({ error: { code: 'INVALID_FIELD', field: 'rating', message: 'Điểm đánh giá phải từ 1 đến 5' } }, 400), async () => {
    await assert.rejects(guestReviewApi.submit('o', 'p', 'tok', { rating: 9, content: '', images: [] }), (error) => error instanceof ReviewApiError && error.field === 'rating')
  })
})

test('admin review filters drop unknown, repeated and out-of-range values', () => {
  assert.deepEqual(readAdminReviewFilters(new URLSearchParams('rating=9&page=0&page_size=500')), { rating: 0, page: 1, page_size: 100 })
  assert.deepEqual(readAdminReviewFilters(new URLSearchParams('rating=3&rating=4&page=2')), { rating: 0, page: 2, page_size: 20 })
  assert.equal(adminReviewSearch({ rating: 5, page: 1, page_size: 20 }).toString(), 'page=1&page_size=20&rating=5')
  assert.equal(adminReviewSearch({ rating: 0, page: 3, page_size: 50 }).toString(), 'page=3&page_size=50')
})

test('admin list validates payload and maps auth failures', async () => {
  await withFetch(async (url) => {
    assert.equal(String(url), '/api/v1/admin/reviews?page=1&page_size=20&rating=4')
    return response({ data: [adminReview()], total: 1, page: 1, page_size: 20 })
  }, async () => {
    const result = await loadAdminReviews({ rating: 4, page: 1, page_size: 20 })
    assert.equal(result.kind, 'ready')
    assert.equal(result.page.data[0].source, 'guest')
  })
  await withFetch(async () => response({ data: [adminReview({ rating: 7 })], total: 1, page: 1, page_size: 20 }), async () => {
    assert.equal((await loadAdminReviews({ rating: 0, page: 1, page_size: 20 })).kind, 'error')
  })
  for (const [status, kind] of [[401, 'unauthenticated'], [403, 'forbidden'], [503, 'error']]) {
    await withFetch(async () => response({ error: { code: 'X', message: 'x' } }, status), async () => {
      assert.equal((await loadAdminReviews({ rating: 0, page: 1, page_size: 20 })).kind, kind)
    })
  }
})

test('admin delete uses DELETE and reports a missing review', async () => {
  const calls = []
  await withFetch(async (url, init) => {
    calls.push([String(url), init.method])
    return response({ data: { id: 'r/1', deleted: true } })
  }, async () => {
    await adminReviews.remove('r/1')
  })
  assert.deepEqual(calls, [['/api/v1/admin/reviews/r%2F1', 'DELETE']])
  await withFetch(async () => response({ error: { code: 'NOT_FOUND', message: 'x' } }, 404), async () => {
    assert.equal((await deleteAdminReview('r1')).kind, 'not-found')
  })
})
