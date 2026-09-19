import { navigate } from '../utils'

type BrandProps = { compact?: boolean }

export function Brand({ compact = false }: BrandProps) {
  return (
    <button className="brand" onClick={() => navigate('/')} aria-label="PulsePoll home">
      <span className="brand-mark" aria-hidden="true"><i /><i /><i /></span>
      {!compact && <span>Pulse<span>Poll</span></span>}
    </button>
  )
}

export function LivePill({ label = 'Live' }: { label?: string }) {
  return <span className="live-pill"><span />{label}</span>
}
