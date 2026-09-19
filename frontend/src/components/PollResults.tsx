import type { Poll } from '../types'
import { formatCount } from '../utils'

type ResultsProps = {
  poll: Poll
  selectedId?: string
  onSelect?: (id: string) => void
  disabled?: boolean
  showSelection?: boolean
}

const hues = ['violet', 'amber', 'teal', 'coral', 'blue', 'pink', 'lime', 'slate']

export function PollResults({ poll, selectedId, onSelect, disabled = false, showSelection = true }: ResultsProps) {
  const hasVotes = poll.totalVotes > 0
  return (
    <div className="results-list">
      {poll.options.map((option, index) => {
        const percentage = hasVotes ? Math.round((option.votes / poll.totalVotes) * 100) : 0
        const selected = selectedId === option.id
        const content = (
          <>
            <span className="option-topline">
              <span className="option-label"><span className="option-radio" aria-hidden="true" />{option.label}</span>
              <span className="option-stat">{percentage}% <em>{formatCount(option.votes)}</em></span>
            </span>
            <span className="result-track" aria-hidden="true"><span className={`result-fill fill-${hues[index % hues.length]}`} style={{ width: `${percentage}%` }} /></span>
          </>
        )
        if (!showSelection || !onSelect) {
          return <div key={option.id} className="result-option result-static">{content}</div>
        }
        return (
          <button
            key={option.id}
            className={`result-option ${selected ? 'selected' : ''}`}
            onClick={() => onSelect(option.id)}
            disabled={disabled}
            type="button"
            aria-pressed={selected}
          >
            {content}
          </button>
        )
      })}
    </div>
  )
}
