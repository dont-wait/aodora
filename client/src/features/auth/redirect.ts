/** Where to land after login: an explicit `next` wins; otherwise admins open the admin dashboard and customers the shop. */
export function landingPath(role: string | undefined, next: string | null): string {
  const target = safeNextPath(next)
  if (target !== '/') return target
  return role === 'admin' ? '/admin' : '/'
}

export function safeNextPath(value: string | null): string {
  if (!value || !value.startsWith('/') || value.startsWith('//') || value.includes('\\')) return '/'
  return value
}
