'use client'

import hljs from 'highlight.js'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { useEffect, useMemo, useState } from 'react'

type SnippetFile = {
  id: string
  snippet_id: string
  path: string
  content: string
  language: string
}

type Snippet = {
  id: string
  files: SnippetFile[]
  created_at: string
  updated_at: string
}

const API_BASE = process.env.NEXT_PUBLIC_CLANKER_API_URL ?? 'http://localhost:8080'

export default function SnippetPage() {
  const params = useParams<{ id: string }>()
  const [snippet, setSnippet] = useState<Snippet | null>(null)
  const [selectedFileID, setSelectedFileID] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    async function load() {
      setLoading(true); setError('')
      try {
        const res = await fetch(`${API_BASE}/snippet/${encodeURIComponent(params.id)}`)
        const body = await res.json().catch(() => null)
        if (!res.ok) throw new Error(body?.error ?? `Request failed: ${res.status}`)
        setSnippet(body)
        setSelectedFileID(body.files?.[0]?.id ?? '')
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load snippet')
      } finally { setLoading(false) }
    }
    load()
  }, [params.id])

  const selected = snippet?.files?.find((file) => file.id === selectedFileID) ?? snippet?.files?.[0] ?? null

  return (
    <main className="page">
      <section className="hero">
        <h1>Clanker Snippet</h1>
        <p><Link href="/">Open another snippet</Link></p>
      </section>
      <section className="panel">
        {loading && <div className="empty">Loading snippet…</div>}
        {error && <div className="error">{error}</div>}
        {snippet && (
          <div className="grid">
            <div className="list">
              <div className="meta">{snippet.id}</div>
              {snippet.files?.map((file) => (
                <button key={file.id} className={`card ${selected?.id === file.id ? 'active' : ''}`} onClick={() => setSelectedFileID(file.id)}>
                  <strong>{file.path || 'snippet'}</strong>
                  <div className="meta">{file.language || 'plaintext'}</div>
                </button>
              ))}
            </div>
            <FileViewer file={selected} />
          </div>
        )}
      </section>
    </main>
  )
}

function FileViewer({ file }: { file: SnippetFile | null }) {
  const highlighted = useMemo(() => {
    if (!file) return ''
    const language = hljs.getLanguage(file.language) ? file.language : 'plaintext'
    return hljs.highlight(file.content, { language }).value
  }, [file])

  if (!file) return <div className="viewer empty">No files in this snippet.</div>

  return (
    <div className="viewer">
      <div className="viewer-head">
        <div><code>{file.path || 'snippet'}</code><div className="meta">{file.language || 'plaintext'}</div></div>
        <button onClick={() => navigator.clipboard.writeText(file.content)}>Copy</button>
      </div>
      <div className="code-wrap">
        <pre><code className={`language-${file.language}`} dangerouslySetInnerHTML={{ __html: highlighted }} /></pre>
      </div>
    </div>
  )
}
