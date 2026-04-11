package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veilence/veilence-mx/backend/internal/api/handlers"
	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.Package{},
		&models.Release{},
		&models.Diff{},
		&models.Analysis{},
		&models.Alert{},
		&models.AlertNote{},
		&models.Setting{},
		&models.AuditLog{},
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
func newDashboardHandlers(db *gorm.DB) *handlers.DashboardHandlers {
	return &handlers.DashboardHandlers{DB: db, Dashboard: repository.NewDashboardRepo(db)}
}

// newPackageHandlers creates a PackageHandlers for testing.
func newPackageHandlers(db *gorm.DB) *handlers.PackageHandlers {
	return &handlers.PackageHandlers{DB: db, Queue: nil, Audit: audit.NewService(db)}
}

// newAlertHandlers creates an AlertHandlers for testing.
func newAlertHandlers(db *gorm.DB) *handlers.AlertHandlers {
	return &handlers.AlertHandlers{DB: db, AlertNotes: repository.NewAlertNoteRepo(db), Audit: audit.NewService(db)}
}

// newSettingsHandlers creates a SettingsHandlers for testing.
func newSettingsHandlers(db *gorm.DB) *handlers.SettingsHandlers {
	return &handlers.SettingsHandlers{DB: db, Audit: audit.NewService(db)}
}

// --- Dashboard ---

func TestGetDashboardStats(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlers(db)
	db.Create(&models.Package{Name: "requests", Registry: "pypi"})
	db.Create(&models.Package{Name: "express", Registry: "npm"})
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

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)

	db.Create(&models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: models.ReleaseStatusCompleted})
	db.Create(&models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: models.ReleaseStatusPending})
	db.Create(&models.Release{PackageID: pkg.ID, Version: "1.2.0", Status: models.ReleaseStatusAnalyzing})

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
	db.Create(&models.Package{Name: "requests", Registry: "pypi"})
	db.Create(&models.Package{Name: "flask", Registry: "pypi", IsCustom: true})
	db.Create(&models.Package{Name: "express", Registry: "npm"})
	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"all packages", "", 3},
		{"filter pypi", "registry=pypi", 2},
		{"filter npm", "registry=npm", 1},
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
		db.Create(&models.Package{Name: fmt.Sprintf("pkg-%02d", i), Registry: "pypi"})
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

func TestListPackages_FilterCustom(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	db.Create(&models.Package{Name: "requests", Registry: "pypi", IsCustom: false})
	db.Create(&models.Package{Name: "my-pkg", Registry: "pypi", IsCustom: true})

	req := httptest.NewRequest(http.MethodGet, "/api/packages?is_custom=true", nil)
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
		{"valid pypi", `{"name":"django","registry":"pypi"}`, http.StatusCreated},
		{"valid npm", `{"name":"express","registry":"npm"}`, http.StatusCreated},
		{"missing name", `{"name":"","registry":"pypi"}`, http.StatusBadRequest},
		{"bad registry", `{"name":"test","registry":"rubygems"}`, http.StatusBadRequest},
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

	db.Create(&models.Package{Name: "requests", Registry: "pypi"})

	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(`{"name":"requests","registry":"pypi"}`))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreatePackage_SetsIsCustom(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(`{"name":"custom-pkg","registry":"npm"}`))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	var pkg models.Package
	db.Where("name = ?", "custom-pkg").First(&pkg)
	assert.True(t, pkg.IsCustom)
}

func TestGetPackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)
	pkg := models.Package{Name: "requests", Registry: "pypi"}
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

	pkg := models.Package{Name: "to-delete", Registry: "npm"}
	db.Create(&pkg)

	r := chi.NewRouter()
	r.Delete("/api/packages/{id}", h.DeletePackage)

	req := httptest.NewRequest(http.MethodDelete, "/api/packages/"+idStr(pkg.ID), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify soft-deleted
	var count int64
	db.Model(&models.Package{}).Where("id = ?", pkg.ID).Count(&count)
	assert.Equal(t, int64(0), count)
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

// --- Releases ---

func TestListPackageReleases(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	db.Create(&models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"})
	db.Create(&models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "pending"})

	// Other package releases should not be included
	pkg2 := models.Package{Name: "flask", Registry: "pypi"}
	db.Create(&pkg2)
	db.Create(&models.Release{PackageID: pkg2.ID, Version: "2.0.0", Status: "completed"})

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

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
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

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel1 := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)

	diff := models.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "test diff", FileChangesCount: 1}
	db.Create(&diff)
	analysis := models.Analysis{DiffID: diff.ID, Classification: "benign", Confidence: 0.95, Reasoning: "OK", ModelUsed: "test", AnalyzerType: "copilot"}
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

