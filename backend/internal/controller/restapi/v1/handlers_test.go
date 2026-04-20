package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	alertnoteuc "github.com/veilence/veilence-mx/backend/internal/usecase/alertnote"
	"github.com/veilence/veilence-mx/backend/internal/usecase/alertuc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/dashboarduc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/pkguc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/releaseuc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/settinguc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&persistent.User{},
		&persistent.Package{},
		&persistent.Release{},
		&persistent.Diff{},
		&persistent.Analysis{},
		&persistent.Alert{},
		&persistent.AlertNote{},
		&persistent.Setting{},
		&persistent.AuditLog{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func idStr(id uint) string {
	return fmt.Sprintf("%d", id)
}

// newDashboardHandlers creates a DashboardHandlers for testing.
func newDashboardHandlers(db *gorm.DB) *v1.DashboardHandlers {
	return newDashboardHandlersWithQueue(db, nil)
}

// newDashboardHandlersWithQueue creates a DashboardHandlers with a queue for testing.
func newDashboardHandlersWithQueue(db *gorm.DB, q *mockEnqueuer) *v1.DashboardHandlers {
	dashRepo := persistent.NewDashboardRepo(db)
	relRepo := persistent.NewReleaseRepo(db)
	diffRepo := persistent.NewDiffRepo(db)
	analysisRepo := persistent.NewAnalysisRepo(db)
	var svc *dashboarduc.UseCase
	if q != nil {
		svc = dashboarduc.New(dashRepo, relRepo, diffRepo, analysisRepo, q)
	} else {
		svc = dashboarduc.New(dashRepo, relRepo, diffRepo, analysisRepo, nil)
	}
	return &v1.DashboardHandlers{DashboardSvc: svc}
}

// newPackageHandlers creates a PackageHandlers for testing.
func newPackageHandlers(db *gorm.DB) *v1.PackageHandlers {
	auditSvc := audit.NewService(persistent.NewAuditLogRepo(db))
	pkgRepo := persistent.NewPackageRepo(db)
	pkgSvc := pkguc.New(pkgRepo, auditSvc)
	relRepo := persistent.NewReleaseRepo(db)
	diffRepo := persistent.NewDiffRepo(db)
	analysisRepo := persistent.NewAnalysisRepo(db)
	relSvc := releaseuc.New(pkgRepo, relRepo, diffRepo, analysisRepo, nil)
	return &v1.PackageHandlers{PkgSvc: pkgSvc, ReleaseSvc: relSvc, Audit: auditSvc}
}

// newPackageHandlersWithQueue creates a PackageHandlers with a mock queue for testing.
func newPackageHandlersWithQueue(db *gorm.DB, q *mockEnqueuer) *v1.PackageHandlers {
	auditSvc := audit.NewService(persistent.NewAuditLogRepo(db))
	pkgRepo := persistent.NewPackageRepo(db)
	pkgSvc := pkguc.New(pkgRepo, auditSvc)
	relRepo := persistent.NewReleaseRepo(db)
	diffRepo := persistent.NewDiffRepo(db)
	analysisRepo := persistent.NewAnalysisRepo(db)
	var relSvc *releaseuc.UseCase
	if q != nil {
		relSvc = releaseuc.New(pkgRepo, relRepo, diffRepo, analysisRepo, q)
	} else {
		relSvc = releaseuc.New(pkgRepo, relRepo, diffRepo, analysisRepo, nil)
	}
	return &v1.PackageHandlers{PkgSvc: pkgSvc, ReleaseSvc: relSvc, Audit: auditSvc}
}

// mockEnqueuer implements queue.Enqueuer for testing.
type mockEnqueuer struct {
	// calls records each Enqueue call as (jobType, referenceID).
	calls []enqueueCall
	// err is returned by Enqueue when non-nil (simulates Redis failure).
	err error
	// nextJobID is the job ID returned by Enqueue (incremented per call).
	nextJobID int
}

type enqueueCall struct {
	jobType     string
	workspaceID uint
	referenceID uint
}

func (m *mockEnqueuer) Enqueue(_ context.Context, jobType string, workspaceID, referenceID uint) (string, error) {
	m.calls = append(m.calls, enqueueCall{jobType: jobType, workspaceID: workspaceID, referenceID: referenceID})
	if m.err != nil {
		return "", m.err
	}
	m.nextJobID++
	return fmt.Sprintf("mock-%d", m.nextJobID), nil
}

// newAlertHandlers creates an AlertHandlers for testing.
func newAlertHandlers(db *gorm.DB) *v1.AlertHandlers {
	alertNoteRepo := persistent.NewAlertNoteRepo(db)
	alertRepo := persistent.NewAlertRepo(db)
	userRepo := persistent.NewUserRepo(db)
	noteSvc := alertnoteuc.New(alertNoteRepo, alertRepo, userRepo)
	auditSvc := audit.NewService(persistent.NewAuditLogRepo(db))
	alertSvc := alertuc.New(alertRepo, auditSvc)
	return &v1.AlertHandlers{AlertSvc: alertSvc, Notes: noteSvc, Audit: auditSvc}
}

// newSettingsHandlers creates a SettingsHandlers for testing.
func newSettingsHandlers(db *gorm.DB) *v1.SettingsHandlers {
	settingRepo := persistent.NewSettingRepo(db)
	settingSvc := settinguc.New(settingRepo)
	return &v1.SettingsHandlers{SettingSvc: settingSvc, Audit: audit.NewService(persistent.NewAuditLogRepo(db))}
}

// --- Dashboard ---

func TestGetDashboardStats(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlers(db)
	db.Create(&persistent.Package{Name: "requests", Ecosystem: "python"})
	db.Create(&persistent.Package{Name: "express", Ecosystem: "npm"})
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats", nil)
	w := httptest.NewRecorder()
	h.GetDashboardStats(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["totalPackages"])
	assert.Equal(t, float64(0), data["totalReleases"])
	assert.Equal(t, float64(0), data["activeAlerts"])
}

func TestGetDashboardStats_WithReleases(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)

	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: persistent.ReleaseStatusCompleted})
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: persistent.ReleaseStatusPending})
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "1.2.0", Status: persistent.ReleaseStatusAnalyzing})

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats", nil)
	w := httptest.NewRecorder()
	h.GetDashboardStats(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(3), data["totalReleases"])
	assert.Equal(t, float64(2), data["pendingAnalyses"]) // pending + analyzing
}

