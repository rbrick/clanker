import { NextRequest } from 'next/server'

const API_BASE = process.env.CLANKER_API_URL ?? process.env.NEXT_PUBLIC_CLANKER_API_URL ?? 'http://localhost:8080'

export async function GET(_request: NextRequest, { params }: { params: Promise<{ repo: string, path?: string[] }> }) {
  const { repo, path = [] } = await params
  const suffix = path.length > 0 ? `/${path.map(encodeURIComponent).join('/')}` : ''
  const upstream = await fetch(`${API_BASE}/git/${encodeURIComponent(repo)}${suffix}`, { cache: 'no-store' })

  if (!upstream.ok) {
    return new Response(await upstream.text().catch(() => 'git object not found'), { status: upstream.status })
  }

  return new Response(upstream.body, {
    status: upstream.status,
    headers: {
      'content-type': upstream.headers.get('content-type') ?? 'application/octet-stream',
      'cache-control': 'no-store',
    },
  })
}
