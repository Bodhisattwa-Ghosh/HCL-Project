import { useState, type FormEvent } from 'react'
import { ApiError, api } from '../api'
import type { Poll } from '../types'
import { Toast } from './Toast'

type CreatePollFormProps = {
  token: string
  onCreated: (poll: Poll) => void
  onCancel: () => void
}

export function CreatePollForm({ token, onCreated, onCancel }: CreatePollFormProps) {
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [closesAt, setClosesAt] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  function updateOption(index: number, value: string) {
    setOptions(options.map((item, itemIndex) => itemIndex === index ? value : item))
  }

  function removeOption(index: number) {
    if (options.length > 2) setOptions(options.filter((_, itemIndex) => itemIndex !== index))
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setBusy(true)
    try {
      const payload = {
        question,
        options: options.map((option) => option.trim()),
        ...(closesAt ? { closesAt: new Date(closesAt).toISOString() } : {}),
      }
      const { poll } = await api.createPoll(token, payload)
      onCreated(poll)
    } catch (caught) {
      setError(caught instanceof ApiError ? caught.message : 'Could not create the poll. Try again shortly.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="create-panel" aria-labelledby="create-poll-heading">
      <div className="section-heading">
        <div>
          <div className="eyebrow">New live poll</div>
          <h2 id="create-poll-heading">What should the room decide?</h2>
        </div>
        <button className="icon-button" type="button" onClick={onCancel} title="Close poll composer" aria-label="Close poll composer">×</button>
      </div>
      <form onSubmit={submit} className="poll-form" noValidate>
        <label>
          Question
          <textarea value={question} onChange={(event) => setQuestion(event.target.value)} maxLength={280} placeholder="Which focus area should lead our next sprint?" required />
          <small>{question.length}/280</small>
        </label>
        <fieldset>
          <legend>Options <span>Choose two to eight</span></legend>
          {options.map((option, index) => (
            <div className="option-field" key={index}>
              <span>{String.fromCharCode(65 + index)}</span>
              <input value={option} onChange={(event) => updateOption(index, event.target.value)} maxLength={100} placeholder={`Option ${index + 1}`} required />
              {options.length > 2 && <button type="button" className="icon-button muted" onClick={() => removeOption(index)} aria-label={`Remove option ${index + 1}`}>×</button>}
            </div>
          ))}
          {options.length < 8 && <button className="add-option" type="button" onClick={() => setOptions([...options, ''])}>+ Add another option</button>}
        </fieldset>
        <label className="close-time">
          Close automatically <span>Optional</span>
          <input type="datetime-local" value={closesAt} onChange={(event) => setClosesAt(event.target.value)} min={localMinimumTime()} />
        </label>
        {error && <Toast message={error} />}
        <div className="form-actions">
          <button className="secondary-button" type="button" onClick={onCancel}>Cancel</button>
          <button className="primary-button" type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create live poll'} {!busy && <span aria-hidden="true">→</span>}</button>
        </div>
      </form>
    </section>
  )
}

function localMinimumTime(): string {
  const date = new Date(Date.now() + 60_000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}
