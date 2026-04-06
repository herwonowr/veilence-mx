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
	return &handlers.DashboardHandlers{DB: db}
}

// newPackageHandlers creates a PackageHandlers for testing.
func newPackageHandlers(db *gorm.DB) *handlers.PackageHandlers {
	return &handlers.PackageHandlers{DB: db, Audit: audit.NewService(db)}
}

// newAlertHandlers creates an AlertHandlers for testing.
func newAlertHandlers(db *gorm.DB) *handlers.AlertHandlers {
	return &handlers.AlertHandlers{DB: db, Audit: audit.NewService(db)}
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
	h := &handlers.DashboardHandlers{DB: db, Queue: nil}

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
	h := &handlers.DashboardHandlers{DB: db, Queue: nil}

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
