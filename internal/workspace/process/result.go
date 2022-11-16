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

func NewDriftDetectionPlansResultProcessor(logger log.Logger, noErrorDriftPlans bool) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "DriftDetectionPlansResult"})

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		hasChanges := false
		hasErrors := false
		for _, wk := range wks {
			var driftPlan model.Plan
			if wk.LastDriftPlan != nil {
				driftPlan = *wk.LastDriftPlan
			}

			logger := logger.WithValues(log.Kv{
				"workspace": wk.Name,
				"run-id":    driftPlan.ID,
				"run-url":   driftPlan.URL,
			})

			switch {
			case driftPlan.HasChanges:
				hasChanges = true
				logger.Warningf("Drift detected")
			case driftPlan.Status == model.PlanStatusFinishedNotOK:
				hasErrors = true
				logger.Warningf("Drift detection plan failed")
			}
		}

		switch {
		case noErrorDriftPlans:
			return wks, nil
		case hasChanges:
			return nil, internalerrors.ErrDriftDetected
		case hasErrors:
			return nil, internalerrors.ErrDriftDetectionPlanFailed
		}

		return wks, nil
	})
}

func NewDetailedJSONResultProcessor(out io.Writer, pretty bool) Processor {
	type jsonResultWorkspace struct {
		Name                    string   `json:"name"`
		ID                      string   `json:"id"`
		Tags                    []string `json:"tags"`
		DriftDetectionRunID     string   `json:"drift_detection_run_id"`
		DriftDetectionRunURL    string   `json:"drift_detection_run_url"`
		Drift                   bool     `json:"drift"`
		DriftDetectionPlanError bool     `json:"drift_detection_plan_error"`
		OK                      bool     `json:"ok"`
	}

	type jsonResult struct {
		Workspaces              map[string]jsonResultWorkspace `json:"workspaces"`
		Drift                   bool                           `json:"drift"`
		DriftDetectionPlanError bool                           `json:"drift_detection_plan_error"`
		OK                      bool                           `json:"ok"`
		CreatedAt               time.Time                      `json:"created_at"`
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		drift := false
		driftError := false
		workspaces := map[string]jsonResultWorkspace{}
		for _, wk := range wks {
			var driftPlan model.Plan
			if wk.LastDriftPlan != nil {
				driftPlan = *wk.LastDriftPlan
			}

			hasDrift := driftPlan.HasChanges
			hasDriftDetectionError := driftPlan.Status == model.PlanStatusFinishedNotOK

			jrwk := jsonResultWorkspace{
				Name:                    wk.Name,
				ID:                      wk.ID,
				Tags:                    wk.Tags,
				DriftDetectionRunID:     driftPlan.ID,
				DriftDetectionRunURL:    driftPlan.URL,
				Drift:                   hasDrift,
				DriftDetectionPlanError: hasDriftDetectionError,
				OK:                      !hasDrift && !hasDriftDetectionError,
			}

			if hasDrift {
				drift = true
				jrwk.Drift = true
			}

			if hasDriftDetectionError {
				driftError = true
				jrwk.DriftDetectionPlanError = true
			}

			workspaces[wk.Name] = jrwk
		}

		root := jsonResult{
			Workspaces:              workspaces,
			Drift:                   drift,
			DriftDetectionPlanError: driftError,
			OK:                      !drift && !driftError,
			CreatedAt:               time.Now().UTC(),
		}

		data, err := marshallJSON(root, pretty)
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

func marshallJSON(obj interface{}, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(obj, "", "\t")
	}
	return json.Marshal(obj)
}
