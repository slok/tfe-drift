package model

import (
	"time"

	"github.com/hashicorp/go-tfe"
)

type Workspace struct {
	Name          string
	ID            string
	LastDriftPlan *Plan

	// OriginalObject is the object from the original APIs (e.g go-tfe).
	OriginalObject *tfe.Workspace
}

// Plan is a run plan used for drift checks.
type Plan struct {
	ID         string
	CreatedAt  time.Time
	Message    string
	HasChanges bool
	Status     PlanStatus
	URL        string

	// OriginalObject is the object from the original APIs (e.g go-tfe).
	OriginalObject *tfe.Run
}

// PlanStatus are the simplified status that this app is interested when we
// talk about a TFE run plan used to drift checks.
type PlanStatus int

const (
	PlanStatusUnknown PlanStatus = iota
	PlanStatusWaiting
	PlanStatusFinishedOK
	PlanStatusFinishedNotOK
)
