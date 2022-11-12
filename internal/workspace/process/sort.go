package process

import (
	"context"
	"sort"
	"time"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

// NewSortByOldestDetectionPlanProcessor will sort the received workspaces by its latest drift detection plan.
// setting the first ones the ones with the oldest detection plan.
func NewSortByOldestDetectionPlanProcessor(logger log.Logger) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "NewSortOldestDetectionPlan"})

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Sorting Workspaces by oldest drift detection")
		sort.SliceStable(wks, func(i, j int) bool {
			// If not last drift plan, treat as the oldest possible TS.
			var ti, tj time.Time
			if wks[i].LastDriftPlan != nil {
				ti = wks[i].LastDriftPlan.CreatedAt
			}
			if wks[j].LastDriftPlan != nil {
				tj = wks[j].LastDriftPlan.CreatedAt
			}

			return ti.Before(tj)
		})

		return wks, nil
	})
}
