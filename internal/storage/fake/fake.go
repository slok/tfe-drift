package fake

import (
	"context"
	"fmt"
	"time"

	"github.com/slok/tfe-drift/internal/model"
	"github.com/slok/tfe-drift/internal/storage/tfe"
)

func NewRepository() tfe.Repository {
	return repository{}
}

type repository struct{}

func (r repository) ListWorkspaces(ctx context.Context, includeTags, excludeTags []string) ([]model.Workspace, error) {
	res := []model.Workspace{}
	for i := 0; i < 10; i++ {
		res = append(res, model.Workspace{
			ID:   fmt.Sprintf("id-wk-%d", i),
			Name: fmt.Sprintf("workspace-%d", i),
			Org:  "fake",
		})
	}
	return res, nil
}

func (r repository) CreateCheckPlan(ctx context.Context, w model.Workspace, message string) (*model.Plan, error) {
	t1 := time.Now()
	t0 := t1.Add(-25 * time.Second)
	return &model.Plan{
		ID:              fmt.Sprintf("plan-%s-%v", w.Name, t0),
		Message:         "This is a fake plan",
		CreatedAt:       t0,
		FinishedAt:      t1,
		PlanRunDuration: t1.Sub(t0),
		HasChanges:      t0.Second()/2 == 0,
		Status:          model.PlanStatusFinishedOK,
		OriginalObject:  nil,
	}, nil
}

func (r repository) GetCheckPlan(ctx context.Context, w model.Workspace, id string) (*model.Plan, error) {
	t1 := time.Now()
	t0 := t1.Add(-25 * time.Second)

	return &model.Plan{
		ID:              fmt.Sprintf("plan-%s-%v", w.Name, t0),
		Message:         "This is a fake plan",
		CreatedAt:       t0,
		FinishedAt:      t1,
		PlanRunDuration: t1.Sub(t0),
		HasChanges:      t0.Second()/2 == 0,
		Status:          model.PlanStatusFinishedOK,
		OriginalObject:  nil,
	}, nil
}

func (r repository) GetLatestCheckPlan(ctx context.Context, w model.Workspace) (*model.Plan, error) {
	t1 := time.Now()
	t0 := t1.Add(-25 * time.Second)

	return &model.Plan{
		ID:              fmt.Sprintf("plan-%s-%v", w.Name, t0),
		Message:         "This is a fake plan",
		CreatedAt:       t0,
		FinishedAt:      t1,
		PlanRunDuration: t1.Sub(t0),
		HasChanges:      t0.Second()/2 == 0,
		Status:          model.PlanStatusFinishedOK,
		OriginalObject:  nil,
	}, nil
}
