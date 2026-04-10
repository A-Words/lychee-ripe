export type AuthMode = 'disabled' | 'oidc'
export type UserRole = 'admin' | 'operator'
export type UserStatus = 'active' | 'disabled'

export interface Principal {
  subject: string
  email: string
  display_name: string
  role: UserRole
  status: UserStatus
  auth_mode: AuthMode
  permissions: string[]
}

export interface AuthSession {
  accessToken: string
  idToken?: string
  expiresAt?: number
}

export interface OIDCDiscoveryDocument {
  issuer: string
  authorization_endpoint: string
  token_endpoint: string
  end_session_endpoint?: string
}
