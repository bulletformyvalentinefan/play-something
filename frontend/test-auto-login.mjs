// Test auto-login: pide login a Spotify y saca tu token solo para probar búsqueda
// Uso: node test-auto-login.mjs
// Te abrirá el navegador a Spotify, logeate con premium, y el test tomará tu token solo
import { exec } from 'child_process'
import { setTimeout as sleep } from 'timers/promises'

const GO = 'http://127.0.0.1:8081'

async function getTokenByLogin() {
  if (process.env.SPOTIFY_TOKEN) return process.env.SPOTIFY_TOKEN

  // si ya hay token en Go (por pnpm dev login previo), úsalo
  try {
    const r = await fetch(`${GO}/api/v1/spotify/auth/token`)
    if (r.ok) {
      const j = await r.json()
      if (j.access_token) {
        console.log('Usando token existente de Go memory')
        return j.access_token
      }
    }
  } catch {}

  console.log('Pidiendo login a Spotify...')
  const res = await fetch(`${GO}/api/v1/spotify/auth/start`, { method: 'POST', headers: { 'Content-Type': 'application/json' } })
  const data = await res.json()
  if (!data.url) throw new Error('No se pudo obtener URL de login: ' + JSON.stringify(data))
  console.log('Abriendo navegador...')
  console.log(data.url)
  const openCmd = process.platform === 'win32' ? `start "" "${data.url}"` : process.platform === 'darwin' ? `open "${data.url}"` : `xdg-open "${data.url}"`
  exec(openCmd, () => {})

  console.log('Esperando que completes login (120s)... Ve a Spotify y autoriza')
  const deadline = Date.now() + 120000
  while (Date.now() < deadline) {
    await sleep(2000)
    try {
      const r = await fetch(`${GO}/api/v1/spotify/auth/token`)
      if (r.ok) {
        const j = await r.json()
        if (j.access_token) {
          console.log('¡Login detectado! Token obtenido')
          return j.access_token
        }
      }
    } catch {}
  }
  throw new Error('Timeout 120s esperando login')
}

async function testSearch(token, q) {
  console.log(`\nBuscando "${q}"...`)
  const res = await fetch(`${GO}/api/v1/spotify/proxy/search?q=${encodeURIComponent(q)}&type=track&limit=5`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  console.log(`Status: ${res.status} ${res.headers.get('X-Source') || ''} ${res.headers.get('X-Cache') || ''}`)
  if (!res.ok) {
    console.log(`Error ${res.status}:`, (await res.text()).slice(0, 500))
    return
  }
  const data = await res.json()
  const items = data.tracks?.items || []
  console.log(`OK: ${items.length} tracks`)
  items.slice(0, 3).forEach((t, i) => console.log(`  [${i}] ${t.name} - ${t.artists?.[0]?.name} (${t.id})`))
  if (items.length === 0) console.log('Sin resultados')
}

const token = await getTokenByLogin()
console.log(`\nToken: ${token.slice(0, 20)}...`)

await testSearch(token, 'pierce the veil')
await testSearch(token, 'muse hysteria')
await testSearch(token, 'queen bohemian')
console.log('\n✓ Tests con tu token premium completados')
