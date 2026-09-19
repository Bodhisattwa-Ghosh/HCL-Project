type ToastProps = {
  message: string
  tone?: 'error' | 'success'
}

export function Toast({ message, tone = 'error' }: ToastProps) {
  return <div className={`toast toast-${tone}`} role={tone === 'error' ? 'alert' : 'status'}>{message}</div>
}
