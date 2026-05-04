import { useQuery } from '@tanstack/react-query'
import { listRoles, listUsers } from '../services/auth-api'

export function useAccessAdmin(enabled: boolean) {
  const usersQuery = useQuery({
    queryKey: ['access-admin', 'users'],
    queryFn: listUsers,
    enabled,
  })
  const rolesQuery = useQuery({
    queryKey: ['access-admin', 'roles'],
    queryFn: listRoles,
    enabled,
  })

  return { usersQuery, rolesQuery }
}
