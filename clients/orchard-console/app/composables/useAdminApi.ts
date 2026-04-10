import type { Orchard, Plot, UserRecord, ListResponse, ResourceStatus } from '~/types/resources'
import type { UserRole, UserStatus } from '~/types/auth'
import { useAuth } from '~/composables/useAuth'

export function useAdminApi() {
  const auth = useAuth()

  const listOrchards = async (includeArchived = false) =>
    (await auth.gatewayFetch<ListResponse<Orchard>>('/v1/orchards', {
      query: { include_archived: includeArchived }
    })).items

  const createOrchard = async (payload: { orchard_id: string, orchard_name: string, status?: ResourceStatus }) =>
    await auth.gatewayFetch<Orchard>('/v1/orchards', {
      method: 'POST',
      body: payload
    })

  const updateOrchard = async (orchardId: string, payload: { orchard_name: string, status: ResourceStatus }) =>
    await auth.gatewayFetch<Orchard>(`/v1/orchards/${encodeURIComponent(orchardId)}`, {
      method: 'PATCH',
      body: payload
    })

  const archiveOrchard = async (orchardId: string) =>
    await auth.gatewayFetch<Orchard>(`/v1/orchards/${encodeURIComponent(orchardId)}`, {
      method: 'DELETE'
    })

  const listPlots = async (orchardId?: string, includeArchived = false) =>
    (await auth.gatewayFetch<ListResponse<Plot>>('/v1/plots', {
      query: {
        orchard_id: orchardId,
        include_archived: includeArchived
      }
    })).items

  const createPlot = async (payload: { plot_id: string, orchard_id: string, plot_name: string, status?: ResourceStatus }) =>
    await auth.gatewayFetch<Plot>('/v1/plots', {
      method: 'POST',
      body: payload
    })

  const updatePlot = async (plotId: string, payload: Partial<{ orchard_id: string, plot_name: string, status: ResourceStatus }>) =>
    await auth.gatewayFetch<Plot>(`/v1/plots/${encodeURIComponent(plotId)}`, {
      method: 'PATCH',
      body: payload
    })

  const archivePlot = async (plotId: string) =>
    await auth.gatewayFetch<Plot>(`/v1/plots/${encodeURIComponent(plotId)}`, {
      method: 'DELETE'
    })

  const listUsers = async () =>
    (await auth.gatewayFetch<ListResponse<UserRecord>>('/v1/admin/users')).items

  const createUser = async (payload: { email: string, display_name: string, role: UserRole, status: UserStatus }) =>
    await auth.gatewayFetch<UserRecord>('/v1/admin/users', {
      method: 'POST',
      body: payload
    })

  const updateUser = async (userId: string, payload: { email: string, display_name: string, role: UserRole, status: UserStatus }) =>
    await auth.gatewayFetch<UserRecord>(`/v1/admin/users/${encodeURIComponent(userId)}`, {
      method: 'PATCH',
      body: payload
    })

  return {
    listOrchards,
    createOrchard,
    updateOrchard,
    archiveOrchard,
    listPlots,
    createPlot,
    updatePlot,
    archivePlot,
    listUsers,
    createUser,
    updateUser
  }
}
