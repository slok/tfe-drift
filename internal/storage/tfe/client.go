package tfe

import (
	"context"

	"github.com/hashicorp/go-tfe"
)

// Client is a helper interface to be able to manage in a simpler way the TFE official client.
type Client interface {
	ListWorkspaces(ctx context.Context, organization string, options *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error)
	CreateRun(ctx context.Context, options tfe.RunCreateOptions) (*tfe.Run, error)
	ReadRun(ctx context.Context, runID string) (*tfe.Run, error)
}

//go:generate mockery --case underscore --output tfemock --outpkg tfemock --name Client

func NewClient(c *tfe.Client) Client {
	return tfeClient{c: c}
}

type tfeClient struct {
	c *tfe.Client
}

func (t tfeClient) ListWorkspaces(ctx context.Context, organization string, options *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {

	return t.c.Workspaces.List(ctx, organization, options)
}

func (t tfeClient) CreateRun(ctx context.Context, options tfe.RunCreateOptions) (*tfe.Run, error) {
	return t.c.Runs.Create(ctx, options)
}

func (t tfeClient) ReadRun(ctx context.Context, runID string) (*tfe.Run, error) {
	return t.c.Runs.Read(ctx, runID)
}
