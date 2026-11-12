export const sidebarFilterKeys = ['search', 'material', 'color', 'size', 'min_price', 'max_price'] as const
export type SidebarFilters = Record<(typeof sidebarFilterKeys)[number], string>
