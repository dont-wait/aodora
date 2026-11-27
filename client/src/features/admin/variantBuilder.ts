import { buildVariantDrafts, parseColorNames, type VariantDraft, type VariantEdits } from './catalog.ts'

export interface BuilderState { sizes: string[]; colorText: string; edits: VariantEdits }
export const emptyBuilder: BuilderState = { sizes: [], colorText: '', edits: {} }
/** Rows for the current builder state (the page also uses this when submitting). */
export const builderRows = (slug: string, state: BuilderState): VariantDraft[] => buildVariantDrafts(slug, parseColorNames(state.colorText), state.sizes, state.edits)