func TestListAlerts(t *testing.T) {
	db := setupTestDB(t)
	h := newAlertHandlers(db)

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel1 := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := models.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"}
	db.Create(&diff)
	analysis := models.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)

	db.Create(&models.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"})
	db.Create(&models.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "medium", Status: "acknowledged", Message: "Maybe bad"})

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
	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel1 := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := models.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "test diff", FileChangesCount: 1, LinesAdded: 5, LinesRemoved: 2}
	db.Create(&diff)
	analysis := models.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, Reasoning: "Bad code", ModelUsed: "test", AnalyzerType: "api"}
	db.Create(&analysis)
	alert := models.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Malicious detected"}
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
	db.Create(&models.Setting{Key: "pypi_poll_interval", Value: "5m"})
	db.Create(&models.Setting{Key: "npm_poll_interval", Value: "10m"})
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	h.GetSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "5m", data["pypi_poll_interval"])
	assert.Equal(t, "10m", data["npm_poll_interval"])
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

	body := `{"pypi_poll_interval":"10m","npm_poll_interval":"15m"}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify settings were saved
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "10m", data["pypi_poll_interval"])
	assert.Equal(t, "15m", data["npm_poll_interval"])
}

func TestUpdateSettings_Upsert(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)

	// Create initial
	db.Create(&models.Setting{Key: "pypi_poll_interval", Value: "5m"})

	// Update existing
	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"pypi_poll_interval":"30m"}`))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var setting models.Setting
	db.Where("key = ?", "pypi_poll_interval").First(&setting)
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

