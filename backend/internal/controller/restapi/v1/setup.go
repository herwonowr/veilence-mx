package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/setup"
)

// SetupHandlers handles the initial setup endpoints.
type SetupHandlers struct {
	Setup *setup.Service
}

type initializeRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	WorkspaceName string `json:"workspaceName"`
	WorkspaceSlug string `json:"workspaceSlug"`
}

// Initialize handles POST /api/setup/initialize - creates the first user and workspace.
func (h *SetupHandlers) Initialize(w http.ResponseWriter, r *http.Request) {
	var req initializeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.WorkspaceName = strings.TrimSpace(req.WorkspaceName)
	req.WorkspaceSlug = strings.TrimSpace(req.WorkspaceSlug)

	if err := validation.ValidateEmail(req.Email); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}
	if err := validation.ValidatePassword(req.Password); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}
	if req.FirstName == "" {
		respondAppError(w, Validation("firstName is required"))
		return
	}
	if req.WorkspaceName == "" {
		respondAppError(w, Validation("workspaceName is required"))
		return
	}
	if req.WorkspaceSlug == "" {
		respondAppError(w, Validation("workspaceSlug is required"))
		return
	}

	result, err := h.Setup.Initialize(r.Context(), setup.InitializeRequest{
		Email:         req.Email,
		Password:      req.Password,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		WorkspaceName: req.WorkspaceName,
		WorkspaceSlug: req.WorkspaceSlug,
	})
	if err != nil {
		if errors.Is(err, entity.ErrSetupAlreadyCompleted) {
			respondError(w, http.StatusConflict, "Setup has already been completed")
			return
		}
		if errors.Is(err, entity.ErrValidation) {
			respondAppError(w, Validation(err.Error()))
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to initialize setup")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"user":         response.UserFromEntity(result.User),
		"accessToken":  result.AccessToken,
		"refreshToken": result.RefreshToken,
		"workspace": map[string]any{
			"id":   result.Workspace.ID,
			"name": result.Workspace.Name,
			"slug": result.Workspace.Slug,
		},
	}, nil)
}
