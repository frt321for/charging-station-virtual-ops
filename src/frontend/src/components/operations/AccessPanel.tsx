import { ShieldCheck } from 'lucide-react'
import { useAccessAdmin } from '../../hooks/useAccessAdmin'
import { StatusBadge } from './StatusBadge'

interface AccessPanelProps {
  enabled: boolean
}

export function AccessPanel({ enabled }: AccessPanelProps) {
  const { usersQuery, rolesQuery } = useAccessAdmin(enabled)
  const users = usersQuery.data ?? []
  const roles = rolesQuery.data ?? []

  return (
    <section className="ops-panel">
      <header>
        <h2>用户角色</h2>
        <StatusBadge tone={usersQuery.isFetching || rolesQuery.isFetching ? 'charge' : 'ok'}>
          {users.length.toString()}
        </StatusBadge>
      </header>
      <div className="ops-access-grid">
        <div className="ops-table-wrap">
          <table>
            <thead>
              <tr>
                <th>用户</th>
                <th>角色</th>
                <th>站点</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr key={user.id}>
                  <td>
                    <strong>{user.displayName}</strong>
                    <span className="ops-cell-sub">{user.username}</span>
                  </td>
                  <td>{user.roles.map((role) => role.name).join(' / ')}</td>
                  <td>{user.sites.length > 0 ? user.sites.map((site) => site.code).join(' / ') : 'ALL'}</td>
                  <td>
                    <StatusBadge tone={user.status === 'active' ? 'ok' : 'idle'}>{user.status}</StatusBadge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {users.length === 0 ? <div className="ops-empty">暂无用户</div> : null}
        </div>

        <div className="ops-role-list">
          {roles.map((role) => (
            <article key={role.id}>
              <div>
                <ShieldCheck size={16} aria-hidden="true" />
                <strong>{role.name}</strong>
                <StatusBadge tone={role.status === 'active' ? 'ok' : 'idle'}>{role.code}</StatusBadge>
              </div>
              <p>{role.permissions.join(' / ')}</p>
            </article>
          ))}
          {roles.length === 0 ? <div className="ops-empty">暂无角色</div> : null}
        </div>
      </div>
    </section>
  )
}