// --- Packages ---

func TestListPackages(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	db.Create(&persistent.Package{Name: "requests", Ecosystem: "python"})
	db.Create(&persistent.Package{Name: "flask", Ecosystem: "python", Source: persistent.PackageSourceManual})
	db.Create(&persistent.Package{Name: "express", Ecosystem: "npm"})
	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"all packages", "", 3},
		{"filter python", "ecosystem=python", 2},
		{"filter npm", "ecosystem=npm", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/packages?"+tt.query, nil)
			w := httptest.NewRecorder()
			h.ListPackages(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			data := resp["data"].([]any)
			assert.Len(t, data, tt.wantCount)
		})
	}
}

func TestListPackages_Pagination(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	for i := range 25 {
		db.Create(&persistent.Package{Name: fmt.Sprintf("pkg-%02d", i), Ecosystem: "python"})
	}

	// Default: page=1, limit=20
	req := httptest.NewRequest(http.MethodGet, "/api/packages", nil)
	w := httptest.NewRecorder()
	h.ListPackages(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 20)

	meta := resp["meta"].(map[string]any)
	assert.Equal(t, float64(25), meta["total"])
	assert.Equal(t, float64(1), meta["page"])

	// Page 2
	req = httptest.NewRequest(http.MethodGet, "/api/packages?page=2", nil)
	w = httptest.NewRecorder()
	h.ListPackages(w, req)

	json.NewDecoder(w.Body).Decode(&resp)
	data = resp["data"].([]any)
	assert.Len(t, data, 5)
}

func TestListPackages_FilterSource(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	db.Create(&persistent.Package{Name: "requests", Ecosystem: "python", Source: persistent.PackageSourceDiscovered})
	db.Create(&persistent.Package{Name: "my-pkg", Ecosystem: "python", Source: persistent.PackageSourceManual})

	req := httptest.NewRequest(http.MethodGet, "/api/packages?source=manual", nil)
	w := httptest.NewRecorder()
	h.ListPackages(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 1)
}

func TestCreatePackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid python", `{"name":"django","ecosystem":"python"}`, http.StatusCreated},
		{"valid npm", `{"name":"express","ecosystem":"npm"}`, http.StatusCreated},
		{"missing name", `{"name":"","ecosystem":"python"}`, http.StatusBadRequest},
		{"bad ecosystem", `{"name":"test","ecosystem":"rubygems"}`, http.StatusBadRequest},
		{"bad json", `{invalid}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db.Exec("DELETE FROM packages")
			req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.CreatePackage(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestCreatePackage_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	db.Create(&persistent.Package{Name: "requests", Ecosystem: "python"})

	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(`{"name":"requests","ecosystem":"python"}`))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreatePackage_SetsSource(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(`{"name":"custom-pkg","ecosystem":"npm"}`))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var pkg persistent.Package
	db.Where("name = ?", "custom-pkg").First(&pkg)
	assert.Equal(t, persistent.PackageSourceManual, pkg.Source)
	assert.Equal(t, persistent.PackageStatusActive, pkg.Status)
}

func TestGetPackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	r := chi.NewRouter()
	r.Get("/api/packages/{id}", h.GetPackage)
	req := httptest.NewRequest(http.MethodGet, "/api/packages/"+idStr(pkg.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "requests", data["name"])
}

func TestGetPackage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	r := chi.NewRouter()
	r.Get("/api/packages/{id}", h.GetPackage)
	req := httptest.NewRequest(http.MethodGet, "/api/packages/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetPackage_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	r := chi.NewRouter()
	r.Get("/api/packages/{id}", h.GetPackage)
	req := httptest.NewRequest(http.MethodGet, "/api/packages/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeletePackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "to-delete", Ecosystem: "npm", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceManual}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Delete("/api/packages/{id}", h.DeletePackage)

	req := httptest.NewRequest(http.MethodDelete, "/api/packages/"+idStr(pkg.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify status is now 'removed' (not hard-deleted)
	var updated persistent.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, persistent.PackageStatusRemoved, updated.Status)
}

func TestDeletePackage_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	r := chi.NewRouter()
	r.Delete("/api/packages/{id}", h.DeletePackage)

	req := httptest.NewRequest(http.MethodDelete, "/api/packages/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Block / Unblock ---

func TestBlockPackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/block", h.BlockPackage)

	body := `{"reason":"known benign, too many false positives"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/block", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "blocked", data["status"])
	assert.Equal(t, "known benign, too many false positives", data["blockedReason"])
	assert.NotNil(t, data["blockedAt"])

	// Verify in DB
	var updated persistent.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, persistent.PackageStatusBlocked, updated.Status)
	assert.NotNil(t, updated.BlockedAt)
}

