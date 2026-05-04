import { useState, type FormEvent } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { LockKeyhole, LogIn } from 'lucide-react'
import { useAuth } from '../../context/useAuth'

export function LoginPage() {
  const auth = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [localError, setLocalError] = useState<string | undefined>()
  const isPending = auth.status === 'loading'

  if (auth.status === 'authenticated') {
    return <Navigate to="/operations" replace />
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLocalError(undefined)
    try {
      await auth.login(username, password)
      navigate('/operations', { replace: true })
    } catch {
      setLocalError('账号或密码错误')
    }
  }

  return (
    <main className="login-shell">
      <section className="login-panel" aria-label="登录">
        <div className="login-brand">
          <span>CHARGE OPS</span>
          <h1>充电运营中台</h1>
        </div>
        <form className="login-form" onSubmit={submit}>
          <label>
            账号
            <input
              name="username"
              autoComplete="username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
            />
          </label>
          <label>
            密码
            <input
              name="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>
          <button type="submit" disabled={isPending || username.trim() === '' || password === ''}>
            {isPending ? <LockKeyhole size={16} aria-hidden="true" /> : <LogIn size={16} aria-hidden="true" />}
            登录
          </button>
          {localError || auth.error ? (
            <output className="login-error" role="alert">
              {localError ?? auth.error}
            </output>
          ) : null}
        </form>
      </section>
    </main>
  )
}
