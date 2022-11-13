package process

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

func NewIncludeNameProcessor(logger log.Logger, regexes []string) (Processor, error) {
	// If no regex, then match all.
	if len(regexes) == 0 {
		return NoopProcessor, nil
	}

	logger = logger.WithValues(log.Kv{"workspace-processor": "IncludeName"})
	rxs, err := compileRegexes(regexes)
	if err != nil {
		return nil, fmt.Errorf("invalid regexes: %w", err)
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Including workspaces by name")

		newWks := []model.Workspace{}
		for _, wk := range wks {
			if !matchStringRegexes(rxs, wk.Name) {
				logger.WithValues(log.Kv{"workspace": wk.Name}).Debugf("Ignoring workspace, excluded by name")
				continue
			}

			newWks = append(newWks, wk)
		}

		return newWks, nil
	}), nil
}

func NewExcludeNameProcessor(logger log.Logger, regexes []string) (Processor, error) {
	logger = logger.WithValues(log.Kv{"workspace-processor": "ExcludeName"})
	rxs, err := compileRegexes(regexes)
	if err != nil {
		return nil, fmt.Errorf("invalid regexes: %w", err)
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Exluding workspaces by name")

		newWks := []model.Workspace{}
		for _, wk := range wks {
			if matchStringRegexes(rxs, wk.Name) {
				logger.WithValues(log.Kv{"workspace": wk.Name}).Debugf("Ignoring workspace, excluded by name")
				continue
			}

			newWks = append(newWks, wk)
		}

		return newWks, nil
	}), nil
}

func NewLimitMaxProcessor(logger log.Logger, max int) Processor {
	// If 0, then no limit.
	if max == 0 {
		return NoopProcessor
	}

	logger = logger.WithValues(log.Kv{"workspace-processor": "LimitMax"})
	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Limiting max drift plan detections")
		if max >= len(wks) {
			return wks, nil
		}

		return wks[:max], nil
	})
}

func NewFilterQueuedDriftDetectorProcessor(logger log.Logger) Processor {
	logger = logger.WithValues(log.Kv{"workspace-processor": "FilterQueuedDriftDetector"})
	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Filtering already queued drift detection plans")

		newWks := []model.Workspace{}
		for _, wk := range wks {
			if wk.LastDriftPlan != nil && wk.LastDriftPlan.Status == model.PlanStatusWaiting {
				logger.WithValues(log.Kv{"workspace": wk.Name}).Debugf("Ignoring workspace, drift detection already queued")
				continue
			}

			newWks = append(newWks, wk)
		}

		return newWks, nil
	})
}

func NewFilterDriftDetectionsBeforeProcessor(logger log.Logger, notBefore time.Duration) Processor {
	// If 0, then no filter.
	if notBefore == 0 {
		return NoopProcessor
	}

	logger = logger.WithValues(log.Kv{"workspace-processor": "FilterDriftDetectionsBefore"})
	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Filtering drift plan detections executed before %s", notBefore)

		newWks := []model.Workspace{}
		for _, wk := range wks {
			if wk.LastDriftPlan != nil && time.Since(wk.LastDriftPlan.CreatedAt) < notBefore {
				logger.WithValues(log.Kv{"workspace": wk.Name}).Debugf("Ignoring workspace, last drift detection plan was %s (min %s)", time.Since(wk.LastDriftPlan.CreatedAt), notBefore)
				continue
			}

			newWks = append(newWks, wk)
		}

		return newWks, nil
	})
}

func compileRegexes(regexes []string) ([]*regexp.Regexp, error) {
	rxs := []*regexp.Regexp{}
	for _, r := range regexes {
		rx, err := regexp.Compile(r)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
		rxs = append(rxs, rx)
	}

	return rxs, nil
}

func matchStringRegexes(rxs []*regexp.Regexp, s string) bool {
	for _, rx := range rxs {
		if rx.MatchString(s) {
			return true
		}
	}

	return false
}
