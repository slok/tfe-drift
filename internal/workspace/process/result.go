package process

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/slok/tfe-drift/internal/internalerrors"
	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

func NewDriftDetectionPlansResultProcessor(logger log.Logger, noErrorOnDrift bool) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "DriftDetectionPlansResult"})

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		hasChanges := false
		hasErrors := false
		for _, wk := range wks {
			logger := logger.WithValues(log.Kv{
				"workspace": wk.Name,
				"run-id":    wk.LastDriftPlan.ID,
				"run-url":   wk.LastDriftPlan.URL,
			})

			switch {
			case wk.LastDriftPlan.HasChanges:
				hasChanges = true
				logger.Warningf("Drift detected")
			case wk.LastDriftPlan.Status == model.PlanStatusFinishedNotOK:
				hasErrors = true
				logger.Warningf("Drift detection plan failed")
			}
		}

		if !noErrorOnDrift && hasChanges {
			return nil, internalerrors.ErrDriftDetected
		}

		if hasErrors {
			return nil, internalerrors.ErrDriftDetectionPlanFailed
		}

		return wks, nil
	})
}

func NewDetailedJSONResultProcessor(out io.Writer) Processor {
	type jsonResultWorkspace struct {
		Name                    string `json:"name"`
		ID                      string `json:"id"`
		DriftDetectionRunID     string `json:"drift_detection_run_id"`
		DriftDetectionRunURL    string `json:"drift_detection_run_url"`
		Drift                   bool   `json:"drift"`
		DriftDetectionPlanError bool   `json:"drift_detection_plan_error"`
	}

	type jsonResult struct {
		Workspaces              map[string]jsonResultWorkspace `json:"workspaces"`
		Drift                   bool                           `json:"drift"`
		DriftDetectionPlanError bool                           `json:"drift_detection_plan_error"`
		CreatedAt               time.Time                      `json:"created_at"`
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		drift := false
		driftError := false
		workspaces := map[string]jsonResultWorkspace{}
		for _, wk := range wks {
			jrwk := jsonResultWorkspace{
				Name:                    wk.Name,
				ID:                      wk.ID,
				DriftDetectionRunID:     wk.LastDriftPlan.ID,
				DriftDetectionRunURL:    wk.LastDriftPlan.URL,
				Drift:                   wk.LastDriftPlan.HasChanges,
				DriftDetectionPlanError: wk.LastDriftPlan.Status == model.PlanStatusFinishedNotOK,
			}

			if wk.LastDriftPlan.HasChanges {
				drift = true
				jrwk.Drift = true
			}

			if wk.LastDriftPlan.Status == model.PlanStatusFinishedNotOK {
				driftError = true
				jrwk.DriftDetectionPlanError = true
			}

			workspaces[wk.Name] = jrwk
		}

		root := jsonResult{
			Workspaces:              workspaces,
			Drift:                   drift,
			DriftDetectionPlanError: driftError,
			CreatedAt:               time.Now().UTC(),
		}
		data, err := json.MarshalIndent(root, "", "\t")
		if err != nil {
			return nil, fmt.Errorf("the result could not be marshaled in JSON: %w", err)
		}

		_, err = out.Write(data)
		if err != nil {
			return nil, fmt.Errorf("result could not be written in the output: %w", err)
		}

		return wks, nil
	})
}
