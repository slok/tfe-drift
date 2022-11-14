package process

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

type WorkspaceCheckPlanGetter interface {
	GetCheckPlan(ctx context.Context, w model.Workspace, id string) (*model.Plan, error)
}

//go:generate mockery --case underscore --output processmock --outpkg processmock --name WorkspaceCheckPlanGetter

func NewDriftDetectionPlanWaitProcessor(logger log.Logger, g WorkspaceCheckPlanGetter, pollingDuration, timeoutDuration time.Duration) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "DriftDetectionPlanWait"})

	// waitResult will be used to send results over a channel.
	type waitResult struct {
		wk  model.Workspace
		err error
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		c := make(chan waitResult)

		for _, wk := range wks {
			wk := wk

			// If no plan to check, no need to wait.
			if wk.LastDriftPlan == nil {
				go func() {
					c <- waitResult{wk: wk, err: nil}
				}()
				continue
			}

			logger := logger.WithValues(log.Kv{"workspace": wk.Name, "run-id": wk.LastDriftPlan.ID})

			// For each workspace wait concurrently.
			go func() {
				planID := wk.LastDriftPlan.ID
				logger.Infof("Waiting for drift detection plan to finish...")

				plan, err := waitForPlan(ctx, g, wk, planID, pollingDuration, timeoutDuration)
				if err == nil {
					wk.LastDriftPlan = plan
				}
				c <- waitResult{wk: wk, err: err}
			}()
		}

		// Wait for all workspace drift detection plan waiters to finish.
		// We index the workspaces to maintain the order with the new result.
		indexedWks := map[string]model.Workspace{}
		for i := 0; i < len(wks); i++ {
			res := <-c
			logger := logger.WithValues(log.Kv{"workspace": res.wk.Name})

			switch {
			case res.wk.LastDriftPlan == nil:
				logger.Debugf("No drift plan to check, ignoring...")
			case res.err != nil:
				// TODO(slok): Add strict as an option so we can fail or not based on this option.
				// Don't stop all the  process for other workspaces because of one workspace error.
				logger.WithValues(log.Kv{"run-id": res.wk.LastDriftPlan.ID}).Errorf("Error while waiting for drift detection plan: %s", res.err)
			default:
				logger.WithValues(log.Kv{"run-id": res.wk.LastDriftPlan.ID}).Infof("Drift detection plan finished")
			}

			indexedWks[res.wk.ID] = res.wk
		}

		// Create again our workspaces list in the same order but with the new data.
		newWks := []model.Workspace{}
		for _, wk := range wks {
			newWks = append(newWks, indexedWks[wk.ID])
		}

		return newWks, nil
	})
}

func waitForPlan(ctx context.Context, g WorkspaceCheckPlanGetter, wk model.Workspace, planID string, pollingDur, timeoutDur time.Duration) (*model.Plan, error) {
	ctx, cancel := context.WithTimeout(ctx, timeoutDur)
	defer cancel()

	// Try getting the first time.
	plan, err := g.GetCheckPlan(ctx, wk, planID)
	if err != nil {
		return nil, fmt.Errorf("could not get check plan %q: %w", planID, err)
	}
	if plan.Status != model.PlanStatusWaiting {
		return plan, nil
	}

	// Wait polling in regular intervals.
	ticker := time.NewTicker(pollingDur)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancellation: %w", ctx.Err())
		case <-ticker.C:
			plan, err := g.GetCheckPlan(ctx, wk, planID)
			if err != nil {
				return nil, fmt.Errorf("could not get check plan %q: %w", planID, err)
			}

			// If not waiting, we are finished.
			if plan.Status != model.PlanStatusWaiting {
				return plan, nil
			}

			// We should continue waiting...
		}
	}
}
