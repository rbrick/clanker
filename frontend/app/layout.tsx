import './globals.css'
import 'highlight.js/styles/github-dark.css'
import type { Metadata } from 'next'

export const metadata: Metadata = {
  title: 'Clanker Snippets',
  description: 'Browse Clanker code snippets',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
