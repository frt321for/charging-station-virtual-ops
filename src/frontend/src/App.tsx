import { RefreshCw, Server, ShieldCheck } from 'lucide-react'
import { useHealth } from './hooks/useHealth'

const dependencyLabels: Record<string, string> = {
  database: 'PostgreSQL / TimescaleDB',
  cache: 'Valkey',
  events: 'NATS JetStream',
}

export default function App() {
  const { data, error, isFetching, refetch } = useHealth()
  const dependencyEntries = Object.keys(dependencyLabels).map((key) => ({
    key,
    value: data?.dependencies[key] ?? 'loading',
  }))

  return (
    <main className="min-h-dvh bg-[var(--surface-page)] text-[var(--text-primary)]">
      <section className="mx-auto flex min-h-dvh w-full max-w-6xl flex-col justify-center px-6 py-10">
        <div className="grid gap-8 lg:grid-cols-[1.15fr_0.85fr] lg:items-end">
          <div className="max-w-3xl">
            <div className="mb-8 inline-flex items-center gap-2 border border-[var(--border-subtle)] bg-[var(--surface-panel)] px-3 py-2 text-sm font-medium text-[var(--text-secondary)]">
              <ShieldCheck size={16} aria-hidden="true" />
              完整平台开发入口
            </div>

            <h1 className="max-w-2xl text-4xl font-semibold leading-tight tracking-normal text-[var(--text-primary)] md:text-6xl">
              园区充电桩虚拟运营中台
            </h1>

            <p className="mt-6 max-w-2xl text-base leading-7 text-[var(--text-secondary)]">
              当前阶段用于验证本地 React 前端、Go 后端和云端基础设施的连接链路。业务 UI
              会先进入 HTML 原型确认，再落到 React 页面。
            </p>
          </div>

          <div className="border border-[var(--border-strong)] bg-[var(--surface-panel)] p-5 shadow-sm">
            <div className="mb-5 flex items-start justify-between gap-4">
              <div>
                <div className="flex items-center gap-2 text-sm font-semibold text-[var(--text-primary)]">
                  <Server size={17} aria-hidden="true" />
                  后端健康状态
                </div>
                <p className="mt-1 text-sm text-[var(--text-muted)]">
                  通过 Vite 代理请求 /api/v1/health
                </p>
              </div>

              <button
                type="button"
                onClick={() => void refetch()}
                className="inline-flex min-h-11 items-center gap-2 border border-[var(--border-strong)] px-3 text-sm font-medium text-[var(--text-primary)] transition hover:bg-[var(--surface-control-hover)] active:translate-y-px disabled:cursor-not-allowed disabled:opacity-50"
                disabled={isFetching}
              >
                <RefreshCw
                  size={16}
                  className={isFetching ? 'animate-spin' : ''}
                  aria-hidden="true"
                />
                刷新
              </button>
            </div>

            {error ? (
              <div
                role="alert"
                className="border border-[var(--danger-border)] bg-[var(--danger-bg)] p-4 text-sm text-[var(--danger-text)]"
              >
                后端健康检查失败，请确认 `scripts/backend.ps1 start` 已运行。
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex items-center justify-between border border-[var(--border-subtle)] px-3 py-3">
                  <span className="text-sm text-[var(--text-secondary)]">服务状态</span>
                  <span className="text-sm font-semibold text-[var(--status-ok)]">
                    {data?.status ?? 'loading'}
                  </span>
                </div>

                {dependencyEntries.map(({ key, value }) => {
                  const status = value === 'ok' ? 'ok' : 'pending'

                  return (
                    <div
                      key={key}
                      className="flex items-center justify-between border border-[var(--border-subtle)] px-3 py-3"
                    >
                      <span className="text-sm text-[var(--text-secondary)]">
                        {dependencyLabels[key] ?? key}
                      </span>
                      <span
                        className={
                          status === 'ok'
                            ? 'text-sm font-semibold text-[var(--status-ok)]'
                            : 'text-sm font-semibold text-[var(--text-muted)]'
                        }
                      >
                        {value}
                      </span>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      </section>
    </main>
  )
}