func TestBlockPackage_NoBody(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "flask", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/block", h.BlockPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/block", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var updated persistent.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, persistent.PackageStatusBlocked, updated.Status)
}

func TestBlockPackage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/block", h.BlockPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/99999/block", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBlockPackage_AlreadyBlocked(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Package is already blocked - can't block again (repo returns not found for non-active)
	pkg := persistent.Package{Name: "bad-pkg", Ecosystem: "npm", Status: persistent.PackageStatusBlocked, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/block", h.BlockPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/block", nil)
	req.ContentLength = 0
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUnblockPackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusBlocked, Source: persistent.PackageSourceDiscovered, BlockedReason: "test"}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/unblock", h.UnblockPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/unblock", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "active", data["status"])

	// Verify in DB
	var updated persistent.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, persistent.PackageStatusActive, updated.Status)
	assert.Empty(t, updated.BlockedReason)
}

func TestUnblockPackage_NotBlocked(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Package is active - can't unblock
	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/unblock", h.UnblockPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/unblock", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePackage_AuditLogged(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "audit-test", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceManual}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Delete("/api/packages/{id}", h.DeletePackage)

	req := httptest.NewRequest(http.MethodDelete, "/api/packages/"+idStr(pkg.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify audit log was created
	var auditLog persistent.AuditLog
	result := db.Where("action = ? AND resource = ?", "remove", "package").First(&auditLog)
	require.NoError(t, result.Error)
	assert.Equal(t, pkg.ID, auditLog.ResourceID)
	assert.Contains(t, auditLog.Details, "audit-test")
}

func TestBlockPackage_AuditLogged(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "audit-block", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/block", h.BlockPackage)

	body := `{"reason":"false positive"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/block", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify audit log was created
	var auditLog persistent.AuditLog
	result := db.Where("action = ? AND resource = ?", "block", "package").First(&auditLog)
	require.NoError(t, result.Error)
	assert.Equal(t, pkg.ID, auditLog.ResourceID)
	assert.Contains(t, auditLog.Details, "false positive")
}

// --- Releases ---

func TestListPackageReleases(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"})
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "pending"})

	// Other package releases should not be included
	pkg2 := persistent.Package{Name: "flask", Ecosystem: "python"}
	db.Create(&pkg2)
	db.Create(&persistent.Release{PackageID: pkg2.ID, Version: "2.0.0", Status: "completed"})

	r := chi.NewRouter()
	r.Get("/api/packages/{id}/releases", h.ListPackageReleases)

	req := httptest.NewRequest(http.MethodGet, "/api/packages/"+idStr(pkg.ID)+"/releases", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 2)

	meta := resp["meta"].(map[string]any)
	assert.Equal(t, float64(2), meta["total"])
}

func TestListPackageReleases_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	r := chi.NewRouter()
	r.Get("/api/packages/{id}/releases", h.ListPackageReleases)

	req := httptest.NewRequest(http.MethodGet, "/api/packages/abc/releases", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetRelease(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)

	r := chi.NewRouter()
	r.Get("/api/releases/{id}", h.GetRelease)

	req := httptest.NewRequest(http.MethodGet, "/api/releases/"+idStr(rel.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "1.0.0", data["version"])
	assert.NotNil(t, data["package"])
}

func TestGetRelease_WithDiffAndAnalysis(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)

	diff := persistent.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "test diff", FileChangesCount: 1}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "benign", Confidence: 0.95, Reasoning: "OK", ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)

	r := chi.NewRouter()
	r.Get("/api/releases/{id}", h.GetRelease)

	req := httptest.NewRequest(http.MethodGet, "/api/releases/"+idStr(rel2.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.NotNil(t, data["diff"])
	assert.NotNil(t, data["analysis"])
}

func TestGetRelease_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	r := chi.NewRouter()
	r.Get("/api/releases/{id}", h.GetRelease)

	req := httptest.NewRequest(http.MethodGet, "/api/releases/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Alerts ---

func TestGetAlert(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 1}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := persistent.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, WorkspaceID: 0, Severity: "critical", Status: "new", Message: "Malicious detected"}
	db.Create(&alert)

	r := chi.NewRouter()
	r.Get("/api/alerts/{id}", h.GetAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts/"+idStr(alert.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "Malicious detected", data["message"])
	assert.Equal(t, "critical", data["severity"])
	assert.Equal(t, "requests", data["packageName"])
	assert.Equal(t, "python", data["packageEcosystem"])
}

func TestGetAlert_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	r := chi.NewRouter()
	r.Get("/api/alerts/{id}", h.GetAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts/99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAlert_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	r := chi.NewRouter()
	r.Get("/api/alerts/{id}", h.GetAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts/abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAlert_WorkspaceScoping(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 5}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := persistent.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	// Alert belongs to workspace 5 - request context has workspaceID=0 (default), so it should not be found
	alert := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, WorkspaceID: 5, Severity: "high", Status: "new", Message: "Other org alert"}
	db.Create(&alert)

	r := chi.NewRouter()
	r.Get("/api/alerts/{id}", h.GetAlert)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts/"+idStr(alert.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// WorkspaceID from context is 0 (no middleware), alert belongs to workspace 5 → not found
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListAlerts(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := persistent.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)

	db.Create(&persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"})
	db.Create(&persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "medium", Status: "acknowledged", Message: "Maybe bad"})

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"all alerts", "", 2},
		{"filter critical", "severity=critical", 1},
		{"filter new", "status=new", 1},
		{"filter acknowledged", "status=acknowledged", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/alerts?"+tt.query, nil)
			w := httptest.NewRecorder()
			h.ListAlerts(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			data := resp["data"].([]any)
			assert.Len(t, data, tt.wantCount)
		})
	}
}

func TestUpdateAlert(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)
	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := persistent.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "test diff", FileChangesCount: 1, LinesAdded: 5, LinesRemoved: 2}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, Reasoning: "Bad code", ModelUsed: "test", AnalyzerType: "api"}
	db.Create(&analysis)
	alert := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Malicious detected"}
	db.Create(&alert)
	r := chi.NewRouter()
	r.Patch("/api/alerts/{id}", h.UpdateAlert)
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"acknowledge", `{"status":"acknowledged"}`, http.StatusOK},
		{"resolve", `{"status":"resolved"}`, http.StatusOK},
		{"back to new", `{"status":"new"}`, http.StatusOK},
		{"invalid status", `{"status":"invalid"}`, http.StatusBadRequest},
		{"bad json", `{bad`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/api/alerts/"+idStr(alert.ID), strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestUpdateAlert_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)
	r := chi.NewRouter()
	r.Patch("/api/alerts/{id}", h.UpdateAlert)

	req := httptest.NewRequest(http.MethodPatch, "/api/alerts/99999", strings.NewReader(`{"status":"acknowledged"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateAlert_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)
	r := chi.NewRouter()
	r.Patch("/api/alerts/{id}", h.UpdateAlert)

	req := httptest.NewRequest(http.MethodPatch, "/api/alerts/abc", strings.NewReader(`{"status":"acknowledged"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Settings ---

func TestGetSettings(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)
	db.Create(&persistent.Setting{Key: "monitoring_interval", Value: "1h"})
	db.Create(&persistent.Setting{Key: "discovery_interval", Value: "24h"})
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	h.GetSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "1h", data["monitoring_interval"])
	assert.Equal(t, "24h", data["discovery_interval"])
}

func TestGetSettings_Empty(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	h.GetSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateSettings_ValidKeys(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)

	body := `{"monitoring_interval":"30m","discovery_scan_depth":"100"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify settings were saved
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "30m", data["monitoring_interval"])
	assert.Equal(t, "100", data["discovery_scan_depth"])
}

func TestUpdateSettings_Upsert(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)

	// Create initial
	db.Create(&persistent.Setting{Key: "monitoring_interval", Value: "1h"})

	// Update existing
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"monitoring_interval":"30m"}`))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var setting persistent.Setting
	db.Where("key = ?", "monitoring_interval").First(&setting)
	assert.Equal(t, "30m", setting.Value)
}

func TestUpdateSettings_InvalidKey(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"invalid_key":"val"}`))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSettings_BadJSON(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{bad}`))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSettings_DiscoverySettings(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)

	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"discovery_scan_depth":"200","discovery_interval":"12h"}`))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var depth, interval persistent.Setting
	db.Where("key = ?", "discovery_scan_depth").First(&depth)
	db.Where("key = ?", "discovery_interval").First(&interval)
	assert.Equal(t, "200", depth.Value)
	assert.Equal(t, "12h", interval.Value)
}

