package process

import (
	"context"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

type WorkspaceCheckPlanCreator interface {
	CreateCheckPlan(ctx context.Context, w model.Workspace, message string) (*model.Plan, error)
}

//go:generate mockery --case underscore --output processmock --outpkg processmock --name WorkspaceCheckPlanCreator

func NewDriftDetectionPlanProcessor(logger log.Logger, c WorkspaceCheckPlanCreator, planMessage string) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "DriftDetectionPlan"})

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		newWks := []model.Workspace{}
		createdPlans := 0
		for _, wk := range wks {
			logger := logger.WithValues(log.Kv{"workspace": wk.Name})

			plan, err := c.CreateCheckPlan(ctx, wk, planMessage)
			if err != nil {
				// TODO(slok): Add strict as an option so we can fail or not based on this option.
				// Don't stop all the  process for other workspaces because of one workspace error.
				logger.Errorf("Could not create drift detection plan: %w", err)
			} else {
				createdPlans++
				wk.LastDriftPlan = plan
				logger.WithValues(log.Kv{"run-id": wk.LastDriftPlan.ID}).Infof("Drift detection plan created")
			}

			newWks = append(newWks, wk)
		}

		logger.Infof("%d drift detection plans created", createdPlans)

		return newWks, nil
	})
}
