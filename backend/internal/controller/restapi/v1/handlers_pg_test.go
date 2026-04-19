//go:build integration

package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	alertnoteuc "github.com/veilence/veilence-mx/backend/internal/usecase/alertnote"
	v1 "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
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

	h := &v1.AlertHandlers{
		DB:    db,
		Notes: alertnoteuc.New(persistent.NewAlertNoteRepo(db), persistent.NewAlertRepo(db), persistent.NewUserRepo(db)),
		Audit: audit.NewService(db),
	}

	// Create test data with mixed case
	pkg1 := persistent.Package{Name: "Requests", Ecosystem: "python", WorkspaceID: 0}
	db.Create(&pkg1)
	pkg2 := persistent.Package{Name: "EXPRESS", Ecosystem: "npm", WorkspaceID: 0}
	db.Create(&pkg2)

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

	db.Create(&persistent.Alert{AnalysisID: analysis1.ID, PackageID: pkg1.ID, WorkspaceID: 0, Severity: "critical", Status: "new", Message: "Malicious code detected"})
	db.Create(&persistent.Alert{AnalysisID: analysis2.ID, PackageID: pkg2.ID, WorkspaceID: 0, Severity: "medium", Status: "new", Message: "Suspicious behavior found"})

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

	h := &v1.PackageHandlers{
		DB:    db,
		Audit: audit.NewService(db),
	}

	db.Create(&persistent.Package{Name: "Django", Ecosystem: "python", WorkspaceID: 0})
	db.Create(&persistent.Package{Name: "FLASK", Ecosystem: "python", WorkspaceID: 0})
	db.Create(&persistent.Package{Name: "express", Ecosystem: "npm", WorkspaceID: 0})

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

	h := &v1.DashboardHandlers{
		DB:        db,
		Dashboard: persistent.NewDashboardRepo(db),
	}

	pkg1 := persistent.Package{Name: "Requests", Ecosystem: "python", WorkspaceID: 0}
	db.Create(&pkg1)
	pkg2 := persistent.Package{Name: "LODASH", Ecosystem: "npm", WorkspaceID: 0}
	db.Create(&pkg2)

	db.Create(&persistent.Release{PackageID: pkg1.ID, Version: "1.0.0", Status: "completed"})
	db.Create(&persistent.Release{PackageID: pkg2.ID, Version: "4.17.0", Status: "completed"})

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
