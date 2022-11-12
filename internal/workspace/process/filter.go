package process

import (
	"context"
	"fmt"
	"regexp"

	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

func NewIncludeNameProcessor(logger log.Logger, regexes []string) (Processor, error) {
	// If no regex, then match all.
	if len(regexes) == 0 {
		return NoopProcessor, nil
	}

	logger = logger.WithValues(log.Kv{"workspace-processor": "IncludeNameProcessor"})
	rxs, err := compileRegexes(regexes)
	if err != nil {
		return nil, fmt.Errorf("invalid regexes: %w", err)
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Including workspaces by name")

		newWks := []model.Workspace{}
		for _, wk := range wks {
			if !matchStringRegexes(rxs, wk.Name) {
				logger.WithValues(log.Kv{"workspace": wk.ID}).Debugf("Ignoring workspace")
				continue
			}

			newWks = append(newWks, wk)
		}

		return newWks, nil
	}), nil
}

func NewExcludeNameProcessor(logger log.Logger, regexes []string) (Processor, error) {
	logger = logger.WithValues(log.Kv{"workspace-processor": "ExcludeNameProcessor"})
	rxs, err := compileRegexes(regexes)
	if err != nil {
		return nil, fmt.Errorf("invalid regexes: %w", err)
	}

	return ProcessorFunc(func(ctx context.Context, wks []model.Workspace) ([]model.Workspace, error) {
		logger.Infof("Exluding workspaces by name")

		newWks := []model.Workspace{}
		for _, wk := range wks {
			if matchStringRegexes(rxs, wk.Name) {
				logger.WithValues(log.Kv{"workspace": wk.ID}).Debugf("Ignoring workspace")
				continue
			}

			newWks = append(newWks, wk)
		}

		return newWks, nil
	}), nil
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
