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
	"github.com/veilence/veilence-mx/backend/internal/registry"
)

func TestNPMClient_GetPackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"name":        "express",
			"description": "Fast web framework",
			"dist-tags":   map[string]string{"latest": "4.18.2"},
			"time":        map[string]string{"4.18.2": "2023-03-01T00:00:00Z"},
			"versions": map[string]any{
				"4.18.2": map[string]any{
					"version": "4.18.2",
					"dist": map[string]any{
						"tarball": "https://registry.npmjs.org/express/-/express-4.18.2.tgz",
						"shasum":  "abc123",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	info, err := client.GetPackage(context.Background(), "express")
	require.NoError(t, err)
	assert.Equal(t, "4.18.2", info.Version)
	assert.NotEmpty(t, info.Versions)
}

func TestNPMClient_GetPackage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	_, err := client.GetPackage(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestNPMClient_GetTopPackages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"objects": []map[string]any{
				{"package": map[string]string{"name": "lodash"}},
				{"package": map[string]string{"name": "react"}},
				{"package": map[string]string{"name": "express"}},
			},
			"total": 3,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	names, err := client.GetTopPackages(context.Background(), 3)
	require.NoError(t, err)
	assert.Len(t, names, 3)
	assert.Equal(t, "lodash", names[0])
}

func TestNPMClient_Name(t *testing.T) {
	client := registry.NewNPMClient()
	assert.Equal(t, "npm", client.Name())
}

func TestNPMClient_DownloadTarball(t *testing.T) {
	tarballContent := []byte("fake npm tarball content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(tarballContent)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	path, err := client.DownloadTarball(context.Background(), server.URL+"/express/-/express-4.18.2.tgz")
	require.NoError(t, err)
	defer os.RemoveAll(filepath.Dir(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, tarballContent, data)
}

func TestNPMClient_DownloadTarball_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	_, err := client.DownloadTarball(context.Background(), server.URL+"/test.tgz")
	assert.Error(t, err)
}

func TestNPMClient_GetPackage_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	_, err := client.GetPackage(context.Background(), "test")
	assert.Error(t, err)
}

func TestNPMClient_GetTopPackages_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	_, err := client.GetTopPackages(context.Background(), 10)
	assert.Error(t, err)
}

func TestNPMClient_GetPackage_ScopedPackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Scoped packages should be encoded correctly in URL
		assert.Contains(t, r.URL.Path, "@angular")

		resp := map[string]any{
			"name":        "@angular/core",
			"description": "Angular core",
			"dist-tags":   map[string]string{"latest": "17.0.0"},
			"time":        map[string]string{"17.0.0": "2023-11-01T00:00:00Z"},
			"versions": map[string]any{
				"17.0.0": map[string]any{
					"version": "17.0.0",
					"dist": map[string]any{
						"tarball": "https://registry.npmjs.org/@angular/core/-/core-17.0.0.tgz",
						"shasum":  "def456",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := registry.NewNPMClient(registry.WithNPMBaseURL(server.URL))
	info, err := client.GetPackage(context.Background(), "@angular/core")
	require.NoError(t, err)
	assert.Equal(t, "17.0.0", info.Version)
}
