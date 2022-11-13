package process

import (
	"context"

	"github.com/slok/tfe-drift/internal/internalerrors"
	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

func NewDriftDetectionPlansResultProcessor(logger log.Logger, noErrorOnDrift bool) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "DriftDetectionPlansResult"})

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		hasChanges := false
		for _, wk := range wks {
			logger := logger.WithValues(log.Kv{"workspace": wk.Name, "run-id": wk.LastDriftPlan.ID})

			if wk.LastDriftPlan.HasChanges {
				hasChanges = true
				logger.Warningf("Drift detected")
			}
		}

		if !noErrorOnDrift && hasChanges {
			return nil, internalerrors.ErrDriftDetected
		}

		return wks, nil
	})
}
