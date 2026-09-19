export function formatCount(value: number): string {
  return new Intl.NumberFormat('en-US').format(value)
}

export function formatDate(value: string): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
  }).format(new Date(value))
}

export function shortDate(value: string | null): string {
  if (!value) return 'No closing time'
  return `Closes ${formatDate(value)}`
}

export function pollUrl(slug: string): string {
  return `${window.location.origin}/p/${slug}`
}

export function navigate(path: string): void {
  window.history.pushState({}, '', path)
  window.dispatchEvent(new PopStateEvent('popstate'))
}
