// Package releaseuc implements the business logic for release management.
package releaseuc

import (
	"context"
	"errors"
	"fmt"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

const (
	jobTypeDiff    = "diff"
	jobTypeAnalyze = "analyze"
)

// UseCase implements usecase.ReleaseService.
type UseCase struct {
	packages usecase.PackageRepository
	releases usecase.ReleaseRepository
	diffs    usecase.DiffRepository
	analyses usecase.AnalysisRepository
	queue    usecase.QueueEnqueuer
}

// New creates a new release UseCase.
func New(
	packages usecase.PackageRepository,
	releases usecase.ReleaseRepository,
	diffs usecase.DiffRepository,
	analyses usecase.AnalysisRepository,
	queue usecase.QueueEnqueuer,
) *UseCase {
	return &UseCase{
		packages: packages,
		releases: releases,
		diffs:    diffs,
		analyses: analyses,
		queue:    queue,
	}
}

// ListByPackage returns a paginated list of releases for a package, scoped to an org.
func (uc *UseCase) ListByPackage(ctx context.Context, orgID, packageID uint, page, limit int) ([]entity.Release, int64, error) {
	// Verify package belongs to the requesting org
	pkg, err := uc.packages.FindByID(ctx, packageID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, 0, entity.ErrNotFound
		}
		return nil, 0, fmt.Errorf("ReleaseUseCase.ListByPackage: verifying package: %w", err)
	}
	if pkg.OrgID != orgID {
		return nil, 0, entity.ErrNotFound
	}

	releases, total, err := uc.releases.FindByPackageID(ctx, packageID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("ReleaseUseCase.ListByPackage: %w", err)
	}
	return releases, total, nil
}

// GetRelease returns a single release with its diff, analysis, and package info.
func (uc *UseCase) GetRelease(ctx context.Context, orgID, releaseID uint) (*entity.ReleaseDetail, error) {
	release, pkg, err := uc.releases.FindByIDWithPackage(ctx, releaseID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("ReleaseUseCase.GetRelease: %w", err)
	}
	if pkg.OrgID != orgID {
		return nil, entity.ErrNotFound
	}

	detail := &entity.ReleaseDetail{
		Release: *release,
		Package: pkg,
	}

	diff, err := uc.diffs.FindFirstByReleaseID(ctx, releaseID)
	if err == nil && diff != nil {
		detail.Diff = diff
		analyses, err := uc.analyses.FindByDiffID(ctx, diff.ID)
		if err == nil && len(analyses) > 0 {
			detail.Analysis = &analyses[0]
		}
	} else if release.Status == entity.ReleaseStatusCompleted {
		detail.IsBaseline = true
	}

	return detail, nil
}

// ReanalyzeRelease re-queues a single release for analysis. Returns the message and job ID.
func (uc *UseCase) ReanalyzeRelease(ctx context.Context, orgID, releaseID uint) (string, string, error) {
	if uc.queue == nil {
		return "", "", fmt.Errorf("ReleaseUseCase.ReanalyzeRelease: queue not configured")
	}

	release, pkg, err := uc.releases.FindByIDWithPackage(ctx, releaseID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return "", "", entity.ErrNotFound
		}
		return "", "", fmt.Errorf("ReleaseUseCase.ReanalyzeRelease: %w", err)
	}
	if pkg.OrgID != orgID {
		return "", "", entity.ErrNotFound
	}

	diff, err := uc.diffs.FindFirstByReleaseID(ctx, release.ID)
	if err != nil || diff == nil {
		// No diff yet — re-enqueue as a diff job from scratch
		if err := uc.releases.UpdateStatus(ctx, release.ID, entity.ReleaseStatusPending); err != nil {
			return "", "", fmt.Errorf("ReleaseUseCase.ReanalyzeRelease: updating status: %w", err)
		}
		jobID, err := uc.queue.Enqueue(ctx, jobTypeDiff, release.ID)
		if err != nil {
			return "", "", fmt.Errorf("ReleaseUseCase.ReanalyzeRelease: enqueue diff: %w", err)
		}
		return "release re-queued for diffing and analysis", jobID, nil
	}

	// Diff exists — re-enqueue analysis job
	if err := uc.releases.UpdateStatus(ctx, release.ID, entity.ReleaseStatusAnalyzing); err != nil {
		return "", "", fmt.Errorf("ReleaseUseCase.ReanalyzeRelease: updating status: %w", err)
	}
	jobID, err := uc.queue.Enqueue(ctx, jobTypeAnalyze, diff.ID)
	if err != nil {
		return "", "", fmt.Errorf("ReleaseUseCase.ReanalyzeRelease: enqueue analyze: %w", err)
	}
	return "release re-queued for analysis", jobID, nil
}

