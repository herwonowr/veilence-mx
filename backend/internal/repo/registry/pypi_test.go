package registry_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veilence/veilence-mx/backend/internal/repo/registry"
)

func TestPyPIClient_GetPackage(t *testing.T) {
	tests := []struct {
		name           string
		packageName    string
		responseStatus int
		responseBody   any
		wantErr        bool
		wantVersion    string
	}{
		{
			name:           "successful fetch",
			packageName:    "requests",
			responseStatus: http.StatusOK,
			responseBody: map[string]any{
				"info": map[string]any{
					"name":    "requests",
					"version": "2.31.0",
					"summary": "HTTP library",
				},
				"releases": map[string]any{
					"2.31.0": []map[string]any{
						{
							"url":                  "https://example.com/requests-2.31.0.tar.gz",
							"packagetype":          "sdist",
							"digests":              map[string]string{"sha256": "abc123"},
							"upload_time_iso_8601": "2023-05-22T15:00:00Z",
						},
					},
				},
			},
			wantVersion: "2.31.0",
		},
		{
			name:           "package not found",
			packageName:    "nonexistent",
			responseStatus: http.StatusNotFound,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer server.Close()

			client := registry.NewPyPIClient(registry.WithPyPIBaseURL(server.URL))
			info, err := client.GetPackage(context.Background(), tt.packageName)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, info.Version)
			assert.NotEmpty(t, info.Versions)
		})
	}
}

func TestPyPIClient_GetTopPackages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"rows": []map[string]any{
				{"project": "requests", "download_count": 1000000},
				{"project": "boto3", "download_count": 800000},
				{"project": "flask", "download_count": 600000},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := registry.NewPyPIClient(registry.WithPyPITopURL(server.URL))
	rankings, err := client.GetTopPackages(context.Background(), 3)
	require.NoError(t, err)
	assert.Len(t, rankings, 3)
	assert.Equal(t, "requests", rankings[0].Name)
	assert.Equal(t, int64(1000000), rankings[0].DownloadCount)
	assert.Equal(t, uint(1), rankings[0].Rank)
	assert.Equal(t, uint(3), rankings[2].Rank)
}

func TestPyPIClient_Name(t *testing.T) {
	client := registry.NewPyPIClient()
	assert.Equal(t, "python", client.Name())
}

func TestPyPIClient_DownloadTarball(t *testing.T) {
	tarballContent := []byte("fake tarball content for testing")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(tarballContent)
	}))
	defer server.Close()

	client := registry.NewPyPIClient(registry.WithPyPIBaseURL(server.URL))
	path, err := client.DownloadTarball(context.Background(), server.URL+"/packages/test-1.0.tar.gz")
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Dir(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, tarballContent, data)
}

func TestPyPIClient_DownloadTarball_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := registry.NewPyPIClient(registry.WithPyPIBaseURL(server.URL))
	_, err := client.DownloadTarball(context.Background(), server.URL+"/packages/test.tar.gz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status")
}

func TestPyPIClient_GetPackage_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := registry.NewPyPIClient(registry.WithPyPIBaseURL(server.URL))
	_, err := client.GetPackage(context.Background(), "test")
	assert.Error(t, err)
}

func TestPyPIClient_GetTopPackages_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := registry.NewPyPIClient(registry.WithPyPITopURL(server.URL))
	_, err := client.GetTopPackages(context.Background(), 10)
	assert.Error(t, err)
}

func TestPyPIClient_GetTopPackages_Limit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"rows": []map[string]any{
				{"project": "a", "download_count": 500},
				{"project": "b", "download_count": 400},
				{"project": "c", "download_count": 300},
				{"project": "d", "download_count": 200},
				{"project": "e", "download_count": 100},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := registry.NewPyPIClient(registry.WithPyPITopURL(server.URL))
	rankings, err := client.GetTopPackages(context.Background(), 3)
	require.NoError(t, err)
	assert.Len(t, rankings, 3)
}
