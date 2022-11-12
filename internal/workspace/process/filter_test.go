package process_test

import (
	"context"
	"testing"

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
