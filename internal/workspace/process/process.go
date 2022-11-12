package process

import (
	"context"
	"fmt"

	"github.com/slok/tfe-drift/internal/model"
)

// Processor knows how to process a list of workspaces.
type Processor interface {
	Process(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error)
}

type ProcessorFunc func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error)

func (p ProcessorFunc) Process(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
	return p(ctx, wks)
}

// NewProcessorChain returns a processor that knows how to execute a chain of processors.
func NewProcessorChain(ps []Processor) Processor {
	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		var err error
		for _, p := range ps {
			wks, err = p.Process(ctx, wks)
			if err != nil {
				return nil, fmt.Errorf("processor failed: %w", err)
			}
		}

		return wks, nil
	})
}

type noopProcessor bool

func (noopProcessor) Process(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
	return wks, nil
}

// NoopProcessor doesn't do anything.
const NoopProcessor = noopProcessor(false)
