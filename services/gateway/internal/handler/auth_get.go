package handler

import (
	"net/http"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/middleware"
)

type principalResponse struct {
	Subject     string   `json:"subject"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Role        string   `json:"role"`
	Status      string   `json:"status"`
	AuthMode    string   `json:"auth_mode"`
	Permissions []string `json:"permissions"`
}

func GetCurrentPrincipal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := middleware.GetPrincipal(r.Context())
		if !ok {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "principal missing", nil)
			return
		}
		writeJSON(w, http.StatusOK, toPrincipalResponse(principal))
	}
}

func toPrincipalResponse(principal domain.Principal) principalResponse {
	return principalResponse{
		Subject:     principal.Subject,
		Email:       principal.Email,
		DisplayName: principal.DisplayName,
		Role:        string(principal.Role),
		Status:      string(principal.Status),
		AuthMode:    string(principal.AuthMode),
		Permissions: principalPermissions(principal.Role),
	}
}

func principalPermissions(role domain.UserRole) []string {
	permissions := []string{"batch:create", "batch:read", "dashboard:read", "plot:read", "orchard:read"}
	if role == domain.UserRoleAdmin {
		return append(permissions, "admin", "plot:write", "orchard:write", "user:write", "reconcile:write")
	}
	return permissions
}
