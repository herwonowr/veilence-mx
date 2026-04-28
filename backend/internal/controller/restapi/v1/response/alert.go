package response

import (
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// AlertResponse is the JSON representation of a security alert.
type AlertResponse struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	AnalysisID  string    `json:"analysisId"`
	ReleaseID   string    `json:"releaseId"`
	PackageID   string    `json:"packageId"`
	Severity    string    `json:"severity"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AlertFromEntity maps a domain Alert to a response DTO.
func AlertFromEntity(a *entity.Alert) AlertResponse {
	return AlertResponse{
		ID:          a.ID,
		WorkspaceID: a.WorkspaceID,
		AnalysisID:  a.AnalysisID,
		ReleaseID:   a.ReleaseID,
		PackageID:   a.PackageID,
		Severity:    string(a.Severity),
		Status:      string(a.Status),
		Message:     a.Message,
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// AlertWithPackageResponse includes the associated package info.
type AlertWithPackageResponse struct {
	AlertResponse
	PackageName      string `json:"packageName"`
	PackageEcosystem string `json:"packageEcosystem"`
}

// AlertWithPackageFromEntity maps a domain AlertWithPackage to a response DTO.
func AlertWithPackageFromEntity(a *entity.AlertWithPackage) AlertWithPackageResponse {
	return AlertWithPackageResponse{
		AlertResponse:    AlertFromEntity(&a.Alert),
		PackageName:      a.PackageName,
		PackageEcosystem: a.PackageEcosystem,
	}
}

// AlertsWithPackageFromEntities maps a slice of domain AlertWithPackage to response DTOs.
func AlertsWithPackageFromEntities(as []entity.AlertWithPackage) []AlertWithPackageResponse {
	result := make([]AlertWithPackageResponse, len(as))
	for i := range as {
		result[i] = AlertWithPackageFromEntity(&as[i])
	}
	return result
}

// AlertDetailResponse is the JSON representation of a single alert with package info (for GetAlert).
func AlertDetailFromEntity(a *entity.Alert, pkg *entity.Package) AlertWithPackageResponse {
	return AlertWithPackageResponse{
		AlertResponse:    AlertFromEntity(a),
		PackageName:      pkg.Name,
		PackageEcosystem: string(pkg.Ecosystem),
	}
}

// AlertNoteResponse is the JSON representation of an alert note.
type AlertNoteResponse struct {
	ID          string    `json:"id"`
	AlertID     string    `json:"alertId"`
	WorkspaceID string    `json:"workspaceId"`
	UserID      string    `json:"userId"`
	UserEmail   string    `json:"userEmail"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AlertNoteFromEntity maps a domain AlertNote to a response DTO.
func AlertNoteFromEntity(n *entity.AlertNote) AlertNoteResponse {
	return AlertNoteResponse{
		ID:          n.ID,
		WorkspaceID: n.WorkspaceID,
		AlertID:     n.AlertID,
		UserID:      n.UserID,
		UserEmail:   n.UserEmail,
		Content:     n.Content,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
	}
}

// AlertNotesFromEntities maps a slice of domain AlertNotes to response DTOs.
func AlertNotesFromEntities(ns []entity.AlertNote) []AlertNoteResponse {
	result := make([]AlertNoteResponse, len(ns))
	for i := range ns {
		result[i] = AlertNoteFromEntity(&ns[i])
	}
	return result
}
