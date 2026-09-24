// Test manual: abre pestaña Spotify premium y guarda token
// Uso: cd frontend && pnpm test:login  (o node scripts/test-login.mjs)
// GO_URL: backend a usar (default http://127.0.0.1:8081, compose: :3210)
import { exec } from 'child_process'
import { setTimeout as sleep } from 'timers/promises'
import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const GO = process.env.GO_URL || 'http://127.0.0.1:8081'
const OUT = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', '.spotify-token')

async function getToken() {
  console.log('Pidiendo URL de login a Go...')
  const res = await fetch(`${GO}/api/v1/spotify/auth/start`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' })
  const data = await res.json()
  if (!data.url) throw new Error('No URL: ' + JSON.stringify(data))
  console.log('Abriendo navegador...')
  console.log(data.url)
  const cmd = process.platform === 'win32' ? `rundll32 url.dll,FileProtocolHandler "${data.url}"` : process.platform === 'darwin' ? `open "${data.url}"` : `xdg-open "${data.url}"`
  exec(cmd, () => {})
  console.log('Logeate con premium y autoriza (120s)...')
  const deadline = Date.now() + 120000
  while (Date.now() < deadline) {
    await sleep(2000)
    try {
      const r = await fetch(`${GO}/api/v1/spotify/auth/token`)
      if (r.ok) {
        const j = await r.json()
        if (j.access_token) return j.access_token
      }
    } catch {}
  }
  throw new Error('Timeout esperando login')
}

const token = await getToken()
console.log(`\nToken: ${token.slice(0, 30)}... (len ${token.length})`)
fs.writeFileSync(OUT, token)
console.log('Guardado en frontend/.spotify-token (ignorado por git, solo tests)')
console.log('Listo: tu token premium está guardado SOLO para tests')
