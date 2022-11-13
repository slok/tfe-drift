package process_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
	"github.com/slok/tfe-drift/internal/workspace/process/processmock"
)

func TestDriftDetectionPlanWaitProcessor(t *testing.T) {
	tests := map[string]struct {
		mock          func(mg *processmock.WorkspaceCheckPlanGetter)
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Not having workspaces shouldn't wait.": {
			mock:          func(mg *processmock.WorkspaceCheckPlanGetter) {},
			workspaces:    []model.Workspace{},
			expWorkspaces: []model.Workspace{},
		},

		"Having plans that are already finished, should end the execution correctly.": {
			mock: func(mg *processmock.WorkspaceCheckPlanGetter) {
				mg.On("GetCheckPlan", mock.Anything, "p1").Once().Return(&model.Plan{ID: "p1", Status: model.PlanStatusFinishedOK}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p2").Once().Return(&model.Plan{ID: "p2", Status: model.PlanStatusFinishedNotOK}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p3").Once().Return(&model.Plan{ID: "p3", Status: model.PlanStatusFinishedOK}, nil)
			},
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1"}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2"}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3"}},
			},
			expWorkspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", Status: model.PlanStatusFinishedOK}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2", Status: model.PlanStatusFinishedNotOK}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", Status: model.PlanStatusFinishedOK}},
			},
		},

		"Having plans that are not finished, should wait until it can end the execution correctly.": {
			mock: func(mg *processmock.WorkspaceCheckPlanGetter) {
				mg.On("GetCheckPlan", mock.Anything, "p1").Once().Return(&model.Plan{ID: "p1", Status: model.PlanStatusFinishedOK}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p2").Once().Return(&model.Plan{ID: "p2", Status: model.PlanStatusWaiting}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p2").Once().Return(&model.Plan{ID: "p2", Status: model.PlanStatusWaiting}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p2").Once().Return(&model.Plan{ID: "p2", Status: model.PlanStatusFinishedNotOK}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p3").Once().Return(&model.Plan{ID: "p3", Status: model.PlanStatusFinishedOK}, nil)
			},
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1"}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2"}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3"}},
			},
			expWorkspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", Status: model.PlanStatusFinishedOK}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2", Status: model.PlanStatusFinishedNotOK}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", Status: model.PlanStatusFinishedOK}},
			},
		},

		"Having errors, should continue waiting for others and not fail.": {
			mock: func(mg *processmock.WorkspaceCheckPlanGetter) {
				mg.On("GetCheckPlan", mock.Anything, "p1").Once().Return(&model.Plan{ID: "p1", Status: model.PlanStatusFinishedOK}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p2").Once().Return(&model.Plan{ID: "p2", Status: model.PlanStatusWaiting}, nil)
				mg.On("GetCheckPlan", mock.Anything, "p2").Once().Return(nil, fmt.Errorf("something"))
				mg.On("GetCheckPlan", mock.Anything, "p3").Once().Return(&model.Plan{ID: "p3", Status: model.PlanStatusFinishedOK}, nil)
			},
			workspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1"}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2"}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3"}},
			},
			expWorkspaces: []model.Workspace{
				{ID: "wk1", LastDriftPlan: &model.Plan{ID: "p1", Status: model.PlanStatusFinishedOK}},
				{ID: "wk2", LastDriftPlan: &model.Plan{ID: "p2"}},
				{ID: "wk3", LastDriftPlan: &model.Plan{ID: "p3", Status: model.PlanStatusFinishedOK}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			mg := processmock.NewWorkspaceCheckPlanGetter(t)
			test.mock(mg)

			p := process.NewDriftDetectionPlanWaitProcessor(log.Noop, mg, 1*time.Millisecond, 1*time.Hour)
			gotWks, err := p.Process(context.Background(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
			mg.AssertExpectations(t) // Check the calls are exactly what we expect.
		})
	}
}