func TestUpdateSettings_EmailDigest(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)

	body := `{"email_digest_enabled":"true","email_digest_frequency":"weekly","email_digest_recipients":"admin@example.com,ops@example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var enabled, frequency, recipients persistent.Setting
	db.Where("key = ?", "email_digest_enabled").First(&enabled)
	db.Where("key = ?", "email_digest_frequency").First(&frequency)
	db.Where("key = ?", "email_digest_recipients").First(&recipients)
	assert.Equal(t, "true", enabled.Value)
	assert.Equal(t, "weekly", frequency.Value)
	assert.Equal(t, "admin@example.com,ops@example.com", recipients.Value)
}

// --- Recent Releases ---

func TestGetRecentReleases(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"})
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "pending"})

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/recent", nil)
	w := httptest.NewRecorder()
	h.GetRecentReleases(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 2)

	// Should have package info
	first := data[0].(map[string]any)
	assert.Equal(t, "requests", first["packageName"])
	assert.Equal(t, "python", first["packageEcosystem"])
}

// --- Reanalyze ---

func TestReanalyzeAll_NoQueue(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlers(db)

	req := httptest.NewRequest(http.MethodPost, "/api/reanalyze", nil)
	w := httptest.NewRecorder()
	h.ReanalyzeAll(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReanalyzeAll_WithDiffs(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlersWithQueue(db, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)

	// Diff without analysis
	db.Create(&persistent.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"})

	// Queue is nil, so reanalyze returns error
	req := httptest.NewRequest(http.MethodPost, "/api/reanalyze", nil)
	w := httptest.NewRecorder()
	h.ReanalyzeAll(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReanalyzeAll_NoDiffs(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlersWithQueue(db, nil)

	// Queue is nil, so reanalyze returns error regardless of diffs
	req := httptest.NewRequest(http.MethodPost, "/api/reanalyze", nil)
	w := httptest.NewRecorder()
	h.ReanalyzeAll(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Pagination Helper ---

func TestParsePagination(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"default limit", "", 5}, // 5 items, limit 20 -> all shown
		{"custom limit", "limit=2", 2},
		{"limit over max", "limit=200", 5}, // capped to 100, but only 5 items
	}

	for i := range 5 {
		db.Create(&persistent.Package{Name: fmt.Sprintf("pkg-%d", i), Ecosystem: "python"})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/packages?"+tt.query, nil)
			w := httptest.NewRecorder()
			h.ListPackages(w, req)

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			data := resp["data"].([]any)
			assert.Len(t, data, tt.wantCount)
		})
	}
}

// =====================================
// v1.0.1 NEW ENDPOINT TESTS
// =====================================

// --- Alert Search ---

func TestListAlerts_Search(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	pkg1 := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg1)
	pkg2 := persistent.Package{Name: "express", Ecosystem: "npm"}
	db.Create(&pkg2)

	// Set up required chain for alerts
	rel1 := persistent.Release{PackageID: pkg1.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	diff1 := persistent.Diff{ReleaseID: rel1.ID, DiffContent: "d"}
	db.Create(&diff1)
	analysis1 := persistent.Analysis{DiffID: diff1.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis1)

	rel2 := persistent.Release{PackageID: pkg2.ID, Version: "2.0.0", Status: "completed"}
	db.Create(&rel2)
	diff2 := persistent.Diff{ReleaseID: rel2.ID, DiffContent: "d"}
	db.Create(&diff2)
	analysis2 := persistent.Analysis{DiffID: diff2.ID, Classification: "suspicious", Confidence: 0.8, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis2)

	db.Create(&persistent.Alert{AnalysisID: analysis1.ID, PackageID: pkg1.ID, Severity: "critical", Status: "new", Message: "Malicious in requests"})
	db.Create(&persistent.Alert{AnalysisID: analysis2.ID, PackageID: pkg2.ID, Severity: "medium", Status: "new", Message: "Suspicious in express"})

	tests := []struct {
		name      string
		search    string
		wantCount int
	}{
		{"search by package name", "requests", 1},
		{"search by message", "Suspicious", 1},
		{"search no results", "nonexistent", 0},
		{"empty search", "", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := ""
			if tt.search != "" {
				query = "search=" + tt.search
			}
			req := httptest.NewRequest(http.MethodGet, "/api/alerts?"+query, nil)
			w := httptest.NewRecorder()
			h.ListAlerts(w, req)
			require.Equal(t, http.StatusOK, w.Code)

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			data := resp["data"].([]any)
			assert.Len(t, data, tt.wantCount)
		})
	}
}

// --- Alert Notes ---

func TestListAlertNotes(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := persistent.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"}
	db.Create(&alert)

	// Create some notes
	db.Create(&persistent.AlertNote{AlertID: alert.ID, WorkspaceID: 0, UserID: 1, UserEmail: "user@example.com", Content: "First note"})
	db.Create(&persistent.AlertNote{AlertID: alert.ID, WorkspaceID: 0, UserID: 2, UserEmail: "other@example.com", Content: "Second note"})

	r := chi.NewRouter()
	r.Get("/api/alerts/{id}/notes", h.ListAlertNotes)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts/"+idStr(alert.ID)+"/notes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 2)
}

func TestListAlertNotes_AlertNotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	r := chi.NewRouter()
	r.Get("/api/alerts/{id}/notes", h.ListAlertNotes)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts/99999/notes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateAlertNote(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	// Create user for email lookup
	db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, email TEXT, first_name TEXT, last_name TEXT, password_hash TEXT, is_active INTEGER DEFAULT 1, email_verified INTEGER DEFAULT 0, last_login_at TEXT, created_at TEXT, updated_at TEXT, deleted_at TEXT)")
	db.Exec("INSERT INTO users (id, email, first_name, last_name, password_hash) VALUES (1, 'user@example.com', 'Test', 'User', 'hash')")

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := persistent.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"}
	db.Create(&alert)

	r := chi.NewRouter()
	r.Post("/api/alerts/{id}/notes", h.CreateAlertNote)

	body := `{"content":"This looks like a false positive"}`
	req := httptest.NewRequest(http.MethodPost, "/api/alerts/"+idStr(alert.ID)+"/notes", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "This looks like a false positive", data["content"])
}

func TestCreateAlertNote_EmptyContent(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := persistent.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"}
	db.Create(&alert)

	r := chi.NewRouter()
	r.Post("/api/alerts/{id}/notes", h.CreateAlertNote)

	body := `{"content":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/alerts/"+idStr(alert.ID)+"/notes", strings.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Bulk Package Import ---

func TestImportPackages(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"packages":[{"name":"django","ecosystem":"python"},{"name":"express","ecosystem":"npm"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["imported"])
	assert.Equal(t, float64(0), data["skipped"])

	// Verify packages exist in DB
	var count int64
	db.Model(&persistent.Package{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestImportPackages_Duplicates(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Pre-create a package
	db.Create(&persistent.Package{Name: "django", Ecosystem: "python"})

	body := `{"packages":[{"name":"django","ecosystem":"python"},{"name":"flask","ecosystem":"python"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(1), data["imported"])
	assert.Equal(t, float64(1), data["skipped"])
}

func TestImportPackages_InvalidEcosystem(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"packages":[{"name":"test","ecosystem":"rubygems"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(0), data["imported"])
	errors := data["errors"].([]any)
	assert.Len(t, errors, 1)
}

func TestImportPackages_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"packages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportPackages_TooMany(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Build a body with 501 packages
	entries := make([]string, 501)
	for i := range entries {
		entries[i] = fmt.Sprintf(`{"name":"pkg-%d","ecosystem":"python"}`, i)
	}
	body := `{"packages":[` + strings.Join(entries, ",") + `]}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportPackages_RequirementsTxt(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	content := "# This is a comment\nrequests==2.31.0\nflask>=2.0\nnumpy\n-i https://pypi.org\ndjango~=4.0\nsetuptools[extra]>=60.0\n\n"
	body := fmt.Sprintf(`{"format":"requirements_txt","content":%q}`, content)
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(5), data["imported"]) // requests, flask, numpy, django, setuptools
	assert.Equal(t, float64(0), data["skipped"])
}

func TestImportPackages_PackageJSON(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkgJSON := `{
		"name": "my-app",
		"dependencies": {"express": "^4.18.0", "lodash": "^4.17.21"},
		"devDependencies": {"jest": "^29.0.0", "typescript": "^5.0.0"}
	}`
	body := fmt.Sprintf(`{"format":"package_json","content":%q}`, pkgJSON)
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(4), data["imported"]) // express, lodash, jest, typescript
}

func TestImportPackages_ListFormat(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	content := "python:requests\nnpm:express\npython:flask\n# comment\n\nnpm:lodash"
	body := fmt.Sprintf(`{"format":"list","content":%q}`, content)
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(4), data["imported"])
}

func TestImportPackages_InvalidFormat(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"format":"invalid","content":"some content"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportPackages_FormatWithoutContent(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"format":"requirements_txt"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestImportPackages_InvalidPackageName(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"packages":[{"name":"valid-pkg","ecosystem":"python"},{"name":"evil pkg!","ecosystem":"python"},{"name":"ok.pkg","ecosystem":"npm"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["imported"]) // valid-pkg and ok.pkg
	errors := data["errors"].([]any)
	assert.Len(t, errors, 1) // evil pkg! has invalid characters
}

func TestCreatePackage_InvalidName(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"name":"invalid name!@#$","ecosystem":"python"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePackage_NameTooLong(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	longName := strings.Repeat("a", 201)
	body := fmt.Sprintf(`{"name":"%s","ecosystem":"python"}`, longName)
	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePackage_ScopedNpmPackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"name":"@angular/core","ecosystem":"npm"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// --- Re-analyze Release ---

func TestReanalyzeRelease_NoQueue(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db) // Queue is nil

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 0}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)

	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/"+idStr(rel.ID)+"/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReanalyzeRelease_NotFound(t *testing.T) {
	db := setupTestDB(t)
	mq := &mockEnqueuer{}
	h := newPackageHandlersWithQueue(db, mq)

	// No release in DB - should get 404 after passing queue nil check
	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/99999/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, mq.calls, "should not enqueue anything for nonexistent release")
}

func TestReanalyzeRelease_SuccessNoDiff(t *testing.T) {
	db := setupTestDB(t)
	mq := &mockEnqueuer{}
	h := newPackageHandlersWithQueue(db, mq)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 0}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)

	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/"+idStr(rel.ID)+"/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify a diff job was enqueued (no diff exists, so it re-diffs from scratch)
	require.Len(t, mq.calls, 1)
	assert.Equal(t, "diff", mq.calls[0].jobType)
	assert.Equal(t, rel.ID, mq.calls[0].referenceID)

	// Verify release status was reset to pending
	var updated persistent.Release
	db.First(&updated, rel.ID)
	assert.Equal(t, persistent.ReleaseStatusPending, updated.Status)

	// Verify response contains jobId
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Contains(t, data["message"], "diffing")
	assert.NotEmpty(t, data["jobId"])
}

func TestReanalyzeRelease_SuccessWithDiff(t *testing.T) {
	db := setupTestDB(t)
	mq := &mockEnqueuer{}
	h := newPackageHandlersWithQueue(db, mq)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 0}
	db.Create(&pkg)
	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := persistent.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "some diff"}
	db.Create(&diff)

	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/"+idStr(rel2.ID)+"/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify an analyze job was enqueued (diff exists, so it re-analyzes)
	require.Len(t, mq.calls, 1)
	assert.Equal(t, "analyze", mq.calls[0].jobType)
	assert.Equal(t, diff.ID, mq.calls[0].referenceID)

	// Verify release status was set to analyzing
	var updated persistent.Release
	db.First(&updated, rel2.ID)
	assert.Equal(t, persistent.ReleaseStatusAnalyzing, updated.Status)

	// Verify response contains jobId
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Contains(t, data["message"], "analysis")
	assert.NotEmpty(t, data["jobId"])
}

func TestReanalyzeRelease_WorkspaceScoping(t *testing.T) {
	db := setupTestDB(t)
	mq := &mockEnqueuer{}
	h := newPackageHandlersWithQueue(db, mq)

	// Create a release belonging to workspace 5; request context has workspaceID=0 (default, no middleware)
	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 5}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)

	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/"+idStr(rel.ID)+"/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// WorkspaceID from context is 0, package belongs to workspace 5 → not found
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Empty(t, mq.calls, "should not enqueue anything for cross-org release")
}

