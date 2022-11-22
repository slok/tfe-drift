package prometheus

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/slok/tfe-drift/internal/info"
	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
)

type Repository interface {
	ListWorkspaces(ctx context.Context, includeTags, excludeTags []string) ([]model.Workspace, error)
	GetLatestCheckPlan(ctx context.Context, w model.Workspace) (*model.Plan, error)
}

//go:generate mockery --case underscore --output prometheusmock --outpkg prometheusmock --name Repository

const (
	stateOk             = "ok"
	stateDrift          = "drift"
	stateDriftPlanError = "drift_plan_error"
)

type collector struct {
	repo        Repository
	wkProcessor process.Processor
	includeTags []string
	excludeTags []string
	logger      log.Logger
	timeout     time.Duration

	stateDesc    *prometheus.Desc
	infoDesc     *prometheus.Desc
	createdDesc  *prometheus.Desc
	finishedDesc *prometheus.Desc
}

func NewCollector(logger log.Logger, repo Repository, wkProcessor process.Processor, includeTags []string, excludeTags []string, timeout time.Duration) prometheus.Collector {
	return collector{
		repo:        repo,
		wkProcessor: wkProcessor,
		includeTags: includeTags,
		excludeTags: excludeTags,
		logger:      logger,
		timeout:     timeout,

		stateDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "drift_detection_state"),
			"The state of a workspaces drift detection.",
			[]string{"workspaces_name", "state"}, nil,
		),
		infoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "info"),
			"Information of the workspace.",
			[]string{"workspaces_name", "workspaces_id", "run_id", "run_url", "tags"}, nil,
		),
		createdDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "drift_detection_create"),
			"Unix epoch timestamp when the drift detection was created.",
			[]string{"workspaces_name"}, nil,
		),
		finishedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "drift_detection_finish"),
			"Unix epoch timestamp when the drift detection ended.",
			[]string{"workspaces_name"}, nil,
		),
	}
}

func (c collector) Describe(ch chan<- *prometheus.Desc) {}
func (c collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Debugf("Collection started")
	defer c.logger.Debugf("Collection finished")

	// Add timeout.
	ctx := context.Background()
	if c.timeout != 0 {
		c, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		ctx = c
	}

	// Collect and add the collected metrics in a goroutine
	// That way we can wait until done or timeout.
	resC := make(chan error)
	go func() {
		metrics, err := c.collect(ctx)
		if err == nil {
			for _, metric := range metrics {
				ch <- metric
			}
		}

		resC <- err
	}()

	// Wait for result or timeout.
	select {
	case <-ctx.Done():
		c.logger.Errorf("Context done: %s", ctx.Err())
	case err := <-resC:
		if err != nil {
			c.logger.Errorf("Collection error: %s", ctx.Err())
		}
	}
}

func (c collector) collect(ctx context.Context) ([]prometheus.Metric, error) {
	wks, err := c.repo.ListWorkspaces(ctx, c.includeTags, c.excludeTags)
	if err != nil {
		return nil, fmt.Errorf("could not list workspaces: %w", err)
	}

	wks, err = c.wkProcessor.Process(ctx, wks)
	if err != nil {
		return nil, fmt.Errorf("could not process workspaces: %w", err)
	}

	metrics := []prometheus.Metric{}
	for _, wk := range wks {
		okValue := 0
		driftValue := 0
		driftPlanErrorValue := 0

		switch {
		case wk.LastDriftPlan == nil:
			continue
		case wk.LastDriftPlan.Status == model.PlanStatusFinishedOK && wk.LastDriftPlan.HasChanges:
			driftValue = 1
		case wk.LastDriftPlan.Status == model.PlanStatusFinishedNotOK:
			driftPlanErrorValue = 1
		case wk.LastDriftPlan.Status == model.PlanStatusFinishedOK:
			okValue = 1
		default:
			continue
		}

		tags := wk.Tags
		sort.Strings(tags)
		tagsLabel := strings.Join(tags, ",")

		metrics = append(metrics,
			// Write all state metrics setting 1 to the states we are in, 0 on the others.
			prometheus.MustNewConstMetric(c.stateDesc, prometheus.GaugeValue, float64(okValue), wk.Name, stateOk),
			prometheus.MustNewConstMetric(c.stateDesc, prometheus.GaugeValue, float64(driftValue), wk.Name, stateDrift),
			prometheus.MustNewConstMetric(c.stateDesc, prometheus.GaugeValue, float64(driftPlanErrorValue), wk.Name, stateDriftPlanError),

			// Info metric.
			prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue, 1, wk.Name, wk.ID, wk.LastDriftPlan.ID, wk.LastDriftPlan.URL, tagsLabel),

			// Timestamps.
			prometheus.MustNewConstMetric(c.createdDesc, prometheus.GaugeValue, float64(wk.LastDriftPlan.CreatedAt.Unix()), wk.Name),
			prometheus.MustNewConstMetric(c.finishedDesc, prometheus.GaugeValue, float64(wk.LastDriftPlan.FinishedAt.Unix()), wk.Name),
		)
	}

	return metrics, nil
}
