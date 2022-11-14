package tfe_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	gotfe "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/storage/tfe"
	"github.com/slok/tfe-drift/internal/storage/tfe/tfemock"
)

func TestRepositoryListWorkspaces(t *testing.T) {
	tests := map[string]struct {
		mock          func(mc *tfemock.Client)
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Having an error while returning workspaces, should fail.": {
			mock: func(mc *tfemock.Client) {
				mc.On("ListWorkspaces", mock.Anything, "test", mock.Anything).Once().Return(nil, fmt.Errorf("something"))
			},
			expErr: true,
		},

		"Not returning workspaces shouldn't fail.": {
			mock: func(mc *tfemock.Client) {
				mc.On("ListWorkspaces", mock.Anything, "test", mock.Anything).Once().Return(&gotfe.WorkspaceList{}, nil)
			},
			expWorkspaces: []model.Workspace{},
		},

		"Returning workspaces should map the model.": {
			mock: func(mc *tfemock.Client) {
				mc.On("ListWorkspaces", mock.Anything, "test", mock.Anything).Once().Return(&gotfe.WorkspaceList{
					Items: []*gotfe.Workspace{
						{ID: "test-id-1", Name: "test-1"},
						{ID: "test-id-2", Name: "test-2"},
						{ID: "test-id-3", Name: "test-3"},
					},
				}, nil)
			},
			expWorkspaces: []model.Workspace{
				{ID: "test-id-1", Name: "test-1", OriginalObject: &gotfe.Workspace{ID: "test-id-1", Name: "test-1"}},
				{ID: "test-id-2", Name: "test-2", OriginalObject: &gotfe.Workspace{ID: "test-id-2", Name: "test-2"}},
				{ID: "test-id-3", Name: "test-3", OriginalObject: &gotfe.Workspace{ID: "test-id-3", Name: "test-3"}},
			},
		},

		"Returning workspaces in multiple pages should map the models.": {
			mock: func(mc *tfemock.Client) {
				mc.On("ListWorkspaces", mock.Anything, "test", mock.Anything).Once().Return(&gotfe.WorkspaceList{
					Pagination: &gotfe.Pagination{CurrentPage: 0, NextPage: 2},
					Items:      []*gotfe.Workspace{{ID: "test-id-1", Name: "test-1"}}}, nil)

				mc.On("ListWorkspaces", mock.Anything, "test", mock.Anything).Once().Return(&gotfe.WorkspaceList{
					Pagination: &gotfe.Pagination{CurrentPage: 2, NextPage: 3},
					Items:      []*gotfe.Workspace{{ID: "test-id-2", Name: "test-2"}}}, nil)

				mc.On("ListWorkspaces", mock.Anything, "test", mock.Anything).Once().Return(&gotfe.WorkspaceList{
					Pagination: &gotfe.Pagination{CurrentPage: 3, NextPage: 0},
					Items:      []*gotfe.Workspace{{ID: "test-id-3", Name: "test-3"}}}, nil)
			},
			expWorkspaces: []model.Workspace{
				{ID: "test-id-1", Name: "test-1", OriginalObject: &gotfe.Workspace{ID: "test-id-1", Name: "test-1"}},
				{ID: "test-id-2", Name: "test-2", OriginalObject: &gotfe.Workspace{ID: "test-id-2", Name: "test-2"}},
				{ID: "test-id-3", Name: "test-3", OriginalObject: &gotfe.Workspace{ID: "test-id-3", Name: "test-3"}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			mc := tfemock.NewClient(t)
			test.mock(mc)

			r, _ := tfe.NewRepository(mc, "test", "https://test-tfe-drift.dev", "test")
			gotWks, err := r.ListWorkspaces(context.TODO(), nil, nil)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}

		})
	}
}

func TestRepositoryCreateCheckPlan(t *testing.T) {
	t0 := time.Now()

	tests := map[string]struct {
		mock      func(mc *tfemock.Client)
		workspace model.Workspace
		expPlan   *model.Plan
		expErr    bool
	}{
		"Having an error while creating a plan, should fail.": {
			mock: func(mc *tfemock.Client) {
				mc.On("CreateRun", mock.Anything, mock.Anything).Once().Return(nil, fmt.Errorf("something"))
			},
			expErr: true,
		},

		"Creating a plan should map the model.": {
			mock: func(mc *tfemock.Client) {
				mc.On("CreateRun", mock.Anything, mock.Anything).Once().Return(&gotfe.Run{
					ID:         "test-id-1",
					Message:    "test-1",
					HasChanges: true,
					Status:     gotfe.RunPlannedAndFinished,
					CreatedAt:  t0,
				}, nil)
			},
			expPlan: &model.Plan{
				ID:         "test-id-1",
				Message:    "test-1",
				HasChanges: true,
				Status:     model.PlanStatusFinishedOK,
				CreatedAt:  t0,
				URL:        "https://test-tfe-drift.dev/app/test/workspaces//runs/test-id-1",
				OriginalObject: &gotfe.Run{
					ID:         "test-id-1",
					Message:    "test-1",
					HasChanges: true,
					Status:     gotfe.RunPlannedAndFinished,
					CreatedAt:  t0,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			mc := tfemock.NewClient(t)
			test.mock(mc)

			r, _ := tfe.NewRepository(mc, "test", "https://test-tfe-drift.dev", "test")
			gotPlan, err := r.CreateCheckPlan(context.TODO(), test.workspace, "test")

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlan, gotPlan)
			}

		})
	}
}