func TestReanalyzeRelease_EnqueueError(t *testing.T) {
	db := setupTestDB(t)
	mq := &mockEnqueuer{err: fmt.Errorf("redis connection refused")}
	h := newPackageHandlersWithQueue(db, mq)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 0}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)

	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/"+idStr(rel.ID)+"/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// Enqueue was attempted but failed
	require.Len(t, mq.calls, 1)
}

func TestReanalyzeRelease_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/abc/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Analysis History ---

func TestGetAnalysisHistory(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)

	// Baseline release (completed, no diff)
	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)

	// Analyzed release
	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := persistent.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"}
	db.Create(&diff)
	db.Create(&persistent.Analysis{DiffID: diff.ID, Classification: "benign", Confidence: 0.95, Reasoning: "OK", ModelUsed: "test", AnalyzerType: "copilot"})

	// Pending release (no diff, not completed)
	rel3 := persistent.Release{PackageID: pkg.ID, Version: "1.2.0", Status: "pending"}
	db.Create(&rel3)

	r := chi.NewRouter()
	r.Get("/api/packages/{id}/analysis-history", h.GetAnalysisHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/packages/"+idStr(pkg.ID)+"/analysis-history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	// Should include baseline + analyzed, NOT pending
	assert.Len(t, data, 2)
}

