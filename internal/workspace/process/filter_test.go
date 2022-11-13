package process_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
)

func TestExcludeNameProcessor(t *testing.T) {
	tests := map[string]struct {
		regexes       []string
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Having no regexes should not exclude anything.": {
			regexes: []string{},
			workspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
		},

		"Having regexes should exclude matched workspaces (Single regex).": {
			regexes: []string{
				"^wk[13]$",
			},
			workspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk2"}, {Name: "wk4"},
			},
		},

		"Having regexes should exclude matched workspaces (Multiple regex).": {
			regexes: []string{
				"^wk[13]$",
				"^wk2$",
			},
			workspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk4"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			p, err := process.NewExcludeNameProcessor(log.Noop, test.regexes)
			require.NoError(err)
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}

func TestIncludeNameProcessor(t *testing.T) {
	tests := map[string]struct {
		regexes       []string
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Having no regexes should include all.": {
			regexes: []string{},
			workspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
		},

		"Having regexes should include matched workspaces (Single regex).": {
			regexes: []string{
				"^wk[13]$",
			},
			workspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk3"},
			},
		},

		"Having regexes should exclude matched workspaces (Multiple regex).": {
			regexes: []string{
				"^wk[13]$",
				"^wk2$",
			},
			workspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			p, err := process.NewIncludeNameProcessor(log.Noop, test.regexes)
			require.NoError(err)
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}

func TestLimitMaxProcessor(t *testing.T) {
	tests := map[string]struct {
		max           int
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Having no limit should not limit.": {
			max:           0,
			workspaces:    []model.Workspace{{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"}},
			expWorkspaces: []model.Workspace{{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"}},
		},

		"Having a limit should limit.": {
			max:           2,
			workspaces:    []model.Workspace{{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"}},
			expWorkspaces: []model.Workspace{{Name: "wk1"}, {Name: "wk2"}},
		},

		"Having a bigger limit than workspaces should return all workspaces.": {
			max:           200,
			workspaces:    []model.Workspace{{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"}},
			expWorkspaces: []model.Workspace{{Name: "wk1"}, {Name: "wk2"}, {Name: "wk3"}, {Name: "wk4"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := process.NewLimitMaxProcessor(log.Noop, test.max)
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}

func TestFilterQueuedDriftDetectorProcessor(t *testing.T) {
	tests := map[string]struct {
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Having queued state plans should ignore them.": {
			workspaces: []model.Workspace{
				{Name: "wk1"},
				{Name: "wk2", LastDriftPlan: &model.Plan{Status: model.PlanStatusWaiting}},
				{Name: "wk3", LastDriftPlan: &model.Plan{Status: model.PlanStatusFinishedOK}},
				{Name: "wk4", LastDriftPlan: &model.Plan{Status: model.PlanStatusWaiting}},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk1"},
				{Name: "wk3", LastDriftPlan: &model.Plan{Status: model.PlanStatusFinishedOK}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := process.NewFilterQueuedDriftDetectorProcessor(log.Noop)
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}

func TestFilterDriftDetectionsBeforeProcessor(t *testing.T) {
	t0 := time.Now()

	tests := map[string]struct {
		notBefore     time.Duration
		workspaces    []model.Workspace
		expWorkspaces []model.Workspace
		expErr        bool
	}{
		"Having drift plans before the max age should filter them.": {
			notBefore: 1 * time.Hour,
			workspaces: []model.Workspace{
				{Name: "wk1"},
				{Name: "wk2", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-1 * 15 * time.Minute)}},
				{Name: "wk3", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-1 * 150 * time.Minute)}},
				{Name: "wk4", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-1 * 59 * time.Minute)}},
			},
			expWorkspaces: []model.Workspace{
				{Name: "wk1"},
				{Name: "wk3", LastDriftPlan: &model.Plan{CreatedAt: t0.Add(-1 * 150 * time.Minute)}},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			p := process.NewFilterDriftDetectionsBeforeProcessor(log.Noop, test.notBefore)
			gotWks, err := p.Process(context.TODO(), test.workspaces)

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces, gotWks)
			}
		})
	}
}
