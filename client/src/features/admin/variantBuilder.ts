import { buildVariantDrafts, type VariantDraft, type VariantEdits } from './catalog.ts'

/** colours: names picked from the product's colour list (never typed freely). */
export interface BuilderState { sizes: string[]; colors: string[]; edits: VariantEdits }
export const emptyBuilder: BuilderState = { sizes: [], colors: [], edits: {} }

/** Rows for the current builder state; colours no longer in `available` are dropped. */
export const builderRows = (slug: string, state: BuilderState, available?: string[]): VariantDraft[] => {
  const colors = available ? state.colors.filter((c) => available.some((a) => a.trim().toLowerCase() === c.trim().toLowerCase())) : state.colors
  return buildVariantDrafts(slug, colors, state.sizes, state.edits)
}
