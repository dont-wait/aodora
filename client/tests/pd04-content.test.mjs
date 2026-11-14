import { test } from 'node:test'
import assert from 'node:assert/strict'
import { blankStory, editableContent, moveEntry, validateEditorialInput } from '../src/features/admin/content.ts'
import { adminCatalog } from '../src/features/admin/catalog.ts'

const record = {
  id: '670000000000000000000001', name: 'Mẫu A', version: 2,
  story: { eyebrow: 'Văn hóa', title: 'Câu chuyện riêng', paragraphs: ['Đoạn A', 'Đoạn B', 'Đoạn C'] },
  fabric: { title: 'Vải mẫu', care: [{ title: 'Giặt', description: 'Giặt tay' }] },
  craft: null,
}
test('nội dung từng mẫu lấy từ bản ghi riêng, không thay bằng chuyện mẫu', () => {
  const a = editableContent(record)
  const b = editableContent({ ...record, id: '670000000000000000000002', story: null, fabric: null })
  assert.deepEqual(a.story.paragraphs, ['Đoạn A', 'Đoạn B', 'Đoạn C'])
  assert.equal(b.story, null)
  a.story.paragraphs[0] = 'Đổi đoạn'
  assert.equal(record.story.paragraphs[0], 'Đoạn A')
  assert.deepEqual(a.fabric.care, [{ title: 'Giặt', description: 'Giặt tay' }])
  assert.deepEqual(a.craft, null)
})
test('đổi thứ tự đoạn kể không làm thay đổi mảng cũ', () => {
  const source = ['Một', 'Hai', 'Ba']
  const result = moveEntry(source, 2, 0)
  assert.deepEqual(result, ['Ba', 'Một', 'Hai'])
  assert.deepEqual(source, ['Một', 'Hai', 'Ba'])
})
test('không chấp nhận đoạn rỗng hoặc ảnh thiếu alt', () => {
  const v = { story: blankStory(), fabric: null, craft: null, expected_version: 0 }
  v.story.paragraphs = ['Đoạn 1', '  ']
  assert.throws(() => validateEditorialInput(v), /phải có nội dung/)
  v.story.paragraphs = ['Đoạn 1']
  v.story.image = { src: 'https://example.test/p.png', alt: '' }
  assert.throws(() => validateEditorialInput(v), /cả URL và mô tả/)
  v.story.image.alt = 'Mô tả ảnh'
  assert.doesNotThrow(() => validateEditorialInput(v))
})
test('API biên tập chỉ gửi nội dung và phiên bản, không gửi tồn kho', async () => {
  const previous = globalThis.fetch
  const requests = []
  globalThis.fetch = async (url, options) => {
    requests.push({ url, options })
    return { ok: true, status: 200, json: async () => ({ data: record }) }
  }
  try {
    assert.equal((await adminCatalog.content(record.id)).id, record.id)
    const input = editableContent(record)
    await adminCatalog.updateContent(record.id, input)
    assert.equal(requests.length, 2)
    assert.equal(requests[0].options.method, 'GET')
    assert.equal(requests[1].options.method, 'PUT')
    assert.equal(requests[1].options.credentials, 'include')
    assert.ok(requests[1].url.endsWith(`/api/v1/admin/products/${record.id}/content`))
    const body = JSON.parse(requests[1].options.body)
    assert.deepEqual(Object.keys(body).sort(), ['craft', 'expected_version', 'fabric', 'story'])
    assert.equal(body.expected_version, 2)
    assert.equal(body.story.paragraphs.length, 3)
  } finally { globalThis.fetch = previous }
})
