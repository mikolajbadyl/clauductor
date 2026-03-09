export function backendUrl(path: string): string {
  if (import.meta.dev) {
    const host = window.location.hostname || 'localhost'
    return `http://${host}:8080${path}`
  }
  return path
}

export function wsUrl(path: string): string {
  if (import.meta.dev) {
    const host = window.location.hostname || 'localhost'
    return `ws://${host}:8080${path}`
  }
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}${path}`
}