func TestGetAnalysisHistory_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	r := chi.NewRouter()
	r.Get("/api/packages/{id}/analysis-history", h.GetAnalysisHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/packages/99999/analysis-history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAnalysisHistory_Empty(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "brand-new", Ecosystem: "python"}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Get("/api/packages/{id}/analysis-history", h.GetAnalysisHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/packages/"+idStr(pkg.ID)+"/analysis-history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 0)
}

// =====================================
// v1.1.0 SUGGESTION / APPROVAL / STALE TESTS
// =====================================

// --- Suggestions ---

func TestListSuggestions(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	db.Create(&persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered})
	db.Create(&persistent.Package{Name: "flask", Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered})
	db.Create(&persistent.Package{Name: "express", Ecosystem: "npm", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceManual})

	req := httptest.NewRequest(http.MethodGet, "/api/packages/suggestions", nil)
	w := httptest.NewRecorder()
	h.ListSuggestions(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 2) // Only suggested packages

	meta := resp["meta"].(map[string]any)
	assert.Equal(t, float64(2), meta["total"])
}

func TestListSuggestions_Empty(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	req := httptest.NewRequest(http.MethodGet, "/api/packages/suggestions", nil)
	w := httptest.NewRecorder()
	h.ListSuggestions(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 0)
}

