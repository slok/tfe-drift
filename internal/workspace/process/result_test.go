package process_test

import (
	"bytes"
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
)

func TestDriftDetectionPlansResultProcessor(t *testing.T) {
	tests := map[string]struct {
		noErrorOnDrift bool
		workspaces     []model.Workspace
		expErr         bool
	}{
		"Not having workspaces shouldn't error.": {
			workspaces: []model.Workspace{},
			expErr:     false,
		},

		"Having workspaces without changes should not fail.": {
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", HasChanges: false}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2", HasChanges: false}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", HasChanges: false}},
			},
			expErr: false,
		},

		"Having a workspace with changes should fail.": {
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", HasChanges: false}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2", HasChanges: true}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", HasChanges: false}},
			},
			expErr: true,
		},

		"Having a workspace with plan errors should fail.": {
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", Status: model.PlanStatusFinishedOK}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2", Status: model.PlanStatusFinishedNotOK}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", Status: model.PlanStatusFinishedOK}},
			},
			expErr: true,
		},

		"Having a workspace with changes but with no error on drift option, should not fail.": {
			noErrorOnDrift: true,
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", HasChanges: false}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2", HasChanges: true}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", Status: model.PlanStatusFinishedNotOK}},
			},
			expErr: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := process.NewDriftDetectionPlansResultProcessor(log.Noop, test.noErrorOnDrift)
			_, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestDetailedJSONResultProcessor(t *testing.T) {
	tests := map[string]struct {
		workspaces     []model.Workspace
		expResultRegex *regexp.Regexp
		expErr         bool
	}{
		"Not having workspaces shouldn't error.": {
			workspaces: []model.Workspace{},
			expResultRegex: regexp.MustCompile(`{
	"workspaces": {},
	"drift": false,
	"drift_detection_plan_error": false,
	"ok": true,
	"created_at": ".*"
}`),
		},

		"Having workspaces should return the result.": {
			workspaces: []model.Workspace{
				{ID: "wk1", Name: "wk1", Tags: []string{"t1"}, LastDriftPlan: &model.Plan{ID: "p1", HasChanges: false}},
				{ID: "wk2", Name: "wk2", Tags: []string{"t2"}, LastDriftPlan: &model.Plan{ID: "p2", HasChanges: true}},
				{ID: "wk3", Name: "wk3", Tags: []string{"t3"}, LastDriftPlan: &model.Plan{ID: "p3", Status: model.PlanStatusFinishedNotOK}},
			},
			expResultRegex: regexp.MustCompile(`{
	"workspaces": {
		"wk1": {
			"name": "wk1",
			"id": "wk1",
			"tags": \[
				"t1"
			\],
			"drift_detection_run_id": "p1",
			"drift_detection_run_url": "",
			"drift": false,
			"drift_detection_plan_error": false,
			"ok": true
		},
		"wk2": {
			"name": "wk2",
			"id": "wk2",
			"tags": \[
				"t2"
			\],
			"drift_detection_run_id": "p2",
			"drift_detection_run_url": "",
			"drift": true,
			"drift_detection_plan_error": false,
			"ok": false
		},
		"wk3": {
			"name": "wk3",
			"id": "wk3",
			"tags": \[
				"t3"
			\],
			"drift_detection_run_id": "p3",
			"drift_detection_run_url": "",
			"drift": false,
			"drift_detection_plan_error": true,
			"ok": false
		}
	},
	"drift": true,
	"drift_detection_plan_error": true,
	"ok": false,
	"created_at": ".*"
}`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			var b bytes.Buffer
			p := process.NewDetailedJSONResultProcessor(&b, true)
			_, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Regexp(test.expResultRegex, b.String())
			}
		})
	}
}
