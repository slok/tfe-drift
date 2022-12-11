package tfe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-tfe"

	"github.com/slok/tfe-drift/internal/internalerrors"
	"github.com/slok/tfe-drift/internal/log"
	"github.com/slok/tfe-drift/internal/model"
)

const (
	messageIDFmt    = "tfe-drift/detector-id/%s"
	defaultPageSize = 100
)

// Repository knows how to manage data on Terraform enterprise or cloud.
type Repository interface {
	ListWorkspaces(ctx context.Context, includeTags, excludeTags []string) ([]model.Workspace, error)
	CreateCheckPlan(ctx context.Context, w model.Workspace, message string) (*model.Plan, error)
	GetCheckPlan(ctx context.Context, w model.Workspace, id string) (*model.Plan, error)
	GetLatestCheckPlan(ctx context.Context, w model.Workspace) (*model.Plan, error)
}

func NewRepository(c Client, tfeOrg, tfeAddress, detectorID string) (Repository, error) {
	return repository{
		c:          c,
		org:        tfeOrg,
		tfeAddress: tfeAddress,
		detectorID: detectorID,
	}, nil
}

type repository struct {
	c          Client
	org        string
	tfeAddress string
	detectorID string
}

func (r repository) ListWorkspaces(ctx context.Context, includeTags, excludeTags []string) ([]model.Workspace, error) {
	includeTagsFilter := strings.Join(includeTags, ",")
	excludeTagsFilter := strings.Join(excludeTags, ",")

	// Get all workspaces using client pagination.
	page := 0
	opts := &tfe.WorkspaceListOptions{
		Tags:        includeTagsFilter,
		ExcludeTags: excludeTagsFilter,
		ListOptions: tfe.ListOptions{PageSize: defaultPageSize, PageNumber: page},
	}
	allWks := []*tfe.Workspace{}
	for {
		opts.PageNumber = page
		wks, err := r.c.ListWorkspaces(ctx, r.org, opts)
		if err != nil {
			return nil, fmt.Errorf("could not get all workspaces: %w", err)
		}

		allWks = append(allWks, wks.Items...)

		// Nothing more to get.
		if wks.Pagination == nil || wks.NextPage == 0 || wks.NextPage == page {
			break
		}
		page = wks.NextPage
	}

	// Map to model.
	wks := make([]model.Workspace, 0, len(allWks))
	for _, wk := range allWks {
		mwk, err := r.mapWorkspaceTFE2Model(wk)
		if err != nil {
			return nil, fmt.Errorf("could not map tfe workspaces to model: %w", err)
		}
		wks = append(wks, *mwk)
	}

	return wks, nil
}

func (r repository) CreateCheckPlan(ctx context.Context, wk model.Workspace, message string) (*model.Plan, error) {
	messageID := fmt.Sprintf(messageIDFmt, r.detectorID)
	finalMessage := fmt.Sprintf("%s: %s", message, messageID)

	run, err := r.c.CreateRun(ctx, tfe.RunCreateOptions{
		PlanOnly:             tfe.Bool(true),
		Message:              tfe.String(finalMessage),
		Workspace:            wk.OriginalObject,
		ConfigurationVersion: nil, // This will make the plan run with the latest revision configured (normally main/master branch).
	})
	if err != nil {
		return nil, fmt.Errorf("could not create a check plan in tfe: %w", err)
	}

	// Map to model.
	plan, err := mapPlanTFE2Model(run)
	if err != nil {
		return nil, fmt.Errorf("could not map tfe run to model: %w", err)
	}

	// Get URL.
	plan.URL = r.runURL(wk.Name, run.ID)

	return plan, nil
}

func (r repository) GetCheckPlan(ctx context.Context, w model.Workspace, id string) (*model.Plan, error) {
	run, err := r.c.ReadRun(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get check plan from tfe: %w", err)
	}

	// Map to model.
	plan, err := mapPlanTFE2Model(run)
	if err != nil {
		return nil, fmt.Errorf("could not map tfe run to model: %w", err)
	}

	// Get URL.
	plan.URL = r.runURL(w.Name, run.ID)

	return plan, nil
}

func (r repository) GetLatestCheckPlan(ctx context.Context, w model.Workspace) (*model.Plan, error) {
	messageID := fmt.Sprintf(messageIDFmt, r.detectorID)
	runs, err := r.c.ListRuns(ctx, w.ID, &tfe.RunListOptions{
		Search:      messageID,
		ListOptions: tfe.ListOptions{PageSize: 1},
	})
	if err != nil {
		return nil, fmt.Errorf("could not get check plan from tfe: %w", err)
	}

	if len(runs.Items) == 0 {
		return nil, fmt.Errorf("check plans missing: %w", internalerrors.ErrNotExist)
	}

	// Map to model.
	run := runs.Items[0]
	plan, err := mapPlanTFE2Model(run)
	if err != nil {
		return nil, fmt.Errorf("could not map tfe run to model: %w", err)
	}

	// Get URL.
	plan.URL = r.runURL(w.Name, run.ID)

	return plan, nil
}

func (r repository) runURL(workspaceName, runID string) string {
	const runURLFmt = "%s/app/%s/workspaces/%s/runs/%s"

	return fmt.Sprintf(runURLFmt, r.tfeAddress, r.org, workspaceName, runID)
}

func (r repository) mapWorkspaceTFE2Model(w *tfe.Workspace) (*model.Workspace, error) {
	return &model.Workspace{
		Name:           w.Name,
		ID:             w.ID,
		Org:            r.org,
		OriginalObject: w,
		Tags:           w.TagNames,
	}, nil
}

func mapPlanTFE2Model(run *tfe.Run) (*model.Plan, error) {
	status := mapTFEStatus2Model(run.Status)

	var duration time.Duration
	var finishedAt time.Time
	if status != model.PlanStatusWaiting && run.StatusTimestamps != nil {
		finishedAt = run.StatusTimestamps.PlannedAndFinishedAt
		duration = run.StatusTimestamps.PlannedAndFinishedAt.Sub(run.StatusTimestamps.PlanningAt)
	}

	return &model.Plan{
		ID:              run.ID,
		Message:         run.Message,
		CreatedAt:       run.CreatedAt,
		FinishedAt:      finishedAt,
		PlanRunDuration: duration,
		HasChanges:      run.HasChanges,
		Status:          status,
		OriginalObject:  run,
	}, nil
}

func mapTFEStatus2Model(s tfe.RunStatus) model.PlanStatus {
	switch s {
	case tfe.RunPlannedAndFinished:
		return model.PlanStatusFinishedOK
	case tfe.RunCanceled, tfe.RunDiscarded, tfe.RunErrored:
		return model.PlanStatusFinishedNotOK
	case tfe.RunFetching, tfe.RunFetchingCompleted, tfe.RunPending, tfe.RunPlanned, tfe.RunPlanning,
		tfe.RunPlanQueued, tfe.RunPrePlanCompleted, tfe.RunPrePlanRunning, tfe.RunQueuing:
		return model.PlanStatusWaiting
	default:
		return model.PlanStatusUnknown
	}
}

func NewDryRunRepository(logger log.Logger, repo Repository) Repository {
	return dryRunRepository{
		Repository: repo,
		logger:     logger,
	}
}

type dryRunRepository struct {
	Repository
	logger log.Logger
}

func (r dryRunRepository) CreateCheckPlan(ctx context.Context, wk model.Workspace, message string) (*model.Plan, error) {
	r.logger.Warningf("Not creating drift detection plan due to dry-run. Using latest drift detection plan instead")
	return r.GetLatestCheckPlan(ctx, wk)
}