func TestListSuggestions_Pagination(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	for i := range 25 {
		db.Create(&persistent.Package{Name: fmt.Sprintf("pkg-%02d", i), Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/packages/suggestions?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	h.ListSuggestions(w, req)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	assert.Len(t, data, 10)

	meta := resp["meta"].(map[string]any)
	assert.Equal(t, float64(25), meta["total"])
}

// --- Approve ---

func TestApprovePackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/approve", h.ApprovePackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "active", data["status"])

	// Verify in DB
	var updated persistent.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, persistent.PackageStatusActive, updated.Status)
}

func TestApprovePackage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/approve", h.ApprovePackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/99999/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestApprovePackage_NotSuggested(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Active package can't be approved (only suggested can)
	pkg := persistent.Package{Name: "flask", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceManual}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/approve", h.ApprovePackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestApprovePackage_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/approve", h.ApprovePackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/abc/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestApprovePackage_AuditLogged(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "audit-approve", Ecosystem: "npm", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/approve", h.ApprovePackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/approve", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify audit log was created
	var auditLog persistent.AuditLog
	result := db.Where("action = ? AND resource = ?", "approve", "package").First(&auditLog)
	require.NoError(t, result.Error)
	assert.Equal(t, pkg.ID, auditLog.ResourceID)
	assert.Contains(t, auditLog.Details, "audit-approve")
}

// --- Reject ---

func TestRejectPackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "malicious-pkg", Ecosystem: "npm", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/reject", h.RejectPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/reject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify in DB - status should be removed
	var updated persistent.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, persistent.PackageStatusRemoved, updated.Status)
}

