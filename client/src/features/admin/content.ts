import type { Image } from './catalog'

// Editorial content is kept apart from product pricing and inventory inputs.
export type StoryContent = { eyebrow: string; title: string; paragraphs: string[]; image: Image | null }
export type ContentItem = { icon?: string; title: string; description: string }
export type CraftStep = { title: string; description: string }
export type FabricContent = { eyebrow: string; title: string; description: string; image: Image | null; care: ContentItem[] }
export type CraftContent = { eyebrow: string; title: string; description: string; image: Image | null; steps: CraftStep[] }
export type ProductContentInput = {
  story: StoryContent | null
  fabric: FabricContent | null
  craft: CraftContent | null
  expected_version: number
}
export type ProductContentRecord = Omit<ProductContentInput, 'expected_version'> & {
  id: string; name: string; version: number
}
export type MetadataWrite = { from: number; to: number }

// Advance only across metadata writes made by this page. A stale editorial
// snapshot must still conflict if another admin changed the product.
export function synchronizeContentVersion(data: ProductContentInput, writes: MetadataWrite[]): ProductContentInput {
  let version = data.expected_version
  for (const write of writes) if (version === write.from) version = write.to
  return version === data.expected_version ? data : { ...data, expected_version: version }
}
export const blankStory = (): StoryContent => ({ eyebrow: '', title: '', image: null, paragraphs: [] })
export const blankFabric = (): FabricContent => ({ eyebrow: '', title: '', description: '', image: null, care: [] })
export const blankCraft = (): CraftContent => ({ eyebrow: '', title: '', description: '', image: null, steps: [] })

// Legacy documents may omit entire sections or arrays. Never invent editorial text.
export function editableContent(data: ProductContentRecord): ProductContentInput {
  return {
    expected_version: data.version ?? 0,
    story: data.story ? { ...data.story, image: data.story.image ?? null, paragraphs: [...(data.story.paragraphs ?? [])] } : null,
    fabric: data.fabric ? { ...data.fabric, image: data.fabric.image ?? null, care: (data.fabric.care ?? []).map((v) => ({ ...v })) } : null,
    craft: data.craft ? { ...data.craft, image: data.craft.image ?? null, steps: (data.craft.steps ?? []).map((v) => ({ ...v })) } : null,
  }
}

export function moveEntry<T>(list: T[], from: number, to: number): T[] {
  if (from < 0 || to < 0 || from >= list.length || to >= list.length || from === to) return list
  const copy = [...list]
  const [entry] = copy.splice(from, 1)
  copy.splice(to, 0, entry)
  return copy
}

export function validateEditorialInput(data: ProductContentInput): void {
  if (!Number.isSafeInteger(data.expected_version) || data.expected_version < 0) throw new Error('Phiên bản nội dung không hợp lệ')
  if (data.story && (data.story.paragraphs.length > 30 || data.story.paragraphs.some((p) => !p.trim()))) {
    throw new Error('Mỗi đoạn câu chuyện phải có nội dung (tối đa 30 đoạn)')
  }
  if (data.fabric && (data.fabric.care.length > 30 || data.fabric.care.some((c) => !c.title.trim() || !c.description.trim()))) {
    throw new Error('Điền đầy đủ tiêu đề và hướng dẫn bảo quản (tối đa 30 mục)')
  }
  if (data.craft && (data.craft.steps.length > 30 || data.craft.steps.some((s) => !s.title.trim() || !s.description.trim()))) {
    throw new Error('Điền đầy đủ tên và mô tả bước chế tác (tối đa 30 bước)')
  }
  for (const [field, part] of Object.entries({ story: data.story, fabric: data.fabric, craft: data.craft })) {
    if (part?.image && (!part.image.src.trim() || !part.image.alt.trim())) {
      throw new Error(`Ảnh ${field} phải có cả URL và mô tả`)
    }
  }
}
