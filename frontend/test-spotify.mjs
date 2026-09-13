// Unit test frontend: busca canciones con tu token premium via Go proxy
// Uso: SPOTIFY_TOKEN="BQ..." node test-spotify.mjs
// O sin token: copia el token de DevTools > Application > Local Storage > spotify_token tras logearte en http://localhost:5173

const token = process.env.SPOTIFY_TOKEN || process.argv[2]
if (!token) {
  console.log('SPOTIFY_TOKEN no seteado')
  console.log('1. pnpm run dev (ya está en http://localhost:5173)')
  console.log('2. Logeate con Spotify premium')
  console.log('3. DevTools > Application > Local Storage > http://localhost:5173 > spotify_token > copia valor')
  console.log('4. SPOTIFY_TOKEN="BQ..." node test-spotify.mjs  ó  node test-spotify.mjs "BQ..."')
  process.exit(0)
}

const base = process.env.GO_PROXY || 'http://127.0.0.1:8081'

async function testSearch(q) {
  console.log(`\nBuscando "${q}"...`)
  const res = await fetch(`${base}/api/v1/spotify/proxy/search?q=${encodeURIComponent(q)}&type=track&limit=5`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  console.log(`Status: ${res.status} ${res.headers.get('X-Source') || ''} ${res.headers.get('X-Cache') || ''}`)
  if (res.status === 429) {
    console.log(`429 rate limit - Retry-After: ${res.headers.get('Retry-After')}s (en Docker Linux con spclient no pasa)`)
    const body = await res.text()
    console.log(body.slice(0, 200))
    return
  }
  if (!res.ok) {
    console.log(`Error ${res.status}:`, await res.text().then(t => t.slice(0, 500)))
    return
  }
  const data = await res.json()
  const items = data.tracks?.items || []
  console.log(`OK: ${items.length} tracks`)
  items.slice(0, 3).forEach((t, i) => {
    const artists = t.artists?.map(a => a.name).join(', ') || t.artist || '?'
    console.log(`  [${i}] ${t.name} - ${artists} (${t.id})`)
  })
  if (items.length === 0) console.log('Sin resultados - revisa query o token')
}

await testSearch('pierce the veil')
await testSearch('muse hysteria')
await testSearch('queen bohemian')
console.log('\nDone. Si ves tracks, la búsqueda funciona. Si 429, espera 30s o usa Docker Linux para spclient.')
