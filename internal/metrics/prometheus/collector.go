package prometheus

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/slok/tfe-drift/internal/info"
	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
)

type WorkspaceRepository interface {
	ListWorkspaces(ctx context.Context, includeTags, excludeTags []string) ([]model.Workspace, error)
}

//go:generate mockery --case underscore --output prometheusmock --outpkg prometheusmock --name WorkspaceRepository

const (
	stateOk             = "ok"
	stateDrift          = "drift"
	stateDriftPlanError = "drift_plan_error"
)

type collector struct {
	repo        WorkspaceRepository
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

func NewCollector(ctx context.Context, logger log.Logger, repo WorkspaceRepository, wkProcessor process.Processor, includeTags []string, excludeTags []string, timeout time.Duration) (prometheus.Collector, error) {
	const paceSeconds = 75
	asyncRepo, err := newAsyncWorkspaceRepository(ctx, logger, repo, paceSeconds*time.Second, includeTags, excludeTags)
	if err != nil {
		return nil, err
	}

	return collector{
		repo:        asyncRepo,
		wkProcessor: wkProcessor,
		includeTags: includeTags,
		excludeTags: excludeTags,
		logger:      logger,
		timeout:     timeout,

		stateDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "drift_detection_state"),
			"The state of a workspaces drift detection.",
			[]string{"workspace_name", "state"}, nil,
		),
		infoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "info"),
			"Information of the workspace.",
			[]string{"workspace_name", "workspace_id", "run_id", "run_url", "tags", "organization_name"}, nil,
		),
		createdDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "drift_detection_create"),
			"Unix epoch timestamp when the drift detection was created.",
			[]string{"workspace_name"}, nil,
		),
		finishedDesc: prometheus.NewDesc(
			prometheus.BuildFQName(info.PrometheusNamespace, "workspace", "drift_detection_finish"),
			"Unix epoch timestamp when the drift detection ended.",
			[]string{"workspace_name"}, nil,
		),
	}, nil
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
			prometheus.MustNewConstMetric(c.infoDesc, prometheus.GaugeValue, 1, wk.Name, wk.ID, wk.LastDriftPlan.ID, wk.LastDriftPlan.URL, tagsLabel, wk.Org),

			// Timestamps.
			prometheus.MustNewConstMetric(c.createdDesc, prometheus.GaugeValue, float64(wk.LastDriftPlan.CreatedAt.Unix()), wk.Name),
			prometheus.MustNewConstMetric(c.finishedDesc, prometheus.GaugeValue, float64(wk.LastDriftPlan.FinishedAt.Unix()), wk.Name),
		)
	}

	return metrics, nil
}

// asyncWorkspaceRepository is a repository that will retrieve the workspaces asynchronously.
//
// This is because retrieving all workspaces takes time and don't change that much and the
// dynamix data of the plans, its hydrated by the processors, this way we will save time
// retrieving workspaces and calls, and return up to date data on the more relevant and
// dynamic data.
type asyncWorkspaceRepository struct {
	includeTagsIndex string
	includeTags      []string
	excludeTagsIndex string
	excludeTags      []string
	r                WorkspaceRepository
	logger           log.Logger
	cache            []model.Workspace
	mu               sync.RWMutex
}

func newAsyncWorkspaceRepository(ctx context.Context, logger log.Logger, r WorkspaceRepository, pace time.Duration, includeTags, excludeTags []string) (WorkspaceRepository, error) {
	ar := &asyncWorkspaceRepository{
		includeTagsIndex: fmt.Sprintf("%v", includeTags),
		includeTags:      includeTags,
		excludeTagsIndex: fmt.Sprintf("%v", excludeTags),
		excludeTags:      excludeTags,
		r:                r,
		logger:           logger,
	}

	// Fill cache for first time.
	wks, err := r.ListWorkspaces(ctx, includeTags, excludeTags)
	if err != nil {
		return nil, fmt.Errorf("could not list workspaces to fill repository cache")
	}
	ar.cache = wks

	// Start workspace async retrieval polling.
	go ar.poll(ctx, pace)

	return ar, nil
}

func (a *asyncWorkspaceRepository) poll(ctx context.Context, pace time.Duration) {
	t := time.NewTicker(pace)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.logger.Debugf("Async workspaces list triggered")
			a.mu.Lock()
			wk, err := a.r.ListWorkspaces(ctx, a.includeTags, a.excludeTags)
			if err != nil {
				a.logger.Errorf("Error retrieving async workspaces: %w", err)
			} else {
				a.cache = wk
			}
			a.mu.Unlock()
		}
	}
}

func (a *asyncWorkspaceRepository) ListWorkspaces(ctx context.Context, includeTags, excludeTags []string) ([]model.Workspace, error) {
	if a.includeTagsIndex != fmt.Sprintf("%v", includeTags) {
		return nil, fmt.Errorf("the include tags are different from the ones used for the cache")
	}

	if a.excludeTagsIndex != fmt.Sprintf("%v", excludeTags) {
		return nil, fmt.Errorf("the exclude tags are different from the ones used for the cache")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.cache, nil
}