func TestRejectPackage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/reject", h.RejectPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/99999/reject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRejectPackage_NotSuggested(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := persistent.Package{Name: "active-pkg", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceManual}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/reject", h.RejectPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/"+idStr(pkg.ID)+"/reject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRejectPackage_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	r := chi.NewRouter()
	r.Post("/api/packages/{id}/reject", h.RejectPackage)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/abc/reject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Bulk Approve ---

func TestBulkApprovePackages_ByIDs(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg1 := persistent.Package{Name: "pkg-1", Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg1)
	pkg2 := persistent.Package{Name: "pkg-2", Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg2)
	pkg3 := persistent.Package{Name: "pkg-3", Ecosystem: "npm", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceManual}
	db.Create(&pkg3)

	body := fmt.Sprintf(`{"packageIds":[%d,%d]}`, pkg1.ID, pkg2.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-approve", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.BulkApprovePackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["approved"])

	// Verify in DB
	var updated1, updated2 persistent.Package
	db.First(&updated1, pkg1.ID)
	db.First(&updated2, pkg2.ID)
	assert.Equal(t, persistent.PackageStatusActive, updated1.Status)
	assert.Equal(t, persistent.PackageStatusActive, updated2.Status)
}

func TestBulkApprovePackages_ByEcosystem(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	db.Create(&persistent.Package{Name: "py-1", Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered})
	db.Create(&persistent.Package{Name: "py-2", Ecosystem: "python", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered})
	db.Create(&persistent.Package{Name: "npm-1", Ecosystem: "npm", Status: persistent.PackageStatusSuggested, Source: persistent.PackageSourceDiscovered})

	body := `{"ecosystem":"python"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-approve", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.BulkApprovePackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["approved"])

	// npm package should still be suggested
	var npmPkg persistent.Package
	db.Where("name = ?", "npm-1").First(&npmPkg)
	assert.Equal(t, persistent.PackageStatusSuggested, npmPkg.Status)
}

func TestBulkApprovePackages_NoParams(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-approve", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.BulkApprovePackages(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkApprovePackages_InvalidEcosystem(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-approve", strings.NewReader(`{"ecosystem":"rubygems"}`))
	w := httptest.NewRecorder()
	h.BulkApprovePackages(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBulkApprovePackages_BadJSON(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-approve", strings.NewReader(`{bad}`))
	w := httptest.NewRecorder()
	h.BulkApprovePackages(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Stale Packages ---

func TestListStalePackages(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Package with no download count update (stale)
	db.Create(&persistent.Package{Name: "old-pkg", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered})
	// Package with recent download count update (not stale)
	now := time.Now()
	db.Create(&persistent.Package{Name: "fresh-pkg", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered, DownloadCountUpdatedAt: &now})

	req := httptest.NewRequest(http.MethodGet, "/api/packages/stale?months=6", nil)
	w := httptest.NewRecorder()
	h.ListStalePackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	// old-pkg has NULL download_count_updated_at, fresh-pkg is recent - old-pkg should be stale
	assert.GreaterOrEqual(t, len(data), 1)
}

func TestListStalePackages_DefaultMonths(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	req := httptest.NewRequest(http.MethodGet, "/api/packages/stale", nil)
	w := httptest.NewRecorder()
	h.ListStalePackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListStalePackages_InvalidMonths(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	tests := []struct {
		name   string
		months string
	}{
		{"zero", "0"},
		{"negative", "-1"},
		{"too high", "25"},
		{"non-numeric", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/packages/stale?months="+tt.months, nil)
			w := httptest.NewRecorder()
			h.ListStalePackages(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// --- Route Registration (smoke test) ---

func TestSuggestionsRouteRegistered(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Create a router that mirrors the production route registration
	r := chi.NewRouter()
	r.Route("/api/packages", func(r chi.Router) {
		r.Get("/suggestions", h.ListSuggestions)
		r.Get("/stale", h.ListStalePackages)
		r.Post("/bulk-approve", h.BulkApprovePackages)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetPackage)
			r.Post("/approve", h.ApprovePackage)
			r.Post("/reject", h.RejectPackage)
		})
	})

	// Verify /suggestions doesn't match /{id}
	req := httptest.NewRequest(http.MethodGet, "/api/packages/suggestions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify /stale doesn't match /{id}
	req = httptest.NewRequest(http.MethodGet, "/api/packages/stale", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
