package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-tfe"
	"gopkg.in/alecthomas/kingpin.v2"

	tfestorage "github.com/slok/tfe-drift/internal/storage/tfe"
	"github.com/slok/tfe-drift/internal/workspace/process"
	wksprocess "github.com/slok/tfe-drift/internal/workspace/process"
)

var (
	waitPolling = 15 * time.Second

	outFormatJSON = "json"
)

type RunCommand struct {
	cmd        *kingpin.CmdClause
	rootConfig *RootCommand

	planMessage               string
	includeNameRegexes        []string
	excludeNameRegexes        []string
	notBefore                 time.Duration
	maxPlans                  int
	waitTimeout               time.Duration
	disableDriftPlanExitCodes bool
	outFormat                 string
	dryRun                    bool
}

// NewRunCommand returns the Run command.
func NewRunCommand(rootConfig *RootCommand, app *kingpin.Application) *RunCommand {
	cmd := app.Command("run", "Runs drift detections.")
	c := &RunCommand{
		cmd:        cmd,
		rootConfig: rootConfig,
	}

	cmd.Flag("plan-message", "Message to set on the executed drift detection plans.").Short('m').Default("Drift detection").StringVar(&c.planMessage)
	cmd.Flag("include-name", "Regex that if matches workspace name it will be included in the drift detection (can be repeated).").Short('i').StringsVar(&c.includeNameRegexes)
	cmd.Flag("exclude-name", "Regex that if matches workspace name it will be excluded from the drift detection (can be repeated).").Short('e').StringsVar(&c.excludeNameRegexes)
	cmd.Flag("limit-max-plans", "The maximum drift detection plans that will be executed.").Short('l').IntVar(&c.maxPlans)
	cmd.Flag("not-before", "Will filter the workspaces that executed a drift detection plan before before this duration.").Short('n').Default("1h").DurationVar(&c.notBefore)
	cmd.Flag("wait-timeout", "Max time duration to wait for drift detection plans to finish.").Default("2h").DurationVar(&c.waitTimeout)
	cmd.Flag("disable-drift-plan-exitcodes", "Will disable the drift detection plans related exit codes (2 and 3).").BoolVar(&c.disableDriftPlanExitCodes)
	cmd.Flag("out-format", "Selects the format of the result output.").Short('o').EnumVar(&c.outFormat, outFormatJSON)
	cmd.Flag("dry-run", "Will execute all the process without creating any drift detection plans, will use latest ones available.").BoolVar(&c.dryRun)

	return c
}

func (c RunCommand) Name() string { return c.cmd.FullCommand() }
func (c RunCommand) Run(ctx context.Context) error {
	logger := c.rootConfig.Logger

	if len(c.excludeNameRegexes) > 0 && len(c.includeNameRegexes) > 0 {
		return fmt.Errorf("include and exclude name options can't be used at the same time")
	}

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
		repo = tfestorage.NewDryRunRepository(logger, repo)
	}

	var includeProcessor process.Processor = process.NoopProcessor
	if len(c.includeNameRegexes) > 0 {
		p, err := wksprocess.NewIncludeNameProcessor(logger, c.includeNameRegexes)
		if err != nil {
			return fmt.Errorf("invalid include processor: %w", err)
		}
		includeProcessor = p
	}

	var excludeProcessor process.Processor = process.NoopProcessor
	if len(c.excludeNameRegexes) > 0 {
		p, err := wksprocess.NewExcludeNameProcessor(logger, c.excludeNameRegexes)
		if err != nil {
			return fmt.Errorf("invalid exclude processor: %w", err)
		}
		excludeProcessor = p
	}

	var resultOutProcessor process.Processor = process.NoopProcessor
	switch c.outFormat {
	case outFormatJSON:
		resultOutProcessor = wksprocess.NewDetailedJSONResultProcessor(c.rootConfig.Stdout)

	}

	wksProcessors := []wksprocess.Processor{
		includeProcessor,
		excludeProcessor,
		wksprocess.NewHydrateLatestDetectionPlanProcessor(ctx, logger, repo),
		wksprocess.NewFilterQueuedDriftDetectorProcessor(logger),
		wksprocess.NewFilterDriftDetectionsBeforeProcessor(logger, c.notBefore),
		wksprocess.NewSortByOldestDetectionPlanProcessor(logger),
		wksprocess.NewLimitMaxProcessor(logger, c.maxPlans),
		wksprocess.NewDriftDetectionPlanProcessor(logger, repo, c.planMessage),
		wksprocess.NewDriftDetectionPlanWaitProcessor(logger, repo, waitPolling, c.waitTimeout),
		resultOutProcessor,
		wksprocess.NewDriftDetectionPlansResultProcessor(logger, c.disableDriftPlanExitCodes),
	}

	// Execute.
	wks, err := repo.ListWorkspaces(ctx)
	if err != nil {
		return fmt.Errorf("could not list workspaces: %w", err)
	}

	chain := wksprocess.NewProcessorChain(wksProcessors)
	_, err = chain.Process(ctx, wks)
	if err != nil {
		return fmt.Errorf("workspaces processing failed: %w", err)
	}

	return nil
}
