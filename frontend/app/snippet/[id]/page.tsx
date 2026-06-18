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
  git_url?: string
  created_at: string
  updated_at: string
}

type FileTreeNode = {
  name: string
  path: string
  children: Map<string, FileTreeNode>
  file?: SnippetFile
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
  const tree = useMemo(() => buildFileTree(snippet?.files ?? []), [snippet?.files])
  const gitURL = useMemo(() => {
    if (!snippet?.git_url) return ''
    if (typeof window === 'undefined') return snippet.git_url
    try {
      const url = new URL(snippet.git_url)
      if (url.hostname === 'localhost' || url.hostname === '127.0.0.1') {
        return `${window.location.origin}${url.pathname}`
      }
    } catch {
      return snippet.git_url
    }
    return snippet.git_url
  }, [snippet?.git_url])

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
              {gitURL && (
                <div className="meta">
                  <code>git clone {gitURL}</code>
                </div>
              )}
              <FileTree nodes={tree} selectedFileID={selected?.id ?? ''} onSelect={setSelectedFileID} />
            </div>
            <FileViewer file={selected} />
          </div>
        )}
      </section>
    </main>
  )
}

function buildFileTree(files: SnippetFile[]) {
  const root = new Map<string, FileTreeNode>()
  for (const file of files) {
    const displayPath = file.path || 'snippet'
    const parts = displayPath.split('/').filter(Boolean)
    let level = root
    let currentPath = ''
    parts.forEach((part, index) => {
      currentPath = currentPath ? `${currentPath}/${part}` : part
      let node = level.get(part)
      if (!node) {
        node = { name: part, path: currentPath, children: new Map() }
        level.set(part, node)
      }
      if (index === parts.length - 1) node.file = file
      level = node.children
    })
  }
  return sortNodes(root)
}

function sortNodes(nodes: Map<string, FileTreeNode>): FileTreeNode[] {
  return [...nodes.values()]
    .map((node) => ({ ...node, children: new Map(sortNodes(node.children).map((child) => [child.name, child])) }))
    .sort((a, b) => {
      if (!a.file && b.file) return -1
      if (a.file && !b.file) return 1
      return a.name.localeCompare(b.name)
    })
}

function FileTree({ nodes, selectedFileID, onSelect }: { nodes: FileTreeNode[], selectedFileID: string, onSelect: (id: string) => void }) {
  if (nodes.length === 0) return <div className="empty">No files in this snippet.</div>
  return <div className="file-tree">{nodes.map((node) => <FileTreeItem key={node.path} node={node} selectedFileID={selectedFileID} onSelect={onSelect} depth={0} />)}</div>
}

function FileTreeItem({ node, selectedFileID, onSelect, depth }: { node: FileTreeNode, selectedFileID: string, onSelect: (id: string) => void, depth: number }) {
  const children = [...node.children.values()]
  const isFile = Boolean(node.file)
  if (isFile) {
    return (
      <button className={`tree-row file ${selectedFileID === node.file?.id ? 'active' : ''}`} style={{ paddingLeft: 12 + depth * 16 }} onClick={() => node.file && onSelect(node.file.id)}>
        <span className="tree-icon">📄</span>
        <span className="tree-name">{node.name}</span>
        {node.file?.language && <span className="tree-lang">{node.file.language}</span>}
      </button>
    )
  }
  return (
    <div className="tree-dir">
      <div className="tree-row dir" style={{ paddingLeft: 12 + depth * 16 }}>
        <span className="tree-icon">📁</span>
        <span className="tree-name">{node.name}</span>
      </div>
      {children.map((child) => <FileTreeItem key={child.path} node={child} selectedFileID={selectedFileID} onSelect={onSelect} depth={depth + 1} />)}
    </div>
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
