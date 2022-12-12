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

func TestHydrateLatestDetectionPlanProcessor(t *testing.T) {
	tests := map[string]struct {
		mock          func(mg *processmock.WorkspaceLatestCheckPlanGetter)
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Not having workspaces shouldn't fail.": {
			mock:          func(mg *processmock.WorkspaceLatestCheckPlanGetter) {},
			workspaces:    []model.Workspace{},
			expWorkspaces: []model.Workspace{},
		},

		"Having workspaces should hydrate with the latest drift detection plans.": {
			mock: func(mg *processmock.WorkspaceLatestCheckPlanGetter) {
				mg.On("GetLatestCheckPlan", mock.Anything, model.Workspace{ID: "wk1"}).Once().Return(&model.Plan{ID: "p1"}, nil)
				mg.On("GetLatestCheckPlan", mock.Anything, model.Workspace{ID: "wk2"}).Once().Return(&model.Plan{ID: "p2"}, nil)
				mg.On("GetLatestCheckPlan", mock.Anything, model.Workspace{ID: "wk3"}).Once().Return(&model.Plan{ID: "p3"}, nil)
			},
			workspaces: []model.Workspace{{ID: "wk1"}, {ID: "wk2"}, {ID: "wk3"}},
			expWorkspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1"}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2"}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3"}},
			},
		},

		"Having an error while getting latest drift detection plans should not stop the process.": {
			mock: func(mg *processmock.WorkspaceLatestCheckPlanGetter) {
				mg.On("GetLatestCheckPlan", mock.Anything, model.Workspace{ID: "wk1"}).Once().Return(&model.Plan{ID: "p1"}, nil)
				mg.On("GetLatestCheckPlan", mock.Anything, model.Workspace{ID: "wk2"}).Once().Return(nil, fmt.Errorf("something"))
				mg.On("GetLatestCheckPlan", mock.Anything, model.Workspace{ID: "wk3"}).Once().Return(&model.Plan{ID: "p3"}, nil)
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
			mc := processmock.NewWorkspaceLatestCheckPlanGetter(t)
			test.mock(mc)

			p := process.NewHydrateLatestDetectionPlanProcessor(context.TODO(), log.Noop, mc, 20)
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}
