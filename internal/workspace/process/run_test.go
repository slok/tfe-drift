package process_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
	"github.com/slok/tfe-drift/internal/workspace/process/processmock"
)

func TestDriftDetectionPlanProcessor(t *testing.T) {
	tests := map[string]struct {
		mock          func(mc *processmock.WorkspaceCheckPlanCreator)
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Not having workspaces shouldn't create any plan.": {
			mock:          func(mc *processmock.WorkspaceCheckPlanCreator) {},
			workspaces:    []model.Workspace{},
			expWorkspaces: []model.Workspace{},
		},

		"Having workspaces should create drift detection plans.": {
			mock: func(mc *processmock.WorkspaceCheckPlanCreator) {
				mc.On("CreateCheckPlan", mock.Anything, model.Workspace{ID: "wk1"}, "test").Once().Return(&model.Plan{ID: "p1"}, nil)
				mc.On("CreateCheckPlan", mock.Anything, model.Workspace{ID: "wk2"}, "test").Once().Return(&model.Plan{ID: "p2"}, nil)
				mc.On("CreateCheckPlan", mock.Anything, model.Workspace{ID: "wk3"}, "test").Once().Return(&model.Plan{ID: "p3"}, nil)
			},
			workspaces: []model.Workspace{{ID: "wk1"}, {ID: "wk2"}, {ID: "wk3"}},
			expWorkspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1"}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2"}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3"}},
			},
		},

		"Having an error while create drift detection plans should not stop the process.": {
			mock: func(mc *processmock.WorkspaceCheckPlanCreator) {
				mc.On("CreateCheckPlan", mock.Anything, model.Workspace{ID: "wk1"}, "test").Once().Return(&model.Plan{ID: "p1"}, nil)
				mc.On("CreateCheckPlan", mock.Anything, model.Workspace{ID: "wk2"}, "test").Once().Return(nil, fmt.Errorf("something"))
				mc.On("CreateCheckPlan", mock.Anything, model.Workspace{ID: "wk3"}, "test").Once().Return(&model.Plan{ID: "p3"}, nil)
			},
			workspaces: []model.Workspace{{ID: "wk1"}, {ID: "wk2"}, {ID: "wk3"}},
			expWorkspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1"}},
				{ID: "wk2"},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3"}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			mc := processmock.NewWorkspaceCheckPlanCreator(t)
			test.mock(mc)

			p := process.NewDriftDetectionPlanProcessor(log.Noop, mc, "test")
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}
