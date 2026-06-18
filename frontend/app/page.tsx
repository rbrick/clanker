'use client'

import { FormEvent, useState } from 'react'
import { useRouter } from 'next/navigation'

export default function Home() {
  const [id, setId] = useState('')
  const router = useRouter()

  function open(event: FormEvent) {
    event.preventDefault()
    if (id.trim()) router.push(`/snippet/${encodeURIComponent(id.trim())}`)
  }

  return (
    <main className="page">
      <section className="hero">
        <h1>Clanker Snippets</h1>
        <p>Paste a snippet UUID to open it. Snippet listing/history is not available.</p>
      </section>
      <section className="panel">
        <form className="controls single" onSubmit={open}>
          <input value={id} onChange={(e) => setId(e.target.value)} placeholder="snippet UUID" />
          <button>Open snippet</button>
        </form>
      </section>
    </main>
  )
}
