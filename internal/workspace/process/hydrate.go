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

func NewHydrateLatestDetectionPlanProcessor(ctx context.Context, logger log.Logger, g WorkspaceLatestCheckPlanGetter, workers int) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "HydrateLatestDetectionPlan"})

	// Run workers for concurrent fetch.
	jobs := make(chan model.Workspace)
	res := make(chan getLatestCheckPlanWorkerResult)
	for i := 0; i < workers; i++ {
		go getLatestCheckPlanWorker(ctx, g, jobs, res)
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Getting Workspaces' latest drift detection")

		// Send retrievals to workers, they will handle concurrency.
		go func() {
			for _, wk := range wks {
				jobs <- wk
			}
		}()

		// Wait for results and index by workspace ID.
		indexedResults := map[string]model.Workspace{}
		for i := 0; i < len(wks); i++ {
			result := <-res

			if result.err != nil && !errors.Is(result.err, internalerrors.ErrNotExist) {
				// TODO(slok): Add strict as an option so we can fail or not based on this option.
				// Don't stop all the  process for other workspaces because of one workspace error.
				logger.WithValues(log.Kv{"workspace": result.wk.Name}).Errorf("could not get latest drift detection plan for workspaces %q: %w", result.wk.Name, result.err)
			}

			result.wk.LastDriftPlan = result.plan
			indexedResults[result.wk.ID] = result.wk
		}

		// Set results in the correct order.
		newWks := []model.Workspace{}
		for _, wk := range wks {
			newWks = append(newWks, indexedResults[wk.ID])
		}

		return newWks, nil
	})
}

type getLatestCheckPlanWorkerResult struct {
	wk   model.Workspace
	plan *model.Plan
	err  error
}

func getLatestCheckPlanWorker(ctx context.Context, g WorkspaceLatestCheckPlanGetter, workspaces <-chan model.Workspace, results chan<- getLatestCheckPlanWorkerResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case wk := <-workspaces:
			plan, err := g.GetLatestCheckPlan(context.Background(), wk)
			results <- getLatestCheckPlanWorkerResult{wk: wk, plan: plan, err: err}
		}
	}
}
