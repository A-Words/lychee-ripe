import type { UserRole, UserStatus } from '~/types/auth'

export type ResourceStatus = 'active' | 'archived'

export interface Orchard {
  orchard_id: string
  orchard_name: string
  status: ResourceStatus
  created_at: string
  updated_at: string
}

export interface Plot {
  plot_id: string
  orchard_id: string
  plot_name: string
  status: ResourceStatus
  created_at: string
  updated_at: string
}

export interface UserRecord {
  id: string
  email: string
  display_name: string
  oidc_subject?: string | null
  role: UserRole
  status: UserStatus
  last_login_at?: string | null
  created_at: string
  updated_at: string
}

export interface ListResponse<T> {
  items: T[]
}

export interface PlotOption {
  plot_id: string
  plot_name: string
}

export interface OrchardWithPlots {
  orchard_id: string
  orchard_name: string
  plots: PlotOption[]
}
