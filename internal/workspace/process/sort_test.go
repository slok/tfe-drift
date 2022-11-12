package process_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
)

func TestSortByOldestDetectionPlanProcessor(t *testing.T) {
	t0 := time.Now()

	tests := map[string]struct {
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Having no workspaces should not fail": {
			workspaces:    []model.Workspace{},
			expWorkspaces: []model.Workspace{},
		},
		"Having workspaces should sort them correctly.": {
			workspaces: []model.Workspace{
				{ID: "w1", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-1 * time.Hour)}},
				{ID: "w2", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-5 * time.Hour)}},
				{ID: "w3", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-2 * time.Hour)}},
				{ID: "w4", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-14 * time.Hour)}},
				{ID: "w5", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-3 * time.Hour)}},
				{ID: "w6"},
			},
			expWorkspaces: []model.Workspace{
				{ID: "w6"},
				{ID: "w4", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-14 * time.Hour)}},
				{ID: "w2", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-5 * time.Hour)}},
				{ID: "w5", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-3 * time.Hour)}},
				{ID: "w3", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-2 * time.Hour)}},
				{ID: "w1", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-1 * time.Hour)}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := process.NewSortByOldestDetectionPlanProcessor(log.Noop)
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}
