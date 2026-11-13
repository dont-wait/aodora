import test from 'node:test'
import assert from 'node:assert/strict'
import { parseImageLines, parseVariantLines } from '../src/features/admin/catalog.ts'

test('nhiều SKU phải có màu, cỡ và tồn kho rõ ràng', () => {
 assert.deepEqual(parseVariantLines('AO-DO-M | Đỏ | M | 10\nAO-XANH-L | Xanh | L | 0'), [
  { sku:'AO-DO-M',color:'Đỏ',size:'M',stock:10 },{ sku:'AO-XANH-L',color:'Xanh',size:'L',stock:0 },
 ])
 assert.throws(() => parseVariantLines('AO-DO-M | Đỏ | M | -1'))
 assert.throws(() => parseVariantLines('AO-DO-M | Đỏ | M'))
})
test('hình ảnh phải có cả src và alt', () => {
 assert.deepEqual(parseImageLines('https://cdn.example.test/a.jpg | Ảnh chính'),[{src:'https://cdn.example.test/a.jpg',alt:'Ảnh chính'}])
 assert.throws(() => parseImageLines('https://cdn.example.test/a.jpg'))
})
