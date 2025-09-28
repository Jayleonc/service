import { api } from '@/api/client'

export function getUserRoles(userId) {
  return api.get(`/v1/roles/${userId}`)
}

export function setUserRoles(userId, roles) {
  return api.put(`/v1/roles/${userId}`, { roles })
}

export function addUserRole(userId, role) {
  return api.post('/v1/roles', { user_id: userId, role })
}
