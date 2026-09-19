import { useState } from 'react'
import { pollUrl } from '../utils'

export function ShareLink({ slug, subtle = false }: { slug: string; subtle?: boolean }) {
  const [copied, setCopied] = useState(false)
  const url = pollUrl(slug)

  async function copy() {
    try {
      await navigator.clipboard.writeText(url)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1800)
    } catch {
      window.prompt('Copy this poll link', url)
    }
  }

  return (
    <div className={`share-link ${subtle ? 'share-link-subtle' : ''}`}>
      {!subtle && <span className="share-label">Share this poll</span>}
      <div className="share-controls">
        {!subtle && <code title={url}>{url}</code>}
        <button className={subtle ? 'icon-button' : 'copy-button'} onClick={copy} title="Copy poll link">
          <span aria-hidden="true">{copied ? '✓' : '⧉'}</span> {subtle ? null : copied ? 'Copied' : 'Copy link'}
        </button>
      </div>
    </div>
  )
}
