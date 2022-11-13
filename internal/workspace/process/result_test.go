package process_test

import (
	"context"
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
