// Small helpers for calling the Go API. In development Vite proxies /api to :8080.

async function request(method, path, body) {
  const res = await fetch(path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.error || `Request failed (${res.status})`)
  }
  return data
}

export const api = {
  summary: () => request('GET', '/api/summary'),
  supplierUpdates: () => request('GET', '/api/supplier/updates'),

  runs: () => request('GET', '/api/runs'),
  runChanges: (id) => request('GET', `/api/runs/${id}/changes`),
  startRun: (updateId) => request('POST', '/api/runs', { update_id: updateId }),
  resumeRun: (id) => request('POST', `/api/runs/${id}/resume`),
  abandonRun: (id) => request('POST', `/api/runs/${id}/abandon`),

  products: () => request('GET', '/api/products'),
  problems: () => request('GET', '/api/problems'),
  resolveProblem: (id) => request('POST', `/api/problems/${id}/resolve`),

  getOutage: () => request('GET', '/api/demo/outage'),
  setOutage: (enabled) => request('POST', '/api/demo/outage', { enabled }),
}

export function formatPrice(cents, currency) {
  if (!currency) return '—'
  return `${(cents / 100).toFixed(2)} ${currency}`
}

export function formatDate(value) {
  if (!value || value.startsWith('0001')) return '—'
  return new Date(value).toLocaleString()
}
