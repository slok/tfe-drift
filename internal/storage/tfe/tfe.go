package tfe

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe"

	"github.com/slok/tfe-drift/internal/model"
)

// Repository knows how to manage data on Terraform enterprise or cloud.
type Repository interface {
	ListWorkspaces(ctx context.Context) ([]model.Workspace, error)
	CreateCheckPlan(ctx context.Context, w model.Workspace, message string) (*model.Plan, error)
	GetCheckPlan(ctx context.Context, id string) (*model.Plan, error)
}

func NewRepository(c Client, org string) (Repository, error) {
	return repository{
		c:   c,
		org: org,
	}, nil
}

type repository struct {
	c   Client
	org string
}

func (r repository) ListWorkspaces(ctx context.Context) ([]model.Workspace, error) {
	allWks := []*tfe.Workspace{}

	// Get all workspaces using client pagination.
	page := 0
	for {
		wks, err := r.c.ListWorkspaces(ctx, r.org, &tfe.WorkspaceListOptions{ListOptions: tfe.ListOptions{PageNumber: page}})
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
		mwk, err := mapWorkspaceTFE2Model(wk)
		if err != nil {
			return nil, fmt.Errorf("could not map tfe workspaces to model: %w", err)
		}
		wks = append(wks, *mwk)
	}

	return wks, nil
}

func (r repository) CreateCheckPlan(ctx context.Context, wk model.Workspace, message string) (*model.Plan, error) {
	run, err := r.c.CreateRun(ctx, tfe.RunCreateOptions{
		PlanOnly:             tfe.Bool(true),
		Message:              tfe.String(message),
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

	return plan, nil
}

func (r repository) GetCheckPlan(ctx context.Context, id string) (*model.Plan, error) {
	run, err := r.c.ReadRun(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get check plan from tfe: %w", err)
	}

	// Map to model.
	plan, err := mapPlanTFE2Model(run)
	if err != nil {
		return nil, fmt.Errorf("could not map tfe run to model: %w", err)
	}

	return plan, nil
}

func mapWorkspaceTFE2Model(w *tfe.Workspace) (*model.Workspace, error) {
	return &model.Workspace{
		Name:           w.Name,
		ID:             w.ID,
		OriginalObject: w,
	}, nil
}

func mapPlanTFE2Model(r *tfe.Run) (*model.Plan, error) {
	return &model.Plan{
		ID:             r.ID,
		Message:        r.Message,
		CreatedAt:      r.CreatedAt,
		HasChanges:     r.HasChanges,
		Status:         mapTFEStatus2Model(r.Status),
		OriginalObject: r,
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
