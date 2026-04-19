package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/pkg/queue"
)

// Pipeline processes diffs through the LLM analyzer and creates alerts.
type Pipeline struct {
	db        *gorm.DB
	analyzers []Analyzer
	notifier  usecase.NotificationDispatcher
}

// NewPipeline creates a new analysis pipeline.
func NewPipeline(db *gorm.DB, notifier usecase.NotificationDispatcher, analyzers ...Analyzer) *Pipeline {
	return &Pipeline{
		db:        db,
		analyzers: analyzers,
		notifier:  notifier,
	}
}

// ProcessJob is the queue worker handler for analyze jobs.
func (p *Pipeline) ProcessJob(ctx context.Context, job *queue.Job) error {
	return p.processDiff(ctx, job.ReferenceID)
}

// processDiff runs all configured analyzers on a diff.
func (p *Pipeline) processDiff(ctx context.Context, diffID uint) error {
	var diff persistent.Diff
	if err := p.db.Preload("Release.Package").First(&diff, diffID).Error; err != nil {
		return fmt.Errorf("loading diff %d: %w", diffID, err)
	}

	// Find previous release version
	var prevRelease persistent.Release
	if err := p.db.First(&prevRelease, diff.PrevReleaseID).Error; err != nil {
		return fmt.Errorf("loading previous release: %w", err)
	}

	pkg := diff.Release.Package

	var analysisCreated bool
	var lastErr error

	for _, analyzer := range p.analyzers {
		result, err := analyzer.Analyze(
			ctx,
			diff.DiffContent,
			pkg.Name,
			string(pkg.Ecosystem),
			prevRelease.Version,
			diff.Release.Version,
		)
		if err != nil {
			slog.Error("analyzer failed", "type", analyzer.Type(), "package", pkg.Name, "error", err)
			lastErr = err
			continue
		}

		// Store analysis
		analysis := persistent.Analysis{
			DiffID:         diff.ID,
			Classification: persistent.Classification(result.Classification),
			Confidence:     result.Confidence,
			Reasoning:      result.Reasoning,
			ModelUsed:      "claude",
			AnalyzerType:   persistent.AnalyzerType(analyzer.Type()),
			RawResponse:    result.RawResponse,
		}

		if err := p.db.Create(&analysis).Error; err != nil {
			slog.Error("failed to save analysis", "error", err)
			lastErr = err
			continue
		}

		analysisCreated = true

		slog.Info("analysis complete",
			"package", pkg.Name,
			"version", diff.Release.Version,
			"classification", result.Classification,
			"confidence", result.Confidence,
			"analyzer", analyzer.Type(),
		)

		// Create alert if suspicious or malicious
		if result.Classification == "suspicious" || result.Classification == "malicious" {
			severity := persistent.AlertSeverityMedium
			if result.Classification == "malicious" {
				severity = persistent.AlertSeverityCritical
			}

			alert := persistent.Alert{
				WorkspaceID:      pkg.WorkspaceID,
				AnalysisID: analysis.ID,
				PackageID:  pkg.ID,
				Severity:   severity,
				Status:     persistent.AlertStatusNew,
				Message:    fmt.Sprintf("Package %s v%s classified as %s (confidence: %.0f%%): %s", pkg.Name, diff.Release.Version, result.Classification, result.Confidence*100, result.Reasoning),
			}

			if err := p.db.Create(&alert).Error; err != nil {
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
						Title:         fmt.Sprintf("%s package detected: %s v%s", strings.ToUpper(result.Classification[:1])+result.Classification[1:], pkg.Name, diff.Release.Version),
						Message:       fmt.Sprintf("Package %s v%s (%s) classified as %s with %.0f%% confidence. %s", pkg.Name, diff.Release.Version, pkg.Ecosystem, result.Classification, result.Confidence*100, result.Reasoning),
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
				Title:         fmt.Sprintf("Analysis failed: %s v%s", pkg.Name, diff.Release.Version),
				Message:       fmt.Sprintf("All analyzers failed for %s v%s (%s). The release will be retried. Error: %v", pkg.Name, diff.Release.Version, pkg.Ecosystem, lastErr),
				ReferenceID:   diff.Release.ID,
				ReferenceType: "release",
			})
		}
		return fmt.Errorf("all analyzers failed for diff %d: %w", diffID, lastErr)
	}

	// Update release status to completed
	p.db.Model(&diff.Release).Update("status", persistent.ReleaseStatusCompleted)

	return nil
}
