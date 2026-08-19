const BASE = '/api/v1/spotify'

async function request(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, options)

  if (!res.ok) {
    let message = `Error ${res.status}`
    try {
      const body = await res.json()
      if (body?.message) message = body.message
    } catch {
      /* cuerpo no JSON */
    }
    throw new Error(message)
  }

  const text = await res.text()
  if (!text) return null

  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

export const api = {
  get: (path) => request(path),
  post: (path, body) =>
    request(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    }),
  del: (path) => request(path, { method: 'DELETE' }),
}