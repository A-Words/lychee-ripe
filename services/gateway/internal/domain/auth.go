package domain

import "time"

type AuthMode string

const (
	AuthModeDisabled AuthMode = "disabled"
	AuthModeOIDC     AuthMode = "oidc"
)

type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleOperator UserRole = "operator"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type ResourceStatus string

const (
	ResourceStatusActive   ResourceStatus = "active"
	ResourceStatusArchived ResourceStatus = "archived"
)

type IdentityClaims struct {
	Subject     string
	Email       string
	DisplayName string
	ExpiresAt   *time.Time
}

type Principal struct {
	Subject     string
	Email       string
	DisplayName string
	Role        UserRole
	Status      UserStatus
	AuthMode    AuthMode
}

type UserRecord struct {
	ID          string
	Email       string
	DisplayName string
	OIDCSubject *string
	Role        UserRole
	Status      UserStatus
	LastLoginAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WebSessionRecord struct {
	SessionIDHash string
	UserID        string
	IDToken       *string
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type WebAuthStateRecord struct {
	State              string
	BrowserBindingHash string
	CodeVerifier       string
	RedirectPath       string
	ExpiresAt          time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
