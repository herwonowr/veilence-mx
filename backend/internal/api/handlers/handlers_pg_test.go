//go:build integration

package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/api/handlers"
	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"
	"github.com/veilence/veilence-mx/backend/internal/testutil"
)

// =====================================================================
// PostgreSQL Integration Tests — Search functionality
//
// These tests verify case-insensitive search against a real PostgreSQL
// database. Run with:
//
//   make db-up
//   psql -h localhost -U veilence -c "CREATE DATABASE veilence_mx_test;" veilence_mx
//   make test-backend-integration
//
// Or with a custom DSN:
//   TEST_DATABASE_URL="postgres://..." go test -tags=integration ./... -v
// =====================================================================

func TestIntegrationPG_AlertSearch_CaseInsensitive(t *testing.T) {
	db := testutil.SetupPostgresDB(t)

	h := &handlers.AlertHandlers{
		DB:         db,
		AlertNotes: repository.NewAlertNoteRepo(db),
		Audit:      audit.NewService(db),
	}

	// Create test data with mixed case
	pkg1 := models.Package{Name: "Requests", Registry: "pypi", OrgID: 0}
	db.Create(&pkg1)
	pkg2 := models.Package{Name: "EXPRESS", Registry: "npm", OrgID: 0}
	db.Create(&pkg2)

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

	db.Create(&models.Alert{AnalysisID: analysis1.ID, PackageID: pkg1.ID, OrgID: 0, Severity: "critical", Status: "new", Message: "Malicious code detected"})
	db.Create(&models.Alert{AnalysisID: analysis2.ID, PackageID: pkg2.ID, OrgID: 0, Severity: "medium", Status: "new", Message: "Suspicious behavior found"})

	tests := []struct {
		name      string
		search    string
		wantCount int
	}{
		{"lowercase search for uppercase package", "requests", 1},
		{"uppercase search for uppercase package", "EXPRESS", 1},
		{"mixed case search", "ReQuEsTs", 1},
		{"search by message lowercase", "malicious", 1},
		{"search by message mixed case", "Suspicious", 1},
		{"partial match", "equ", 1},     // matches "Requests"
		{"no match", "nonexistent", 0},
		{"empty search returns all", "", 2},
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
			assert.Len(t, data, tt.wantCount, "search=%q", tt.search)
		})
	}
}

func TestIntegrationPG_PackageSearch_CaseInsensitive(t *testing.T) {
	db := testutil.SetupPostgresDB(t)

	h := &handlers.PackageHandlers{
		DB:    db,
		Audit: audit.NewService(db),
	}

	db.Create(&models.Package{Name: "Django", Registry: "pypi", OrgID: 0})
	db.Create(&models.Package{Name: "FLASK", Registry: "pypi", OrgID: 0})
	db.Create(&models.Package{Name: "express", Registry: "npm", OrgID: 0})

	tests := []struct {
		name      string
		search    string
		wantCount int
	}{
		{"lowercase finds capitalized", "django", 1},
		{"uppercase finds lowercase", "EXPRESS", 1},
		{"mixed case", "fLaSk", 1},
		{"partial match", "ang", 1}, // matches "Django"
		{"no match", "nonexistent", 0},
		{"empty returns all", "", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := ""
			if tt.search != "" {
				query = "search=" + tt.search
			}
			req := httptest.NewRequest(http.MethodGet, "/api/packages?"+query, nil)
			w := httptest.NewRecorder()
			h.ListPackages(w, req)
			require.Equal(t, http.StatusOK, w.Code)

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			data := resp["data"].([]any)
			assert.Len(t, data, tt.wantCount, "search=%q", tt.search)
		})
	}
}

func TestIntegrationPG_ReleaseSearch_CaseInsensitive(t *testing.T) {
	db := testutil.SetupPostgresDB(t)

	h := &handlers.DashboardHandlers{
		DB:        db,
		Dashboard: repository.NewDashboardRepo(db),
	}

	pkg1 := models.Package{Name: "Requests", Registry: "pypi", OrgID: 0}
	db.Create(&pkg1)
	pkg2 := models.Package{Name: "LODASH", Registry: "npm", OrgID: 0}
	db.Create(&pkg2)

	db.Create(&models.Release{PackageID: pkg1.ID, Version: "1.0.0", Status: "completed"})
	db.Create(&models.Release{PackageID: pkg2.ID, Version: "4.17.0", Status: "completed"})

	tests := []struct {
		name      string
		search    string
		wantCount int
	}{
		{"lowercase finds capitalized", "requests", 1},
		{"uppercase finds uppercase", "LODASH", 1},
		{"mixed case", "lOdAsH", 1},
		{"no match", "nonexistent", 0},
		{"empty returns all", "", 2},
	}

	r := chi.NewRouter()
	r.Get("/api/dashboard/recent", h.GetRecentReleases)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := ""
			if tt.search != "" {
				query = "search=" + tt.search
			}
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/recent?"+query, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusOK, w.Code)

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)
			data := resp["data"].([]any)
			assert.Len(t, data, tt.wantCount, "search=%q", tt.search)
		})
	}
}