func TestRepositoryGetCheckPlan(t *testing.T) {
	t0 := time.Now()

	tests := map[string]struct {
		mock    func(mc *tfemock.Client)
		expPlan *model.Plan
		expErr  bool
	}{
		"Having an error while getting a plan, should fail.": {
			mock: func(mc *tfemock.Client) {
				mc.On("ReadRun", mock.Anything, "test").Once().Return(nil, fmt.Errorf("something"))
			},
			expErr: true,
		},

		"Getting a plan should map the model.": {
			mock: func(mc *tfemock.Client) {
				mc.On("ReadRun", mock.Anything, "test").Once().Return(&gotfe.Run{
					ID:         "test-id-1",
					Message:    "test-1",
					HasChanges: false,
					Status:     gotfe.RunPlanQueued,
					CreatedAt:  t0,
				}, nil)
			},
			expPlan: &model.Plan{
				ID:         "test-id-1",
				Message:    "test-1",
				HasChanges: false,
				Status:     model.PlanStatusWaiting,
				CreatedAt:  t0,
				URL:        "https://test-tfe-drift.dev/app/test/workspaces//runs/test-id-1",
				OriginalObject: &gotfe.Run{
					ID:         "test-id-1",
					Message:    "test-1",
					HasChanges: false,
					Status:     gotfe.RunPlanQueued,
					CreatedAt:  t0,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			mc := tfemock.NewClient(t)
			test.mock(mc)

			r, _ := tfe.NewRepository(mc, "test", "https://test-tfe-drift.dev", "test")
			gotPlan, err := r.GetCheckPlan(context.TODO(), model.Workspace{}, "test")

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlan, gotPlan)
			}
		})
	}
}

func TestRepositoryLatestCheckPlan(t *testing.T) {
	t0 := time.Now()

	tests := map[string]struct {
		mock      func(mc *tfemock.Client)
		workspace model.Workspace
		expPlan   *model.Plan
		expErr    bool
	}{
		"Having an error while getting a plan, should fail.": {
			workspace: model.Workspace{ID: "test"},
			mock: func(mc *tfemock.Client) {
				mc.On("ListRuns", mock.Anything, "test", mock.Anything).Once().Return(nil, fmt.Errorf("something"))
			},
			expErr: true,
		},

		"Having no runs should return fail.": {
			workspace: model.Workspace{ID: "test"},
			mock: func(mc *tfemock.Client) {
				mc.On("ListRuns", mock.Anything, "test", mock.Anything).Once().Return(&gotfe.RunList{}, nil)
			},
			expErr: true,
		},

		"Getting a plan should map the model.": {
			workspace: model.Workspace{ID: "test"},
			mock: func(mc *tfemock.Client) {
				expOpts := &gotfe.RunListOptions{
					Search:      "tfe-drift/detector-id/test-id",
					ListOptions: gotfe.ListOptions{PageSize: 1},
				}
				mc.On("ListRuns", mock.Anything, "test", expOpts).Once().Return(&gotfe.RunList{Items: []*gotfe.Run{
					{
						ID:         "test-id-1",
						Message:    "test-1",
						HasChanges: false,
						Status:     gotfe.RunPlanQueued,
						CreatedAt:  t0,
					}}}, nil)
			},
			expPlan: &model.Plan{
				ID:         "test-id-1",
				Message:    "test-1",
				HasChanges: false,
				Status:     model.PlanStatusWaiting,
				CreatedAt:  t0,
				URL:        "https://test-tfe-drift.dev/app/test/workspaces//runs/test-id-1",
				OriginalObject: &gotfe.Run{
					ID:         "test-id-1",
					Message:    "test-1",
					HasChanges: false,
					Status:     gotfe.RunPlanQueued,
					CreatedAt:  t0,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			// Mocks.
			mc := tfemock.NewClient(t)
			test.mock(mc)

			r, _ := tfe.NewRepository(mc, "test", "https://test-tfe-drift.dev", "test-id")
			gotPlan, err := r.GetLatestCheckPlan(context.TODO(), test.workspace)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expPlan, gotPlan)
			}
		})
	}
}
