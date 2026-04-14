// Package dashboarduc implements the business logic for dashboard operations.
package dashboarduc

import (
	"context"
	"fmt"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

const jobTypeAnalyze = "analyze"

// UseCase implements usecase.DashboardService.
type UseCase struct {
	dashboard usecase.DashboardRepository
	releases  usecase.ReleaseRepository
	diffs     usecase.DiffRepository
	analyses  usecase.AnalysisRepository
	queue     usecase.QueueEnqueuer
}

// New creates a new dashboard UseCase.
func New(
	dashboard usecase.DashboardRepository,
	releases usecase.ReleaseRepository,
	diffs usecase.DiffRepository,
	analyses usecase.AnalysisRepository,
	queue usecase.QueueEnqueuer,
) *UseCase {
	return &UseCase{
		dashboard: dashboard,
		releases:  releases,
		diffs:     diffs,
		analyses:  analyses,
		queue:     queue,
	}
}

// GetStats returns the overview statistics for the dashboard.
func (uc *UseCase) GetStats(ctx context.Context, orgID uint) (*entity.DashboardStats, error) {
	stats, err := uc.dashboard.GetStats(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("DashboardUseCase.GetStats: %w", err)
	}
	return stats, nil
}

// GetRecentReleases returns a paginated list of recent releases with package and classification info.
func (uc *UseCase) GetRecentReleases(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error) {
	results, total, err := uc.releases.FindByOrgIDWithDetails(ctx, orgID, page, limit, sortClause, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("DashboardUseCase.GetRecentReleases: %w", err)
	}
	return results, total, nil
}

// GetChartData returns all chart data for the dashboard within the given time range.
func (uc *UseCase) GetChartData(ctx context.Context, orgID uint, from, to time.Time) (*entity.ChartData, error) {
	data := &entity.ChartData{}

	// 1. Release activity
	activityRows, err := uc.dashboard.GetReleaseActivity(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("DashboardUseCase.GetChartData: release activity: %w", err)
	}

	dayMap := make(map[string]int64)
	for _, row := range activityRows {
		dayMap[row.Date.Format("2006-01-02")] = row.Count
	}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		data.ReleaseActivity = append(data.ReleaseActivity, entity.ChartReleaseActivityPoint{
			Date:     key,
			Releases: dayMap[key],
		})
	}

	// 2. Classification distribution
	classRows, err := uc.dashboard.GetClassificationDistribution(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("DashboardUseCase.GetChartData: classification dist: %w", err)
	}
	baselineCount, err := uc.dashboard.GetBaselineCount(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("DashboardUseCase.GetChartData: baseline count: %w", err)
	}
	for _, row := range classRows {
		data.Classifications = append(data.Classifications, entity.ChartClassificationCount{
			Classification: row.Classification,
			Count:          row.Count,
		})
	}
	if baselineCount > 0 {
		data.Classifications = append(data.Classifications, entity.ChartClassificationCount{
			Classification: "baseline",
			Count:          baselineCount,
		})
	}
	if len(data.Classifications) == 0 {
		data.Classifications = []entity.ChartClassificationCount{}
	}

	// 3. Ecosystem distribution
	ecoRows, err := uc.dashboard.GetEcosystemDistribution(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("DashboardUseCase.GetChartData: ecosystem dist: %w", err)
	}
	for _, row := range ecoRows {
		data.Ecosystems = append(data.Ecosystems, entity.ChartEcosystemCount{
			Ecosystem: row.Ecosystem,
			Count:     row.Count,
		})
	}
	if len(data.Ecosystems) == 0 {
		data.Ecosystems = []entity.ChartEcosystemCount{}
	}

	// 4. Alerts by severity
	alertRows, err := uc.dashboard.GetAlertsBySeverity(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("DashboardUseCase.GetChartData: alerts by severity: %w", err)
	}
	sevMap := make(map[string]int64)
	for _, row := range alertRows {
		sevMap[row.Severity] = row.Count
	}
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		data.AlertsBySeverity = append(data.AlertsBySeverity, entity.ChartAlertSeverityCount{
			Severity: sev,
			Count:    sevMap[sev],
		})
	}

	// 5. Release statuses
	statusRows, err := uc.dashboard.GetReleaseStatusDistribution(ctx, orgID, from, to)
	if err != nil {
		return nil, fmt.Errorf("DashboardUseCase.GetChartData: release statuses: %w", err)
	}
	for _, row := range statusRows {
		data.ReleaseStatuses = append(data.ReleaseStatuses, entity.ChartReleaseStatusCount{
			Status: row.Status,
			Count:  row.Count,
		})
	}
	if len(data.ReleaseStatuses) == 0 {
		data.ReleaseStatuses = []entity.ChartReleaseStatusCount{}
	}

	return data, nil
}

// ReanalyzeAll re-queues all unanalyzed diffs for analysis, scoped to the given org.
func (uc *UseCase) ReanalyzeAll(ctx context.Context, orgID uint) (int, error) {
	if uc.queue == nil {
		return 0, fmt.Errorf("DashboardUseCase.ReanalyzeAll: queue not configured")
	}

	diffIDs, err := uc.dashboard.GetUnanalyzedDiffIDs(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("DashboardUseCase.ReanalyzeAll: %w", err)
	}

	queued := 0
	for _, id := range diffIDs {
		if _, err := uc.queue.Enqueue(ctx, jobTypeAnalyze, id); err != nil {
			return queued, fmt.Errorf("DashboardUseCase.ReanalyzeAll: enqueue diff %d: %w", id, err)
		}
		queued++
	}
	return queued, nil
}
