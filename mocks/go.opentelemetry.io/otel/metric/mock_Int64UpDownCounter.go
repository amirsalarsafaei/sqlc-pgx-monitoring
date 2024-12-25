// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"
)

// MockInt64UpDownCounter is an autogenerated mock type for the Int64UpDownCounter type
type MockInt64UpDownCounter struct {
	mock.Mock
}

type MockInt64UpDownCounter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockInt64UpDownCounter) EXPECT() *MockInt64UpDownCounter_Expecter {
	return &MockInt64UpDownCounter_Expecter{mock: &_m.Mock}
}

// Add provides a mock function with given fields: ctx, incr, options
func (_m *MockInt64UpDownCounter) Add(ctx context.Context, incr int64, options ...metric.AddOption) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, incr)
	_ca = append(_ca, _va...)
	_m.Called(_ca...)
}

// MockInt64UpDownCounter_Add_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Add'
type MockInt64UpDownCounter_Add_Call struct {
	*mock.Call
}

// Add is a helper method to define mock.On call
//   - ctx context.Context
//   - incr int64
//   - options ...metric.AddOption
func (_e *MockInt64UpDownCounter_Expecter) Add(ctx interface{}, incr interface{}, options ...interface{}) *MockInt64UpDownCounter_Add_Call {
	return &MockInt64UpDownCounter_Add_Call{Call: _e.mock.On("Add",
		append([]interface{}{ctx, incr}, options...)...)}
}

func (_c *MockInt64UpDownCounter_Add_Call) Run(run func(ctx context.Context, incr int64, options ...metric.AddOption)) *MockInt64UpDownCounter_Add_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.AddOption, len(args)-2)
		for i, a := range args[2:] {
			if a != nil {
				variadicArgs[i] = a.(metric.AddOption)
			}
		}
		run(args[0].(context.Context), args[1].(int64), variadicArgs...)
	})
	return _c
}

func (_c *MockInt64UpDownCounter_Add_Call) Return() *MockInt64UpDownCounter_Add_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockInt64UpDownCounter_Add_Call) RunAndReturn(run func(context.Context, int64, ...metric.AddOption)) *MockInt64UpDownCounter_Add_Call {
	_c.Run(run)
	return _c
}

// int64UpDownCounter provides a mock function with no fields
func (_m *MockInt64UpDownCounter) int64UpDownCounter() {
	_m.Called()
}

// MockInt64UpDownCounter_int64UpDownCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'int64UpDownCounter'
type MockInt64UpDownCounter_int64UpDownCounter_Call struct {
	*mock.Call
}

// int64UpDownCounter is a helper method to define mock.On call
func (_e *MockInt64UpDownCounter_Expecter) int64UpDownCounter() *MockInt64UpDownCounter_int64UpDownCounter_Call {
	return &MockInt64UpDownCounter_int64UpDownCounter_Call{Call: _e.mock.On("int64UpDownCounter")}
}

func (_c *MockInt64UpDownCounter_int64UpDownCounter_Call) Run(run func()) *MockInt64UpDownCounter_int64UpDownCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockInt64UpDownCounter_int64UpDownCounter_Call) Return() *MockInt64UpDownCounter_int64UpDownCounter_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockInt64UpDownCounter_int64UpDownCounter_Call) RunAndReturn(run func()) *MockInt64UpDownCounter_int64UpDownCounter_Call {
	_c.Run(run)
	return _c
}

// NewMockInt64UpDownCounter creates a new instance of MockInt64UpDownCounter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockInt64UpDownCounter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockInt64UpDownCounter {
	mock := &MockInt64UpDownCounter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}