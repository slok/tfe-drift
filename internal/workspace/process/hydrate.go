package process

import (
	"context"
	"errors"

	"github.com/slok/tfe-drift/internal/internalerrors"
	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

type WorkspaceLatestCheckPlanGetter interface {
	GetLatestCheckPlan(ctx context.Context, w model.Workspace) (*model.Plan, error)
}

//go:generate mockery --case underscore --output processmock --outpkg processmock --name WorkspaceLatestCheckPlanGetter

func NewHydrateLatestDetectionPlanProcessor(logger log.Logger, g WorkspaceLatestCheckPlanGetter) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "HydrateLatestDetectionPlan"})

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Getting Workspaces' latest drift detection")

		// TODO(slok): Add concurrency.
		newWks := []model.Workspace{}
		for _, wk := range wks {
			plan, err := g.GetLatestCheckPlan(ctx, wk)
			if err != nil && !errors.Is(err, internalerrors.ErrNotExist) {
				// TODO(slok): Add strict as an option so we can fail or not based on this option.
				// Don't stop all the  process for other workspaces because of one workspace error.
				logger.WithValues(log.Kv{"workspace": wk.Name}).Errorf("could not get latest drift detection plan for workspaces %q: %w", wk.Name, err)
			}
			wk.LastDriftPlan = plan
			newWks = append(newWks, wk)
		}

		return newWks, nil
	})
}
