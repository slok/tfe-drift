package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	wkprocess "github.com/slok/tfe-drift/internal/workspace/process"
)

type WorkspaceLister interface {
	ListWorkspaces(ctx context.Context, includeTags, excludeTags []string) ([]model.Workspace, error)
}

type DriftDetectorConfig struct {
	Logger             log.Logger
	Interval           time.Duration
	WorkspaceLister    WorkspaceLister
	WorkspaceProcessor wkprocess.Processor
	IncludeTags        []string
	ExcludeTags        []string
}

func (c *DriftDetectorConfig) defaults() error {
	if c.Interval == 0 {
		return fmt.Errorf("interval can't be 0")
	}

	if c.WorkspaceLister == nil {
		return fmt.Errorf("workspace lister is required")
	}

	if c.WorkspaceProcessor == nil {
		return fmt.Errorf("workspace processor is required")
	}

	if c.Logger == nil {
		c.Logger = log.Noop
	}
	c.Logger = c.Logger.WithValues(log.Kv{"svc": "interval.DriftDetector"})

	return nil
}

type DriftDetector struct {
	logger      log.Logger
	interval    time.Duration
	wkLister    WorkspaceLister
	wprocessor  wkprocess.Processor
	includeTags []string
	excludeTags []string
}

func NewDriftDetector(config DriftDetectorConfig) (*DriftDetector, error) {
	err := config.defaults()
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &DriftDetector{
		logger:      config.Logger,
		interval:    config.Interval,
		wkLister:    config.WorkspaceLister,
		wprocessor:  config.WorkspaceProcessor,
		includeTags: config.IncludeTags,
		excludeTags: config.ExcludeTags,
	}, nil
}

func (d DriftDetector) Run(ctx context.Context) error {
	t := time.NewTicker(d.interval)

	// We run this once outside the loop so we don't wait for the first tick.
	d.logger.Infof("Drift detection started")
	err := d.run(ctx)
	if err != nil {
		d.logger.Errorf("Drift detection failed: %s", err)
	} else {
		d.logger.Infof("Drift detection finished")
	}

	for {
		select {
		case <-ctx.Done():
			d.logger.Infof("Stopping controller...")
			return ctx.Err()
		case <-t.C:
			d.logger.Infof("Drift detection started")

			err := d.run(ctx)
			if err != nil {
				d.logger.Errorf("Drift detection failed: %s", err)
			} else {
				d.logger.Infof("Drift detection finished")
			}
		}
	}
}

func (d DriftDetector) run(ctx context.Context) error {
	wks, err := d.wkLister.ListWorkspaces(ctx, d.includeTags, d.excludeTags)
	if err != nil {
		return fmt.Errorf("could not list workspaces: %w", err)
	}

	if len(wks) == 0 {
		d.logger.Warningf("0 workspaces selected")
		return nil
	}

	_, err = d.wprocessor.Process(ctx, wks)
	if err != nil {
		return fmt.Errorf("workspaces processing failed: %w", err)
	}

	return nil
}
