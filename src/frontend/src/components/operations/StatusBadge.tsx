interface StatusBadgeProps {
  children: string
  tone?: 'ok' | 'charge' | 'warn' | 'danger' | 'idle'
}

export function StatusBadge({ children, tone = 'idle' }: StatusBadgeProps) {
  return <span className={`ops-status ops-status--${tone}`}>{children}</span>
}
