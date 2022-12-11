package prometheus_test

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/slok/tfe-drift/internal/log"
	internalprometheus "github.com/slok/tfe-drift/internal/metrics/prometheus"
	"github.com/slok/tfe-drift/internal/metrics/prometheus/prometheusmock"
	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/workspace/process"
)

func TestCollector(t *testing.T) {
	t0, _ := time.Parse(time.RFC3339, "2022-11-21T17:43:53+00:00")

	tests := map[string]struct {
		mock           func(mr *prometheusmock.Repository)
		expMetrics     string
		expMetricNames []string
	}{
		"No workspaces shouldn't return any metric.": {
			mock: func(mr *prometheusmock.Repository) {
				wks := []model.Workspace{}
				mr.On("ListWorkspaces", mock.Anything, mock.Anything, mock.Anything).Once().Return(wks, nil)
			},
			expMetrics:     ``,
			expMetricNames: []string{""},
		},

		"Having workspaces should return metrics.": {
			mock: func(mr *prometheusmock.Repository) {
				wks := []model.Workspace{
					{Name: "test1", ID: "test-id-1", Tags: []string{"t1a", "t1b"}, Org: "test-org", LastDriftPlan: &model.Plan{
						ID:         "test-run1",
						URL:        "https://test-run1.dev",
						Status:     model.PlanStatusFinishedOK,
						HasChanges: true,
						CreatedAt:  t0,
						FinishedAt: t0.Add(10 * time.Second),
					}},
					{Name: "test2", ID: "test-id-2", Tags: []string{"t2d", "t2c"}, Org: "test-org", LastDriftPlan: &model.Plan{
						ID:         "test-run2",
						URL:        "https://test-run2.dev",
						Status:     model.PlanStatusFinishedNotOK,
						HasChanges: false,
						CreatedAt:  t0.Add(55 * time.Second),
						FinishedAt: t0.Add(72 * time.Second),
					}},
					{Name: "test3", ID: "test-id-3", Tags: []string{"t3c", "t3b", "t3a"}, Org: "test-org", LastDriftPlan: &model.Plan{
						ID:         "test-run3",
						URL:        "https://test-run3.dev",
						Status:     model.PlanStatusFinishedOK,
						HasChanges: false,
						CreatedAt:  t0.Add(120 * time.Second),
						FinishedAt: t0.Add(145 * time.Second),
					}},
				}
				mr.On("ListWorkspaces", mock.Anything, mock.Anything, mock.Anything).Once().Return(wks, nil)
			},
			expMetrics: `
# HELP tfe_drift_workspace_drift_detection_create Unix epoch timestamp when the drift detection was created.
# TYPE tfe_drift_workspace_drift_detection_create gauge
tfe_drift_workspace_drift_detection_create{workspace_name="test1"} 1.669052633e+09
tfe_drift_workspace_drift_detection_create{workspace_name="test2"} 1.669052688e+09
tfe_drift_workspace_drift_detection_create{workspace_name="test3"} 1.669052753e+09

# HELP tfe_drift_workspace_drift_detection_finish Unix epoch timestamp when the drift detection ended.
# TYPE tfe_drift_workspace_drift_detection_finish gauge
tfe_drift_workspace_drift_detection_finish{workspace_name="test1"} 1.669052643e+09
tfe_drift_workspace_drift_detection_finish{workspace_name="test2"} 1.669052705e+09
tfe_drift_workspace_drift_detection_finish{workspace_name="test3"} 1.669052778e+09

# HELP tfe_drift_workspace_drift_detection_state The state of a workspaces drift detection.
# TYPE tfe_drift_workspace_drift_detection_state gauge
tfe_drift_workspace_drift_detection_state{state="drift",workspace_name="test1"} 1
tfe_drift_workspace_drift_detection_state{state="drift",workspace_name="test2"} 0
tfe_drift_workspace_drift_detection_state{state="drift",workspace_name="test3"} 0
tfe_drift_workspace_drift_detection_state{state="drift_plan_error",workspace_name="test1"} 0
tfe_drift_workspace_drift_detection_state{state="drift_plan_error",workspace_name="test2"} 1
tfe_drift_workspace_drift_detection_state{state="drift_plan_error",workspace_name="test3"} 0
tfe_drift_workspace_drift_detection_state{state="ok",workspace_name="test1"} 0
tfe_drift_workspace_drift_detection_state{state="ok",workspace_name="test2"} 0
tfe_drift_workspace_drift_detection_state{state="ok",workspace_name="test3"} 1

# HELP tfe_drift_workspace_info Information of the workspace.
# TYPE tfe_drift_workspace_info gauge
tfe_drift_workspace_info{organization_name="test-org",run_id="test-run1",run_url="https://test-run1.dev",tags="t1a,t1b",workspace_id="test-id-1",workspace_name="test1"} 1
tfe_drift_workspace_info{organization_name="test-org",run_id="test-run2",run_url="https://test-run2.dev",tags="t2c,t2d",workspace_id="test-id-2",workspace_name="test2"} 1
tfe_drift_workspace_info{organization_name="test-org",run_id="test-run3",run_url="https://test-run3.dev",tags="t3a,t3b,t3c",workspace_id="test-id-3",workspace_name="test3"} 1
`,
			expMetricNames: []string{
				"tfe_drift_workspace_drift_detection_state",
				"tfe_drift_workspace_info",
				"tfe_drift_workspace_drift_detection_create",
				"tfe_drift_workspace_drift_detection_finish",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			mr := prometheusmock.NewRepository(t)
			test.mock(mr)

			// Create collector.
			c := internalprometheus.NewCollector(log.Noop, mr, process.NoopProcessor, nil, nil, 1*time.Second)

			// Register exporter.
			reg := prometheus.NewRegistry()
			reg.MustRegister(c)

			// Check metrics.
			err := testutil.GatherAndCompare(reg, strings.NewReader(test.expMetrics), test.expMetricNames...)
			assert.NoError(err)
		})
	}
}
