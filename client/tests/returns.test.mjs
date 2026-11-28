import { test } from 'node:test'
import assert from 'node:assert/strict'
import { draftBody, evidenceKindOf, evidenceProblem, returnDeadline, timeLeft, validateReturnDraft, MAX_VIDEO_BYTES } from '../src/features/returns/returns.ts'

test('deadline is 3 days after delivery and counts down', () => {
  const d = returnDeadline('2026-11-25T10:00:00Z')
  assert.equal(d.toISOString(), '2026-11-28T10:00:00.000Z')
  assert.equal(timeLeft(d, Date.parse('2026-11-26T05:00:00Z')), 'còn 2 ngày 5 giờ')
  assert.equal(timeLeft(d, Date.parse('2026-11-28T09:20:00Z')), 'còn 40 phút')
  assert.equal(timeLeft(d, Date.parse('2026-11-28T10:00:01Z')), '')
})

test('evidence accepts photos and videos only, with limits', () => {
  assert.equal(evidenceKindOf({ type: 'video/quicktime', name: 'a.MOV' }), 'video')
  assert.equal(evidenceKindOf({ type: '', name: 'shot.JPG' }), 'image')
  assert.match(evidenceProblem({ type: 'image/heic', name: 'IMG.HEIC', size: 1 }, []), /HEIC/)
  assert.match(evidenceProblem({ type: 'application/pdf', name: 'a.pdf', size: 1 }, []), /chỉ nhận/)
  assert.match(evidenceProblem({ type: 'video/mp4', name: 'v.mp4', size: MAX_VIDEO_BYTES + 1 }, []), /50 MB/)
  assert.match(evidenceProblem({ type: 'video/mp4', name: 'v.mp4', size: 10 }, [{ kind: 'video' }, { kind: 'video' }]), /2 video/)
  assert.equal(evidenceProblem({ type: 'image/png', name: 'a.png', size: 10 }, [{ kind: 'video' }]), '')
})

test('draft needs items, reason and at least one evidence', () => {
  const empty = { quantities: {}, reason: '', note: '', evidence: [] }
  assert.deepEqual(Object.keys(validateReturnDraft(empty)).sort(), ['evidence', 'items', 'reason'])
  const other = { quantities: { S1: 1 }, reason: 'other', note: 'ngắn', evidence: [{ url: 'u.jpg', kind: 'image' }] }
  assert.deepEqual(Object.keys(validateReturnDraft(other)), ['note'])
  const ok = { quantities: { S1: 1, S2: 0 }, reason: 'defect', note: '  bung chỉ ', evidence: [{ url: 'u.jpg', kind: 'image' }] }
  assert.deepEqual(validateReturnDraft(ok), {})
  assert.deepEqual(draftBody('o1', ok), { order_id: 'o1', items: [{ sku: 'S1', quantity: 1 }], reason: 'defect', note: 'bung chỉ', evidence: [{ url: 'u.jpg' }] })
})
