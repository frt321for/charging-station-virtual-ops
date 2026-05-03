import { Activity, RefreshCw } from 'lucide-react'
import { useHealth } from '../../hooks/useHealth'

const dependencyLabels: Record<string, string> = {
  database: 'PostgreSQL / TimescaleDB',
  cache: 'Valkey',
  events: 'NATS JetStream',
}

export function DevHealthPage() {
  const { data, error, isFetching, refetch } = useHealth()
  const dependencyEntries = Object.keys(dependencyLabels).map((key) => ({
    key,
    value: data?.dependencies[key] ?? 'loading',
  }))
  const status = data?.status ?? 'loading'

  return (
    <main className="min-h-dvh bg-[var(--surface-page)] text-[var(--text-primary)]">
      <section className="mx-auto flex min-h-dvh w-full max-w-7xl flex-col px-5 py-6 md:px-8">
        <header className="flex flex-col gap-4 border-b border-[var(--border-subtle)] pb-5 md:flex-row md:items-center md:justify-between">
          <h1 className="text-2xl font-semibold leading-tight tracking-normal text-[var(--text-primary)] md:whitespace-nowrap md:text-3xl">
            园区充电桩虚拟运营中台
          </h1>

          <button
            type="button"
            onClick={() => void refetch()}
            className="inline-flex min-h-11 items-center gap-2 border border-[var(--border-strong)] px-3 text-sm font-medium text-[var(--text-primary)] transition hover:bg-[var(--surface-control-hover)] active:translate-y-px disabled:cursor-not-allowed disabled:opacity-50"
            disabled={isFetching}
          >
            <RefreshCw size={16} className={isFetching ? 'animate-spin' : ''} aria-hidden="true" />
            刷新
          </button>
        </header>

        <div className="flex flex-1 py-8">
          <section className="w-full max-w-xl border border-[var(--border-strong)] bg-[var(--surface-panel)] p-5 shadow-sm">
            <div className="mb-5 flex items-center justify-between gap-4">
              <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
                <Activity size={17} aria-hidden="true" />
                联通状态
              </div>
            </div>

            {error ? (
              <div
                role="alert"
                className="border border-[var(--danger-border)] bg-[var(--danger-bg)] p-4 text-sm text-[var(--danger-text)]"
              >
                联通失败
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex items-center justify-between border border-[var(--border-subtle)] px-3 py-3">
                  <span className="text-sm text-[var(--text-secondary)]">API</span>
                  <span className="text-sm font-semibold text-[var(--status-ok)]">{status}</span>
                </div>

                {dependencyEntries.map(({ key, value }) => {
                  const statusClass =
                    value === 'ok'
                      ? 'text-sm font-semibold text-[var(--status-ok)]'
                      : 'text-sm font-semibold text-[var(--text-muted)]'

                  return (
                    <div
                      key={key}
                      className="flex items-center justify-between border border-[var(--border-subtle)] px-3 py-3"
                    >
                      <span className="text-sm text-[var(--text-secondary)]">
                        {dependencyLabels[key] ?? key}
                      </span>
                      <span className={statusClass}>{value}</span>
                    </div>
                  )
                })}
              </div>
            )}
          </section>
        </div>
      </section>
    </main>
  )
}
