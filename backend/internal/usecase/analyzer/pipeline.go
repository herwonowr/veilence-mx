package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/pkg/queue"
)

// Pipeline processes diffs through the LLM analyzer and creates alerts.
type Pipeline struct {
	repo      PipelineRepository
	analyzers []Analyzer
	notifier  usecase.NotificationDispatcher
}

// NewPipeline creates a new analysis pipeline.
func NewPipeline(repo PipelineRepository, notifier usecase.NotificationDispatcher, analyzers ...Analyzer) *Pipeline {
	return &Pipeline{
		repo:      repo,
		analyzers: analyzers,
		notifier:  notifier,
	}
}

// ProcessJob is the queue worker handler for analyze jobs.
func (p *Pipeline) ProcessJob(ctx context.Context, job *queue.Job) error {
	return p.processDiff(ctx, job.ReferenceID)
}

// processDiff runs all configured analyzers on a diff.
func (p *Pipeline) processDiff(ctx context.Context, diffID string) error {
	diff, release, pkg, err := p.repo.FindDiffWithRelease(ctx, diffID)
	if err != nil {
		return fmt.Errorf("loading diff %s: %w", diffID, err)
	}

	// Find previous release version
	prevRelease, err := p.repo.FindReleaseByID(ctx, diff.PrevReleaseID)
	if err != nil {
		return fmt.Errorf("loading previous release: %w", err)
	}

	var analysisCreated bool
	var lastErr error

	// Truncate diff content to prevent excessive LLM token usage (Finding 13)
	const maxDiffSize = 100 * 1024 // 100KB
	diffContent := diff.DiffContent
	if len(diffContent) > maxDiffSize {
		diffContent = diffContent[:maxDiffSize] + "\n... [truncated]"
		slog.Warn("diff content truncated for analysis",
			"diff_id", diffID,
			"original_size", len(diff.DiffContent),
			"truncated_size", maxDiffSize,
		)
	}

	for _, analyzer := range p.analyzers {
		result, err := analyzer.Analyze(
			ctx,
			diffContent,
			pkg.Name,
			string(pkg.Ecosystem),
			prevRelease.Version,
			release.Version,
		)
		if err != nil {
			slog.Error("analyzer failed", "type", analyzer.Type(), "package", pkg.Name, "error", err)
			lastErr = err
			continue
		}

		// Store analysis
		analysis := &entity.Analysis{
			DiffID:         diff.ID,
			Classification: entity.Classification(result.Classification),
			Confidence:     result.Confidence,
			Reasoning:      result.Reasoning,
			ModelUsed:      "claude",
			AnalyzerType:   entity.AnalyzerType(analyzer.Type()),
			RawResponse:    result.RawResponse,
		}

		if err := p.repo.CreateAnalysis(ctx, analysis); err != nil {
			slog.Error("failed to save analysis", "error", err)
			lastErr = err
			continue
		}

		analysisCreated = true

		slog.Info("analysis complete",
			"package", pkg.Name,
			"version", release.Version,
			"classification", result.Classification,
			"confidence", result.Confidence,
			"analyzer", analyzer.Type(),
		)

		// Create alert if suspicious or malicious
		if result.Classification == "suspicious" || result.Classification == "malicious" {
			severity := entity.AlertSeverityMedium
			if result.Classification == "malicious" {
				severity = entity.AlertSeverityCritical
			}

			alert := &entity.Alert{
				WorkspaceID: pkg.WorkspaceID,
				AnalysisID:  analysis.ID,
				PackageID:   pkg.ID,
				Severity:    severity,
				Status:      entity.AlertStatusNew,
				Message:     fmt.Sprintf("Package %s v%s classified as %s (confidence: %.0f%%): %s", pkg.Name, release.Version, result.Classification, result.Confidence*100, result.Reasoning),
			}

			if err := p.repo.CreateAlert(ctx, alert); err != nil {
				slog.Error("failed to create alert", "error", err)
			} else {
				slog.Warn("alert created",
					"package", pkg.Name,
					"severity", severity,
					"classification", result.Classification,
				)
				// Dispatch notification for malicious/suspicious alert
				if p.notifier != nil {
					notifSeverity := "high"
					notifEventType := entity.NotifEventAlertSuspicious
					if result.Classification == "malicious" {
						notifSeverity = "critical"
						notifEventType = entity.NotifEventAlertMalicious
					}
					p.notifier.DispatchEvent(ctx, pkg.WorkspaceID, entity.NotificationEvent{
						Severity:      notifSeverity,
						EventType:     notifEventType,
						Title:         fmt.Sprintf("%s package detected: %s v%s", strings.ToUpper(result.Classification[:1])+result.Classification[1:], pkg.Name, release.Version),
						Message:       fmt.Sprintf("Package %s v%s (%s) classified as %s with %.0f%% confidence. %s", pkg.Name, release.Version, pkg.Ecosystem, result.Classification, result.Confidence*100, result.Reasoning),
						ReferenceID:   alert.ID,
						ReferenceType: "alert",
					})
				}
			}
		}
	}

	// If no analysis was created, return error so the job is retried
	if !analysisCreated {
		if p.notifier != nil {
			p.notifier.DispatchEvent(ctx, pkg.WorkspaceID, entity.NotificationEvent{
				Severity:      "high",
				EventType:     entity.NotifEventAnalysisError,
				Title:         fmt.Sprintf("Analysis failed: %s v%s", pkg.Name, release.Version),
				Message:       fmt.Sprintf("All analyzers failed for %s v%s (%s). The release will be retried. Error: %v", pkg.Name, release.Version, pkg.Ecosystem, lastErr),
				ReferenceID:   release.ID,
				ReferenceType: "release",
			})
		}
		return fmt.Errorf("all analyzers failed for diff %s: %w", diffID, lastErr)
	}

	// Update release status to completed
	if err := p.repo.UpdateReleaseStatus(ctx, release.ID, entity.ReleaseStatusCompleted); err != nil {
		slog.Error("failed to update release status", "release_id", release.ID, "error", err)
	}

	return nil
}
