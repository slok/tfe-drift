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

		"Having a workspace with changes but with no error on drift option, should not fail.": {
			noErrorOnDrift: true,
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", HasChanges: false}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2", HasChanges: true}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", HasChanges: false}},
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
			workspaces:     []model.Workspace{},
			expResultRegex: regexp.MustCompile(`{\n\t"workspaces": {},\n\t"drift": false,\n\t"created_at": ".*"\n}`),
		},

		"Having workspaces should return the result.": {
			workspaces: []model.Workspace{
				{ID: "wk1", Name: "wk1", LastDriftPlan: &model.Plan{ID: "p1", HasChanges: false}},
				{ID: "wk2", Name: "wk2", LastDriftPlan: &model.Plan{ID: "p2", HasChanges: true}},
				{ID: "wk3", Name: "wk3", LastDriftPlan: &model.Plan{ID: "p3", HasChanges: false}},
			},
			expResultRegex: regexp.MustCompile(`{
	"workspaces": {
		"wk1": {
			"name": "wk1",
			"id": "wk1",
			"drift_detection_run_id": "p1",
			"drift_detection_run_url": "",
			"drift": false
		},
		"wk2": {
			"name": "wk2",
			"id": "wk2",
			"drift_detection_run_id": "p2",
			"drift_detection_run_url": "",
			"drift": true
		},
		"wk3": {
			"name": "wk3",
			"id": "wk3",
			"drift_detection_run_id": "p3",
			"drift_detection_run_url": "",
			"drift": false
		}
	},
	"drift": true,
	"created_at": ".*"
}`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			var b bytes.Buffer
			p := process.NewDetailedJSONResultProcessor(&b)
			_, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Regexp(test.expResultRegex, b.String())
			}
		})
	}
}