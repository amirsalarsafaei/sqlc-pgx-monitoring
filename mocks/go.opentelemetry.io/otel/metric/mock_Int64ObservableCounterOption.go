// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"
)

// MockInt64ObservableCounterOption is an autogenerated mock type for the Int64ObservableCounterOption type
type MockInt64ObservableCounterOption struct {
	mock.Mock
}

type MockInt64ObservableCounterOption_Expecter struct {
	mock *mock.Mock
}

func (_m *MockInt64ObservableCounterOption) EXPECT() *MockInt64ObservableCounterOption_Expecter {
	return &MockInt64ObservableCounterOption_Expecter{mock: &_m.Mock}
}

// applyInt64ObservableCounter provides a mock function with given fields: _a0
func (_m *MockInt64ObservableCounterOption) applyInt64ObservableCounter(_a0 metric.Int64ObservableCounterConfig) metric.Int64ObservableCounterConfig {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for applyInt64ObservableCounter")
	}

	var r0 metric.Int64ObservableCounterConfig
	if rf, ok := ret.Get(0).(func(metric.Int64ObservableCounterConfig) metric.Int64ObservableCounterConfig); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(metric.Int64ObservableCounterConfig)
	}

	return r0
}

// MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'applyInt64ObservableCounter'
type MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call struct {
	*mock.Call
}

// applyInt64ObservableCounter is a helper method to define mock.On call
//   - _a0 metric.Int64ObservableCounterConfig
func (_e *MockInt64ObservableCounterOption_Expecter) applyInt64ObservableCounter(_a0 interface{}) *MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call {
	return &MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call{Call: _e.mock.On("applyInt64ObservableCounter", _a0)}
}

func (_c *MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call) Run(run func(_a0 metric.Int64ObservableCounterConfig)) *MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(metric.Int64ObservableCounterConfig))
	})
	return _c
}

func (_c *MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call) Return(_a0 metric.Int64ObservableCounterConfig) *MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call) RunAndReturn(run func(metric.Int64ObservableCounterConfig) metric.Int64ObservableCounterConfig) *MockInt64ObservableCounterOption_applyInt64ObservableCounter_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockInt64ObservableCounterOption creates a new instance of MockInt64ObservableCounterOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockInt64ObservableCounterOption(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockInt64ObservableCounterOption {
	mock := &MockInt64ObservableCounterOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}