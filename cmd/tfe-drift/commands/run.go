package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/hashicorp/go-tfe"

	tfestorage "github.com/slok/tfe-drift/internal/storage/tfe"
	"github.com/slok/tfe-drift/internal/workspace/process"
	wksprocess "github.com/slok/tfe-drift/internal/workspace/process"
)

var (
	waitPolling = 15 * time.Second

	outFormatJSON       = "json"
	outFormatPrettyJSON = "pretty-json"
)

type RunCommand struct {
	cmd        *kingpin.CmdClause
	rootConfig *RootCommand

	planMessage               string
	includeNameRegexes        []string
	excludeNameRegexes        []string
	includeTags               []string
	excludeTags               []string
	notBefore                 time.Duration
	maxPlans                  int
	waitTimeout               time.Duration
	disableDriftPlanExitCodes bool
	outFormat                 string
	dryRun                    bool
	fetchWorkers              int
}

// NewRunCommand returns the Run command.
func NewRunCommand(rootConfig *RootCommand, app *kingpin.Application) *RunCommand {
	cmd := app.Command("run", "Runs drift detections.")
	c := &RunCommand{
		cmd:        cmd,
		rootConfig: rootConfig,
	}

	cmd.Flag("plan-message", "Message to set on the executed drift detection plans.").Short('m').Default("Drift detection").StringVar(&c.planMessage)
	cmd.Flag("include-name", "Regex that if matches workspace name it will be included in the drift detection (can be repeated or comma separated).").Short('i').StringsVar(&c.includeNameRegexes)
	cmd.Flag("exclude-name", "Regex that if matches workspace name it will be excluded from the drift detection (can be repeated or comma separated).").Short('e').StringsVar(&c.excludeNameRegexes)
	cmd.Flag("include-tag", "The workspaces that match the tag will be included (can be repeated or comma separated).").Short('t').StringsVar(&c.includeTags)
	cmd.Flag("exclude-tag", "The workspaces that match the tag will be excluded (can be repeated or comma separated).").Short('x').StringsVar(&c.excludeTags)
	cmd.Flag("limit-max-plans", "The maximum drift detection plans that will be executed.").Short('l').IntVar(&c.maxPlans)
	cmd.Flag("not-before", "Will filter the workspaces that executed a drift detection plan before before this duration.").Short('n').Default("1h").DurationVar(&c.notBefore)
	cmd.Flag("wait-timeout", "Max time duration to wait for drift detection plans to finish.").Default("2h").DurationVar(&c.waitTimeout)
	cmd.Flag("disable-drift-plan-exitcodes", "Will disable the drift detection plans related exit codes (2 and 3).").BoolVar(&c.disableDriftPlanExitCodes)
	cmd.Flag("out-format", "Selects the format of the result output.").Short('o').EnumVar(&c.outFormat, outFormatJSON, outFormatPrettyJSON)
	cmd.Flag("dry-run", "Will execute all the process without creating any drift detection plans, will use latest ones available.").BoolVar(&c.dryRun)
	cmd.Flag("fetch-workers", "The number of workers running concurrently to fetch workspaces information.").Default("20").IntVar(&c.fetchWorkers)

	return c
}

func (c RunCommand) Name() string { return c.cmd.FullCommand() }
func (c RunCommand) Run(ctx context.Context) error {
	logger := c.rootConfig.Logger

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
		repo = tfestorage.NewDryRunRepository(logger, repo)
	}

	var includeProcessor process.Processor = process.NoopProcessor
	if len(includeNameRegexes) > 0 {
		p, err := wksprocess.NewIncludeNameProcessor(logger, includeNameRegexes)
		if err != nil {
			return fmt.Errorf("invalid include processor: %w", err)
		}
		includeProcessor = p
	}

	var excludeProcessor process.Processor = process.NoopProcessor
	if len(excludeNameRegexes) > 0 {
		p, err := wksprocess.NewExcludeNameProcessor(logger, excludeNameRegexes)
		if err != nil {
			return fmt.Errorf("invalid exclude processor: %w", err)
		}
		excludeProcessor = p
	}

	var resultOutProcessor process.Processor = process.NoopProcessor
	switch c.outFormat {
	case outFormatJSON:
		resultOutProcessor = wksprocess.NewDetailedJSONResultProcessor(c.rootConfig.Stdout, false)
	case outFormatPrettyJSON:
		resultOutProcessor = wksprocess.NewDetailedJSONResultProcessor(c.rootConfig.Stdout, true)
	}

	wksProcessors := []wksprocess.Processor{
		includeProcessor,
		excludeProcessor,
		wksprocess.NewHydrateLatestDetectionPlanProcessor(ctx, logger, repo, c.fetchWorkers),
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
	logger.Infof("Retrieving workspaces")
	wks, err := repo.ListWorkspaces(ctx, includeTags, excludeTags)
	if err != nil {
		return fmt.Errorf("could not list workspaces: %w", err)
	}

	if len(wks) == 0 {
		return fmt.Errorf("0 workspaces selected")
	}

	chain := wksprocess.NewProcessorChain(wksProcessors)
	_, err = chain.Process(ctx, wks)
	if err != nil {
		return fmt.Errorf("workspaces processing failed: %w", err)
	}

	return nil
}

// splitRepeatedArg will split the strings inside each repeated arg and return flatten.
func splitRepeatedArg(ss []string, c string) []string {
	newSS := []string{}
	for _, s := range ss {
		newSS = append(newSS, strings.Split(s, c)...)
	}

	return newSS
}