func TestUpdateSettings_VersionDepth(t *testing.T) {
	db := setupTestDB(t)
	h := newSettingsHandlers(db)

	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"version_depth_mode":"custom","version_depth_count":"3"}`))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var mode, count models.Setting
	db.Where("key = ?", "version_depth_mode").First(&mode)
	db.Where("key = ?", "version_depth_count").First(&count)
	assert.Equal(t, "custom", mode.Value)
	assert.Equal(t, "3", count.Value)
}

// --- Recent Releases ---

func TestGetRecentReleases(t *testing.T) {
	db := setupTestDB(t)
	h := newDashboardHandlers(db)

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	db.Create(&models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"})
	db.Create(&models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "pending"})

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
	assert.Equal(t, "pypi", first["packageRegistry"])
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
	h := &handlers.DashboardHandlers{DB: db, Dashboard: repository.NewDashboardRepo(db), Queue: nil}

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel1 := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)

	// Diff without analysis
	db.Create(&models.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"})

	// Queue is nil, so reanalyze returns error
	req := httptest.NewRequest(http.MethodPost, "/api/reanalyze", nil)
	w := httptest.NewRecorder()
	h.ReanalyzeAll(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReanalyzeAll_NoDiffs(t *testing.T) {
	db := setupTestDB(t)
	h := &handlers.DashboardHandlers{DB: db, Dashboard: repository.NewDashboardRepo(db), Queue: nil}

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
		db.Create(&models.Package{Name: fmt.Sprintf("pkg-%d", i), Registry: "pypi"})
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

	pkg1 := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg1)
	pkg2 := models.Package{Name: "express", Registry: "npm"}
	db.Create(&pkg2)

	// Set up required chain for alerts
	rel1 := models.Release{PackageID: pkg1.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	diff1 := models.Diff{ReleaseID: rel1.ID, DiffContent: "d"}
	db.Create(&diff1)
	analysis1 := models.Analysis{DiffID: diff1.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis1)

	rel2 := models.Release{PackageID: pkg2.ID, Version: "2.0.0", Status: "completed"}
	db.Create(&rel2)
	diff2 := models.Diff{ReleaseID: rel2.ID, DiffContent: "d"}
	db.Create(&diff2)
	analysis2 := models.Analysis{DiffID: diff2.ID, Classification: "suspicious", Confidence: 0.8, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis2)

	db.Create(&models.Alert{AnalysisID: analysis1.ID, PackageID: pkg1.ID, Severity: "critical", Status: "new", Message: "Malicious in requests"})
	db.Create(&models.Alert{AnalysisID: analysis2.ID, PackageID: pkg2.ID, Severity: "medium", Status: "new", Message: "Suspicious in express"})

	tests := []struct {
		name      string
		search    string
		wantCount int
		pgOnly    bool // ILIKE is PostgreSQL-specific, skip on SQLite
	}{
		{"search by package name", "requests", 1, true},
		{"search by message", "Suspicious", 1, true},
		{"search no results", "nonexistent", 0, true},
		{"empty search", "", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pgOnly {
				t.Skip("ILIKE requires PostgreSQL; search tested via integration tests")
			}
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

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := models.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := models.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := models.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"}
	db.Create(&alert)

	// Create some notes
	db.Create(&models.AlertNote{AlertID: alert.ID, OrgID: 0, UserID: 1, UserEmail: "user@example.com", Content: "First note"})
	db.Create(&models.AlertNote{AlertID: alert.ID, OrgID: 0, UserID: 2, UserEmail: "other@example.com", Content: "Second note"})

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

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := models.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := models.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := models.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"}
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

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)
	rel := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := models.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := models.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := models.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "Bad"}
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

	body := `{"packages":[{"name":"django","registry":"pypi"},{"name":"express","registry":"npm"}]}`
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
	db.Model(&models.Package{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestImportPackages_Duplicates(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	// Pre-create a package
	db.Create(&models.Package{Name: "django", Registry: "pypi"})

	body := `{"packages":[{"name":"django","registry":"pypi"},{"name":"flask","registry":"pypi"}]}`
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

func TestImportPackages_InvalidRegistry(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"packages":[{"name":"test","registry":"rubygems"}]}`
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
		entries[i] = fmt.Sprintf(`{"name":"pkg-%d","registry":"pypi"}`, i)
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

	content := "pypi:requests\nnpm:express\npypi:flask\n# comment\n\nnpm:lodash"
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

	body := `{"packages":[{"name":"valid-pkg","registry":"pypi"},{"name":"evil pkg!","registry":"pypi"},{"name":"ok.pkg","registry":"npm"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages/bulk-import", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportPackages(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["imported"])  // valid-pkg and ok.pkg
	errors := data["errors"].([]any)
	assert.Len(t, errors, 1) // evil pkg! has invalid characters
}

func TestCreatePackage_InvalidName(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"name":"invalid name!@#$","registry":"pypi"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePackage_NameTooLong(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	longName := strings.Repeat("a", 201)
	body := fmt.Sprintf(`{"name":"%s","registry":"pypi"}`, longName)
	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePackage_ScopedNpmPackage(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db)

	body := `{"name":"@angular/core","registry":"npm"}`
	req := httptest.NewRequest(http.MethodPost, "/api/packages", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.CreatePackage(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

// --- Re-analyze Release ---

func TestReanalyzeRelease_NoQueue(t *testing.T) {
	db := setupTestDB(t)
	h := newPackageHandlers(db) // Queue is nil

	pkg := models.Package{Name: "requests", Registry: "pypi", OrgID: 0}
	db.Create(&pkg)
	rel := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)

	r := chi.NewRouter()
	r.Post("/api/releases/{id}/reanalyze", h.ReanalyzeRelease)

	req := httptest.NewRequest(http.MethodPost, "/api/releases/"+idStr(rel.ID)+"/reanalyze", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReanalyzeRelease_NotFound(t *testing.T) {
	t.Skip("Cannot test release-not-found path without a queue mock; handler returns 500 (queue nil) before checking release existence")
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

	pkg := models.Package{Name: "requests", Registry: "pypi"}
	db.Create(&pkg)

	// Baseline release (completed, no diff)
	rel1 := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)

	// Analyzed release
	rel2 := models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := models.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"}
	db.Create(&diff)
	db.Create(&models.Analysis{DiffID: diff.ID, Classification: "benign", Confidence: 0.95, Reasoning: "OK", ModelUsed: "test", AnalyzerType: "copilot"})

	// Pending release (no diff, not completed)
	rel3 := models.Release{PackageID: pkg.ID, Version: "1.2.0", Status: "pending"}
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

	pkg := models.Package{Name: "brand-new", Registry: "pypi"}
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
