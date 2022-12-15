// Code generated by mockery v2.14.1. DO NOT EDIT.

package prometheusmock

import (
	context "context"

	model "github.com/slok/tfe-drift/internal/model"
	mock "github.com/stretchr/testify/mock"
)

// WorkspaceRepository is an autogenerated mock type for the WorkspaceRepository type
type WorkspaceRepository struct {
	mock.Mock
}

// ListWorkspaces provides a mock function with given fields: ctx, includeTags, excludeTags
func (_m *WorkspaceRepository) ListWorkspaces(ctx context.Context, includeTags []string, excludeTags []string) ([]model.Workspace, error) {
	ret := _m.Called(ctx, includeTags, excludeTags)

	var r0 []model.Workspace
	if rf, ok := ret.Get(0).(func(context.Context, []string, []string) []model.Workspace); ok {
		r0 = rf(ctx, includeTags, excludeTags)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Workspace)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, []string, []string) error); ok {
		r1 = rf(ctx, includeTags, excludeTags)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewWorkspaceRepository interface {
	mock.TestingT
	Cleanup(func())
}

// NewWorkspaceRepository creates a new instance of WorkspaceRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewWorkspaceRepository(t mockConstructorTestingTNewWorkspaceRepository) *WorkspaceRepository {
	mock := &WorkspaceRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
