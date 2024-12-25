// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"
)

// MockInt64Callback is an autogenerated mock type for the Int64Callback type
type MockInt64Callback struct {
	mock.Mock
}

type MockInt64Callback_Expecter struct {
	mock *mock.Mock
}

func (_m *MockInt64Callback) EXPECT() *MockInt64Callback_Expecter {
	return &MockInt64Callback_Expecter{mock: &_m.Mock}
}

// Execute provides a mock function with given fields: _a0, _a1
func (_m *MockInt64Callback) Execute(_a0 context.Context, _a1 metric.Int64Observer) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Execute")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, metric.Int64Observer) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockInt64Callback_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type MockInt64Callback_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 metric.Int64Observer
func (_e *MockInt64Callback_Expecter) Execute(_a0 interface{}, _a1 interface{}) *MockInt64Callback_Execute_Call {
	return &MockInt64Callback_Execute_Call{Call: _e.mock.On("Execute", _a0, _a1)}
}

func (_c *MockInt64Callback_Execute_Call) Run(run func(_a0 context.Context, _a1 metric.Int64Observer)) *MockInt64Callback_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metric.Int64Observer))
	})
	return _c
}

func (_c *MockInt64Callback_Execute_Call) Return(_a0 error) *MockInt64Callback_Execute_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockInt64Callback_Execute_Call) RunAndReturn(run func(context.Context, metric.Int64Observer) error) *MockInt64Callback_Execute_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockInt64Callback creates a new instance of MockInt64Callback. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockInt64Callback(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockInt64Callback {
	mock := &MockInt64Callback{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
