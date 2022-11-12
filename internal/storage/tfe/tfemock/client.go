// Code generated by mockery v2.14.1. DO NOT EDIT.

package tfemock

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	tfe "github.com/hashicorp/go-tfe"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// CreateRun provides a mock function with given fields: ctx, options
func (_m *Client) CreateRun(ctx context.Context, options tfe.RunCreateOptions) (*tfe.Run, error) {
	ret := _m.Called(ctx, options)

	var r0 *tfe.Run
	if rf, ok := ret.Get(0).(func(context.Context, tfe.RunCreateOptions) *tfe.Run); ok {
		r0 = rf(ctx, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tfe.Run)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, tfe.RunCreateOptions) error); ok {
		r1 = rf(ctx, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListWorkspaces provides a mock function with given fields: ctx, organization, options
func (_m *Client) ListWorkspaces(ctx context.Context, organization string, options *tfe.WorkspaceListOptions) (*tfe.WorkspaceList, error) {
	ret := _m.Called(ctx, organization, options)

	var r0 *tfe.WorkspaceList
	if rf, ok := ret.Get(0).(func(context.Context, string, *tfe.WorkspaceListOptions) *tfe.WorkspaceList); ok {
		r0 = rf(ctx, organization, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tfe.WorkspaceList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, *tfe.WorkspaceListOptions) error); ok {
		r1 = rf(ctx, organization, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadRun provides a mock function with given fields: ctx, runID
func (_m *Client) ReadRun(ctx context.Context, runID string) (*tfe.Run, error) {
	ret := _m.Called(ctx, runID)

	var r0 *tfe.Run
	if rf, ok := ret.Get(0).(func(context.Context, string) *tfe.Run); ok {
		r0 = rf(ctx, runID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*tfe.Run)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, runID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewClient(t mockConstructorTestingTNewClient) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}