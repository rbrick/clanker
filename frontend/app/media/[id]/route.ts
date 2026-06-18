import { NextRequest } from 'next/server'

const API_BASE = process.env.CLANKER_API_URL ?? process.env.NEXT_PUBLIC_CLANKER_API_URL ?? 'http://localhost:8080'

export async function GET(_request: NextRequest, { params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  const upstream = await fetch(`${API_BASE}/media/${encodeURIComponent(id)}`, { cache: 'no-store' })

  if (!upstream.ok) {
    const body = await upstream.text().catch(() => '')
    return new Response(body || 'media not found', {
      status: upstream.status,
      headers: { 'content-type': upstream.headers.get('content-type') ?? 'text/plain' },
    })
  }

  return new Response(upstream.body, {
    status: upstream.status,
    headers: {
      'content-type': upstream.headers.get('content-type') ?? 'application/octet-stream',
      'cache-control': 'public, max-age=31536000, immutable',
    },
  })
}
