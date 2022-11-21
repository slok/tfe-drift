package commands

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/hashicorp/go-tfe"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/slok/tfe-drift/internal/controller"
	"github.com/slok/tfe-drift/internal/log"
	internalprometheus "github.com/slok/tfe-drift/internal/metrics/prometheus"
	tfestorage "github.com/slok/tfe-drift/internal/storage/tfe"
	"github.com/slok/tfe-drift/internal/workspace/process"
	wksprocess "github.com/slok/tfe-drift/internal/workspace/process"
)

type ControllerCommand struct {
	cmd        *kingpin.CmdClause
	rootConfig *RootCommand

	planMessage          string
	includeNameRegexes   []string
	excludeNameRegexes   []string
	includeTags          []string
	excludeTags          []string
	notBefore            time.Duration
	maxPlans             int
	waitTimeout          time.Duration
	dryRun               bool
	detectInterval       time.Duration
	disableDriftDetector bool
	metricsTimeout       time.Duration
	ListenAddress        string
	MetricsPath          string
	HealthCheckPath      string
	PprofPath            string
}

// NewControllerCommand returns the Controller command.
func NewControllerCommand(rootConfig *RootCommand, app *kingpin.Application) *ControllerCommand {
	cmd := app.Command("controller", "Runs drift detector in controller mode.")
	c := &ControllerCommand{
		cmd:        cmd,
		rootConfig: rootConfig,
	}

	cmd.Flag("plan-message", "Message to set on the executed drift detection plans.").Short('m').Default("Drift detection").StringVar(&c.planMessage)
	cmd.Flag("include-name", "Regex that if matches workspace name it will be included in the drift detection (can be repeated or comma separated).").Short('i').StringsVar(&c.includeNameRegexes)
	cmd.Flag("exclude-name", "Regex that if matches workspace name it will be excluded from the drift detection (can be repeated or comma separated).").Short('e').StringsVar(&c.excludeNameRegexes)
	cmd.Flag("include-tag", "The workspaces that match the tag will be included (can be repeated or comma separated).").Short('t').StringsVar(&c.includeTags)
	cmd.Flag("exclude-tag", "The workspaces that match the tag will be excluded (can be repeated or comma separated).").Short('x').StringsVar(&c.excludeTags)
	cmd.Flag("limit-max-plans", "The maximum drift detection plans that will be executed.").Short('l').Default("1").IntVar(&c.maxPlans)
	cmd.Flag("not-before", "Will filter the workspaces that executed a drift detection plan before before this duration.").Short('n').Default("1h").DurationVar(&c.notBefore)
	cmd.Flag("wait-timeout", "Max time duration to wait for drift detection plans to finish.").Default("1h").DurationVar(&c.waitTimeout)
	cmd.Flag("dry-run", "Will execute all the process without creating any drift detection plans, will use latest ones available.").BoolVar(&c.dryRun)
	cmd.Flag("detect-interval", "The interval that the app will run a drift detection.").Default("5m").DurationVar(&c.detectInterval)
	cmd.Flag("disable-drift-detector", "Will disable the drift detector, this can be useful when you want ot run only the metrics exporter.").BoolVar(&c.disableDriftDetector)
	cmd.Flag("metrics-exporter-timeout", "Duration timeout used for the prometheus exporter metrics collector.").Default("45s").DurationVar(&c.metricsTimeout)
	cmd.Flag("listen-address", "The address where the will be listening.").Default(":8080").StringVar(&c.ListenAddress)
	cmd.Flag("metrics-path", "The path where Prometheus metrics will be served.").Default("/metrics").StringVar(&c.MetricsPath)
	cmd.Flag("health-check-path", "The path where the health check will be served.").Default("/status").StringVar(&c.HealthCheckPath)
	cmd.Flag("pprof-path", "The path where the pprof handlers will be served.").Default("/debug/pprof").StringVar(&c.PprofPath)

	return c
}

