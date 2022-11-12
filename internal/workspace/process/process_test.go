package process_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
)

func TestProcessorChain(t *testing.T) {
	tests := map[string]struct {
		workspaces    func() []model.Workspace
		processors    func() []process.Processor
		expWorkspaces func() []model.Workspace
		expErr        bool
	}{
		"No processors chain shouldn't fail": {
			workspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}}
			},
			processors: func() []process.Processor {
				return []process.Processor{}
			},
			expWorkspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}}
			},
		},

		"Processors that mutate the workspaces by aggregating should work correctly.": {
			workspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}}
			},
			processors: func() []process.Processor {
				return []process.Processor{
					process.ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
						return append(wks, model.Workspace{ID: "test3"}, model.Workspace{ID: "test4"}), nil
					}),
				}
			},
			expWorkspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}, {ID: "test3"}, {ID: "test4"}}
			},
		},

		"Processors that mutate the workspaces by reducing should work correctly.": {
			workspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}}
			},
			processors: func() []process.Processor {
				return []process.Processor{
					process.ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
						return wks[:len(wks)-1], nil
					}),
				}
			},
			expWorkspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}}
			},
		},

		"Processors that mutate the workspaces should work correctly.": {
			workspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}}
			},
			processors: func() []process.Processor {
				return []process.Processor{
					process.ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
						newWks := []model.Workspace{}
						for _, wk := range wks {
							wk.ID = wk.ID + "-mutated"
							newWks = append(newWks, wk)
						}
						return newWks, nil
					}),
				}
			},
			expWorkspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1-mutated"}, {ID: "test2-mutated"}}
			},
		},

		"Multiple processors should work correctly.": {
			workspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}}
			},
			processors: func() []process.Processor {
				return []process.Processor{
					process.ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
						return append(wks, model.Workspace{ID: "test3"}), nil
					}),
					process.ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
						return append(wks, model.Workspace{ID: "test4"}), nil
					}),
					process.ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
						return append(wks, model.Workspace{ID: "test5"}), nil
					}),
				}
			},
			expWorkspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}, {ID: "test3"}, {ID: "test4"}, {ID: "test5"}}
			},
		},

		"A chain that fails should stop fail.": {
			workspaces: func() []model.Workspace {
				return []model.Workspace{{ID: "test1"}, {ID: "test2"}}
			},
			processors: func() []process.Processor {
				return []process.Processor{
					process.ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
						return nil, fmt.Errorf("something")
					}),
				}
			},
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			pc := process.NewProcessorChain(test.processors())
			gotWks, err := pc.Process(context.TODO(), test.workspaces())

			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expWorkspaces(), gotWks)
			}
		})
	}
}
