// Code generated by mockery. DO NOT EDIT.

package exampletable

import (
	context "context"

	pgtype "github.com/jackc/pgx/v5/pgtype"
	mock "github.com/stretchr/testify/mock"
)

// MockQuerier is an autogenerated mock type for the Querier type
type MockQuerier struct {
	mock.Mock
}

type MockQuerier_Expecter struct {
	mock *mock.Mock
}

func (_m *MockQuerier) EXPECT() *MockQuerier_Expecter {
	return &MockQuerier_Expecter{mock: &_m.Mock}
}

// ExampleQuery provides a mock function with given fields: ctx, db, foo
func (_m *MockQuerier) ExampleQuery(ctx context.Context, db DBTX, foo pgtype.Text) ([]ExampleTable, error) {
	ret := _m.Called(ctx, db, foo)

	if len(ret) == 0 {
		panic("no return value specified for ExampleQuery")
	}

	var r0 []ExampleTable
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, DBTX, pgtype.Text) ([]ExampleTable, error)); ok {
		return rf(ctx, db, foo)
	}
	if rf, ok := ret.Get(0).(func(context.Context, DBTX, pgtype.Text) []ExampleTable); ok {
		r0 = rf(ctx, db, foo)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ExampleTable)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, DBTX, pgtype.Text) error); ok {
		r1 = rf(ctx, db, foo)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQuerier_ExampleQuery_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ExampleQuery'
type MockQuerier_ExampleQuery_Call struct {
	*mock.Call
}

// ExampleQuery is a helper method to define mock.On call
//   - ctx context.Context
//   - db DBTX
//   - foo pgtype.Text
func (_e *MockQuerier_Expecter) ExampleQuery(ctx interface{}, db interface{}, foo interface{}) *MockQuerier_ExampleQuery_Call {
	return &MockQuerier_ExampleQuery_Call{Call: _e.mock.On("ExampleQuery", ctx, db, foo)}
}

func (_c *MockQuerier_ExampleQuery_Call) Run(run func(ctx context.Context, db DBTX, foo pgtype.Text)) *MockQuerier_ExampleQuery_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(DBTX), args[2].(pgtype.Text))
	})
	return _c
}

func (_c *MockQuerier_ExampleQuery_Call) Return(_a0 []ExampleTable, _a1 error) *MockQuerier_ExampleQuery_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQuerier_ExampleQuery_Call) RunAndReturn(run func(context.Context, DBTX, pgtype.Text) ([]ExampleTable, error)) *MockQuerier_ExampleQuery_Call {
	_c.Call.Return(run)
	return _c
}

// ExampleQuery2 provides a mock function with given fields: ctx, db, foo
func (_m *MockQuerier) ExampleQuery2(ctx context.Context, db DBTX, foo pgtype.Text) ([]ExampleTable, error) {
	ret := _m.Called(ctx, db, foo)

	if len(ret) == 0 {
		panic("no return value specified for ExampleQuery2")
	}

	var r0 []ExampleTable
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, DBTX, pgtype.Text) ([]ExampleTable, error)); ok {
		return rf(ctx, db, foo)
	}
	if rf, ok := ret.Get(0).(func(context.Context, DBTX, pgtype.Text) []ExampleTable); ok {
		r0 = rf(ctx, db, foo)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ExampleTable)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, DBTX, pgtype.Text) error); ok {
		r1 = rf(ctx, db, foo)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockQuerier_ExampleQuery2_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ExampleQuery2'
type MockQuerier_ExampleQuery2_Call struct {
	*mock.Call
}

// ExampleQuery2 is a helper method to define mock.On call
//   - ctx context.Context
//   - db DBTX
//   - foo pgtype.Text
func (_e *MockQuerier_Expecter) ExampleQuery2(ctx interface{}, db interface{}, foo interface{}) *MockQuerier_ExampleQuery2_Call {
	return &MockQuerier_ExampleQuery2_Call{Call: _e.mock.On("ExampleQuery2", ctx, db, foo)}
}

func (_c *MockQuerier_ExampleQuery2_Call) Run(run func(ctx context.Context, db DBTX, foo pgtype.Text)) *MockQuerier_ExampleQuery2_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(DBTX), args[2].(pgtype.Text))
	})
	return _c
}

func (_c *MockQuerier_ExampleQuery2_Call) Return(_a0 []ExampleTable, _a1 error) *MockQuerier_ExampleQuery2_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockQuerier_ExampleQuery2_Call) RunAndReturn(run func(context.Context, DBTX, pgtype.Text) ([]ExampleTable, error)) *MockQuerier_ExampleQuery2_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockQuerier creates a new instance of MockQuerier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockQuerier(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockQuerier {
	mock := &MockQuerier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
