package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// Pipeline processes diffs through the LLM providers and creates alerts.
type Pipeline struct {
	repo      PipelineRepository
	providers []LLMProvider
	notifier  usecase.NotificationDispatcher
}

// NewPipeline creates a new analysis pipeline.
func NewPipeline(repo PipelineRepository, notifier usecase.NotificationDispatcher, providers ...LLMProvider) *Pipeline {
	return &Pipeline{
		repo:      repo,
		providers: providers,
		notifier:  notifier,
	}
}

// ProcessDiff runs all configured LLM providers on a diff.
func (p *Pipeline) ProcessDiff(ctx context.Context, diffID string) error {
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

	// Safety truncation in case diff content exceeds limit (should already be
	// truncated by the differ, but guard against edge cases).
	const maxDiffSize = 100 * 1024 // 100KB
	diffContent := diff.DiffContent
	diffTruncated := diff.Truncated
	if len(diffContent) > maxDiffSize {
		diffContent = diffContent[:maxDiffSize] + "\n\n--- DIFF TRUNCATED (exceeded 100KB limit) ---\n"
		diffTruncated = true
		slog.Warn("diff content truncated for analysis",
			"diff_id", diffID,
			"original_size", len(diff.DiffContent),
			"truncated_size", maxDiffSize,
		)
	}

	// Prepend truncation warning so the LLM knows the diff is incomplete.
	// This is critical for security - the LLM must factor in that malicious
	// changes may exist beyond the truncation boundary.
	if diffTruncated {
		truncationWarning := fmt.Sprintf(
			"WARNING: This diff has been truncated from %d bytes to %d bytes. "+
				"You are NOT seeing the complete set of changes. "+
				"Malicious code may exist beyond the truncation boundary. "+
				"Factor this into your confidence score and flag if the visible "+
				"changes suggest further review of the full diff is warranted.\n\n",
			diff.OriginalSize, len(diffContent),
		)
		diffContent = truncationWarning + diffContent
	}

	for _, provider := range p.providers {
		result, err := provider.Analyze(
			ctx,
			diffContent,
			pkg.Name,
			string(pkg.Ecosystem),
			prevRelease.Version,
			release.Version,
			diffTruncated,
		)
		if err != nil {
			slog.Error("provider failed", "type", provider.Type(), "package", pkg.Name, "error", err)
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
			AnalyzerType:   entity.AnalyzerType(provider.Type()),
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
			"provider", provider.Type(),
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
				ReleaseID:   release.ID,
				PackageID:   pkg.ID,
				Severity:    severity,
				Status:      entity.AlertStatusNew,
				Message:     fmt.Sprintf("Package %s %s classified as %s (confidence: %.0f%%): %s", pkg.Name, entity.FormatVersion(release.Version), result.Classification, result.Confidence*100, result.Reasoning),
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
						Title:         fmt.Sprintf("%s package detected: %s %s", strings.ToUpper(result.Classification[:1])+result.Classification[1:], pkg.Name, entity.FormatVersion(release.Version)),
						Message:       fmt.Sprintf("Package %s %s (%s) classified as %s with %.0f%% confidence. %s", pkg.Name, entity.FormatVersion(release.Version), pkg.Ecosystem, result.Classification, result.Confidence*100, result.Reasoning),
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
				Title:         fmt.Sprintf("Analysis failed: %s %s", pkg.Name, entity.FormatVersion(release.Version)),
				Message:       fmt.Sprintf("All providers failed for %s %s (%s). The release will be retried. Error: %v", pkg.Name, entity.FormatVersion(release.Version), pkg.Ecosystem, lastErr),
				ReferenceID:   release.ID,
				ReferenceType: "release",
			})
		}
		return fmt.Errorf("all providers failed for diff %s: %w", diffID, lastErr)
	}

	// Update release status to completed
	if err := p.repo.UpdateReleaseStatus(ctx, release.ID, entity.ReleaseStatusCompleted); err != nil {
		slog.Error("failed to update release status", "release_id", release.ID, "error", err)
	}

	return nil
}