// GetAnalysisHistory returns the analysis history for a package across all its releases.
func (uc *UseCase) GetAnalysisHistory(ctx context.Context, orgID, packageID uint) ([]entity.AnalysisHistoryEntry, error) {
	// Verify package belongs to org
	pkg, err := uc.packages.FindByID(ctx, packageID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("ReleaseUseCase.GetAnalysisHistory: verifying package: %w", err)
	}
	if pkg.OrgID != orgID {
		return nil, entity.ErrNotFound
	}

	// Load all releases
	releases, err := uc.releases.FindByPackageIDAll(ctx, packageID)
	if err != nil {
		return nil, fmt.Errorf("ReleaseUseCase.GetAnalysisHistory: loading releases: %w", err)
	}

	if len(releases) == 0 {
		return []entity.AnalysisHistoryEntry{}, nil
	}

	// Batch-load diffs
	releaseIDs := make([]uint, len(releases))
	for i, rel := range releases {
		releaseIDs[i] = rel.ID
	}

	diffs, err := uc.diffs.FindByReleaseIDs(ctx, releaseIDs)
	if err != nil {
		return nil, fmt.Errorf("ReleaseUseCase.GetAnalysisHistory: loading diffs: %w", err)
	}
	diffByRelease := make(map[uint]entity.Diff)
	for _, d := range diffs {
		diffByRelease[d.ReleaseID] = d
	}

	// Batch-load analyses
	diffIDs := make([]uint, 0, len(diffs))
	for _, d := range diffs {
		diffIDs = append(diffIDs, d.ID)
	}
	analysisByDiff := make(map[uint]entity.Analysis)
	if len(diffIDs) > 0 {
		analyses, err := uc.analyses.FindByDiffIDs(ctx, diffIDs)
		if err != nil {
			return nil, fmt.Errorf("ReleaseUseCase.GetAnalysisHistory: loading analyses: %w", err)
		}
		for _, a := range analyses {
			analysisByDiff[a.DiffID] = a
		}
	}

	// Build history entries
	result := make([]entity.AnalysisHistoryEntry, 0)
	for _, rel := range releases {
		d, hasDiff := diffByRelease[rel.ID]
		if !hasDiff {
			if rel.Status == entity.ReleaseStatusCompleted {
				result = append(result, entity.AnalysisHistoryEntry{
					ReleaseID:      rel.ID,
					Version:        rel.Version,
					Classification: "baseline",
					PublishedAt:    rel.PublishedAt.Format("2006-01-02T15:04:05Z"),
				})
			}
			continue
		}

		a, hasAnalysis := analysisByDiff[d.ID]
		if !hasAnalysis {
			continue
		}

		result = append(result, entity.AnalysisHistoryEntry{
			ReleaseID:      rel.ID,
			Version:        rel.Version,
			Classification: string(a.Classification),
			Confidence:     a.Confidence,
			Reasoning:      a.Reasoning,
			ModelUsed:      a.ModelUsed,
			AnalyzerType:   string(a.AnalyzerType),
			AnalyzedAt:     a.CreatedAt.Format("2006-01-02T15:04:05Z"),
			PublishedAt:    rel.PublishedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return result, nil
}