func (c ControllerCommand) Name() string { return c.cmd.FullCommand() }
func (c ControllerCommand) Run(ctx context.Context) error {
	logger := c.rootConfig.Logger
	notVerboseLogger := infoAsDebugLogger{Logger: logger}

	if len(c.excludeNameRegexes) > 0 && len(c.includeNameRegexes) > 0 {
		return fmt.Errorf("include and exclude name options can't be used at the same time")
	}

	if len(c.includeTags) > 0 && len(c.excludeTags) > 0 {
		return fmt.Errorf("include and exclude tag options can't be used at the same time")
	}

	// Sanitize names and tags by splitting using commas.
	const repeatedArgSplitChar = ","
	excludeNameRegexes := splitRepeatedArg(c.excludeNameRegexes, repeatedArgSplitChar)
	includeNameRegexes := splitRepeatedArg(c.includeNameRegexes, repeatedArgSplitChar)
	includeTags := splitRepeatedArg(c.includeTags, repeatedArgSplitChar)
	excludeTags := splitRepeatedArg(c.excludeTags, repeatedArgSplitChar)

	config := &tfe.Config{
		Token:   c.rootConfig.TFEToken,
		Address: c.rootConfig.TFEAddress,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		return err
	}

	// Prepare processor chain.
	repoTFEClient := tfestorage.NewClient(client)
	repo, err := tfestorage.NewRepository(repoTFEClient, c.rootConfig.TFEOrg, c.rootConfig.TFEAddress, c.rootConfig.AppID)
	if err != nil {
		return fmt.Errorf("could not create tfe storage repository: %w", err)
	}

	if c.dryRun {
		repo = tfestorage.NewDryRunRepository(notVerboseLogger, repo)
	}

	var includeProcessor process.Processor = process.NoopProcessor
	if len(includeNameRegexes) > 0 {
		p, err := wksprocess.NewIncludeNameProcessor(notVerboseLogger, includeNameRegexes)
		if err != nil {
			return fmt.Errorf("invalid include processor: %w", err)
		}
		includeProcessor = p
	}

	var excludeProcessor process.Processor = process.NoopProcessor
	if len(excludeNameRegexes) > 0 {
		p, err := wksprocess.NewExcludeNameProcessor(notVerboseLogger, excludeNameRegexes)
		if err != nil {
			return fmt.Errorf("invalid exclude processor: %w", err)
		}
		excludeProcessor = p
	}

	var g run.Group

	// Controller.
	if c.disableDriftDetector {
		logger.Infof("Drift detector controller disabled")
	} else {
		chain := wksprocess.NewProcessorChain([]wksprocess.Processor{
			includeProcessor,
			excludeProcessor,
			wksprocess.NewHydrateLatestDetectionPlanProcessor(ctx, notVerboseLogger, repo),
			wksprocess.NewFilterQueuedDriftDetectorProcessor(notVerboseLogger),
			wksprocess.NewFilterDriftDetectionsBeforeProcessor(notVerboseLogger, c.notBefore),
			wksprocess.NewSortByOldestDetectionPlanProcessor(notVerboseLogger),
			wksprocess.NewLimitMaxProcessor(notVerboseLogger, c.maxPlans),
			wksprocess.NewDriftDetectionPlanProcessor(notVerboseLogger, repo, c.planMessage),
			wksprocess.NewDriftDetectionPlanWaitProcessor(notVerboseLogger, repo, waitPolling, c.waitTimeout),
		})

		ctrl, err := controller.NewDriftDetector(controller.DriftDetectorConfig{
			Logger:             logger,
			Interval:           c.detectInterval,
			WorkspaceLister:    repo,
			WorkspaceProcessor: chain,
			IncludeTags:        includeTags,
			ExcludeTags:        excludeTags,
		})
		if err != nil {
			return fmt.Errorf("controller drift detector could not be created: %w", err)
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		g.Add(
			func() error {
				err = ctrl.Run(ctx)
				if err != nil {
					return fmt.Errorf("controller drift detector had an error: %w", err)
				}

				return nil
			},
			func(_ error) {
				cancel()
			},
		)
	}

	// Serving HTTP server.
	{
		chain := wksprocess.NewProcessorChain([]wksprocess.Processor{
			includeProcessor,
			excludeProcessor,
			wksprocess.NewHydrateLatestDetectionPlanProcessor(ctx, notVerboseLogger, repo),
		})

		// Register metrics collector to create the exporter.
		promCollector := internalprometheus.NewCollector(logger, repo, chain, includeTags, excludeTags, c.metricsTimeout)
		prometheus.DefaultRegisterer.MustRegister(promCollector)

		logger := logger.WithValues(log.Kv{
			"addr":         c.ListenAddress,
			"metrics":      c.MetricsPath,
			"health-check": c.HealthCheckPath,
			"pprof":        c.PprofPath,
		})
		mux := http.NewServeMux()

		// Metrics.
		mux.Handle(c.MetricsPath, promhttp.Handler())

		// Pprof.
		mux.HandleFunc(c.PprofPath+"/", pprof.Index)
		mux.HandleFunc(c.PprofPath+"/cmdline", pprof.Cmdline)
		mux.HandleFunc(c.PprofPath+"/profile", pprof.Profile)
		mux.HandleFunc(c.PprofPath+"/symbol", pprof.Symbol)
		mux.HandleFunc(c.PprofPath+"/trace", pprof.Trace)

		// Health check.
		mux.Handle(c.HealthCheckPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"status":"ok"}`)) }))

		// Create server.
		server := &http.Server{
			Addr:    c.ListenAddress,
			Handler: mux,
		}

		g.Add(
			func() error {
				logger.Infof("HTTP server listening for requests")
				return server.ListenAndServe()
			},
			func(_ error) {
				logger.Infof("HTTP server shutdown, draining connections...")

				ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)
				if err != nil {
					logger.Errorf("Error shutting down server: %w", err)
				}

				logger.Infof("Connections drained")
			},
		)
	}

	// In case we are stopped from the upper level context.
	{
		g.Add(
			func() error {
				<-ctx.Done()
				return nil
			},
			func(_ error) {},
		)
	}

	return g.Run()
}

// infoAsDebugLogger is a logger that will be used when we have reusable components that
// in some cases we want them verbose and others not.
type infoAsDebugLogger struct {
	log.Logger
}

func (i infoAsDebugLogger) Infof(format string, args ...any) {
	i.Logger.Debugf(format, args...)
}

func (i infoAsDebugLogger) WithValues(kv log.Kv) log.Logger {
	return infoAsDebugLogger{Logger: i.Logger.WithValues(kv)}
}

func (i infoAsDebugLogger) WithCtxValues(ctx context.Context) log.Logger {
	return infoAsDebugLogger{Logger: i.Logger.WithCtxValues(ctx)}
}
