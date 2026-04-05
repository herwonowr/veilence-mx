package analyzer

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/queue"
)

// Pipeline processes diffs through the LLM analyzer and creates alerts.
type Pipeline struct {
	db        *gorm.DB
	analyzers []Analyzer
}

// NewPipeline creates a new analysis pipeline.
func NewPipeline(db *gorm.DB, analyzers ...Analyzer) *Pipeline {
	return &Pipeline{
		db:        db,
		analyzers: analyzers,
	}
}

// ProcessJob is the queue worker handler for analyze jobs.
func (p *Pipeline) ProcessJob(ctx context.Context, job *queue.Job) error {
	return p.processDiff(ctx, job.ReferenceID)
}

// processDiff runs all configured analyzers on a diff.
func (p *Pipeline) processDiff(ctx context.Context, diffID uint) error {
	var diff models.Diff
	if err := p.db.Preload("Release.Package").First(&diff, diffID).Error; err != nil {
		return fmt.Errorf("loading diff %d: %w", diffID, err)
	}

	// Find previous release version
	var prevRelease models.Release
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
			string(pkg.Registry),
			prevRelease.Version,
			diff.Release.Version,
		)
		if err != nil {
			slog.Error("analyzer failed", "type", analyzer.Type(), "package", pkg.Name, "error", err)
			lastErr = err
			continue
		}

		// Store analysis
		analysis := models.Analysis{
			DiffID:         diff.ID,
			Classification: models.Classification(result.Classification),
			Confidence:     result.Confidence,
			Reasoning:      result.Reasoning,
			ModelUsed:      "claude",
			AnalyzerType:   models.AnalyzerType(analyzer.Type()),
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
			severity := models.AlertSeverityMedium
			if result.Classification == "malicious" {
				severity = models.AlertSeverityCritical
			}

			alert := models.Alert{
				OrgID:      pkg.OrgID,
				AnalysisID: analysis.ID,
				PackageID:  pkg.ID,
				Severity:   severity,
				Status:     models.AlertStatusNew,
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
			}
		}
	}

	// If no analysis was created, return error so the job is retried
	if !analysisCreated {
		return fmt.Errorf("all analyzers failed for diff %d: %w", diffID, lastErr)
	}

	// Update release status to completed
	p.db.Model(&diff.Release).Update("status", models.ReleaseStatusCompleted)

	return nil
}
