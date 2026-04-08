package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lychee-ripe/gateway/internal/domain"
	"github.com/lychee-ripe/gateway/internal/service"
)

type orchardService interface {
	List(ctx context.Context, includeArchived bool) ([]domain.OrchardRecord, error)
	Create(ctx context.Context, input service.OrchardInput) (domain.OrchardRecord, error)
	Update(ctx context.Context, orchardID string, input service.OrchardInput) (domain.OrchardRecord, error)
	Archive(ctx context.Context, orchardID string) (domain.OrchardRecord, error)
}

type plotService interface {
	List(ctx context.Context, orchardID string, includeArchived bool) ([]domain.PlotRecord, error)
	Create(ctx context.Context, input service.PlotInput) (domain.PlotRecord, error)
	Update(ctx context.Context, plotID string, input service.PlotInput) (domain.PlotRecord, error)
	Archive(ctx context.Context, plotID string) (domain.PlotRecord, error)
}

type userAdminService interface {
	ListUsers(ctx context.Context) ([]domain.UserRecord, error)
	CreateUser(ctx context.Context, input service.UserCreateInput) (domain.UserRecord, error)
	UpdateUser(ctx context.Context, input service.UserUpdateInput) (domain.UserRecord, error)
}

type orchardRequest struct {
	OrchardID   string `json:"orchard_id"`
	OrchardName string `json:"orchard_name"`
	Status      string `json:"status"`
}

type plotRequest struct {
	PlotID    string `json:"plot_id"`
	OrchardID string `json:"orchard_id"`
	PlotName  string `json:"plot_name"`
	Status    string `json:"status"`
}

type userRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type orchardResponse struct {
	OrchardID   string    `json:"orchard_id"`
	OrchardName string    `json:"orchard_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type plotResponse struct {
	PlotID    string    `json:"plot_id"`
	OrchardID string    `json:"orchard_id"`
	PlotName  string    `json:"plot_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type userResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	OIDCSubject *string    `json:"oidc_subject"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func ListOrchards(svc orchardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.List(r.Context(), queryBool(r, "include_archived"))
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		resp := make([]orchardResponse, 0, len(items))
		for _, item := range items {
			resp = append(resp, toOrchardResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": resp})
	}
}

func CreateOrchard(svc orchardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req orchardRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		created, err := svc.Create(r.Context(), service.OrchardInput{
			OrchardID:   req.OrchardID,
			OrchardName: req.OrchardName,
			Status:      domain.ResourceStatus(strings.TrimSpace(req.Status)),
		})
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, toOrchardResponse(created))
	}
}

func UpdateOrchard(svc orchardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req orchardRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		updated, err := svc.Update(r.Context(), r.PathValue("orchard_id"), service.OrchardInput{
			OrchardName: req.OrchardName,
			Status:      domain.ResourceStatus(strings.TrimSpace(req.Status)),
		})
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toOrchardResponse(updated))
	}
}

func ArchiveOrchard(svc orchardService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updated, err := svc.Archive(r.Context(), r.PathValue("orchard_id"))
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toOrchardResponse(updated))
	}
}

func ListPlots(svc plotService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.List(r.Context(), strings.TrimSpace(r.URL.Query().Get("orchard_id")), queryBool(r, "include_archived"))
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		resp := make([]plotResponse, 0, len(items))
		for _, item := range items {
			resp = append(resp, toPlotResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": resp})
	}
}

func CreatePlot(svc plotService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req plotRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		created, err := svc.Create(r.Context(), service.PlotInput{
			PlotID:    req.PlotID,
			OrchardID: req.OrchardID,
			PlotName:  req.PlotName,
			Status:    domain.ResourceStatus(strings.TrimSpace(req.Status)),
		})
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, toPlotResponse(created))
	}
}

func UpdatePlot(svc plotService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req plotRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		updated, err := svc.Update(r.Context(), r.PathValue("plot_id"), service.PlotInput{
			OrchardID: req.OrchardID,
			PlotName:  req.PlotName,
			Status:    domain.ResourceStatus(strings.TrimSpace(req.Status)),
		})
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toPlotResponse(updated))
	}
}

func ArchivePlot(svc plotService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updated, err := svc.Archive(r.Context(), r.PathValue("plot_id"))
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toPlotResponse(updated))
	}
}

func ListUsers(svc userAdminService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.ListUsers(r.Context())
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		resp := make([]userResponse, 0, len(items))
		for _, item := range items {
			resp = append(resp, toUserResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": resp})
	}
}

func CreateUser(svc userAdminService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req userRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		created, err := svc.CreateUser(r.Context(), service.UserCreateInput{
			Email:       req.Email,
			DisplayName: req.DisplayName,
			Role:        domain.UserRole(strings.TrimSpace(req.Role)),
			Status:      domain.UserStatus(strings.TrimSpace(req.Status)),
		})
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, toUserResponse(created))
	}
}

func UpdateUser(svc userAdminService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req userRequest
		if err := decodeJSONBody(r, &req); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
			return
		}
		updated, err := svc.UpdateUser(r.Context(), service.UserUpdateInput{
			ID:          r.PathValue("user_id"),
			Email:       req.Email,
			DisplayName: req.DisplayName,
			Role:        domain.UserRole(strings.TrimSpace(req.Role)),
			Status:      domain.UserStatus(strings.TrimSpace(req.Status)),
		})
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, toUserResponse(updated))
	}
}

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRequest):
		writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error(), nil)
	case errors.Is(err, service.ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", err.Error(), nil)
	case errors.Is(err, service.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", err.Error(), nil)
	default:
		writeError(w, r, http.StatusServiceUnavailable, "service_unavailable", "service unavailable", nil)
	}
}

func toOrchardResponse(item domain.OrchardRecord) orchardResponse {
	return orchardResponse{
		OrchardID:   item.OrchardID,
		OrchardName: item.OrchardName,
		Status:      string(item.Status),
		CreatedAt:   item.CreatedAt.UTC(),
		UpdatedAt:   item.UpdatedAt.UTC(),
	}
}

func toPlotResponse(item domain.PlotRecord) plotResponse {
	return plotResponse{
		PlotID:    item.PlotID,
		OrchardID: item.OrchardID,
		PlotName:  item.PlotName,
		Status:    string(item.Status),
		CreatedAt: item.CreatedAt.UTC(),
		UpdatedAt: item.UpdatedAt.UTC(),
	}
}

func toUserResponse(item domain.UserRecord) userResponse {
	return userResponse{
		ID:          item.ID,
		Email:       item.Email,
		DisplayName: item.DisplayName,
		OIDCSubject: item.OIDCSubject,
		Role:        string(item.Role),
		Status:      string(item.Status),
		LastLoginAt: item.LastLoginAt,
		CreatedAt:   item.CreatedAt.UTC(),
		UpdatedAt:   item.UpdatedAt.UTC(),
	}
}

func queryBool(r *http.Request, key string) bool {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}
