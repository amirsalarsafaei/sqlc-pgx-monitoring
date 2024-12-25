// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"
)

// MockInt64UpDownCounterOption is an autogenerated mock type for the Int64UpDownCounterOption type
type MockInt64UpDownCounterOption struct {
	mock.Mock
}

type MockInt64UpDownCounterOption_Expecter struct {
	mock *mock.Mock
}

func (_m *MockInt64UpDownCounterOption) EXPECT() *MockInt64UpDownCounterOption_Expecter {
	return &MockInt64UpDownCounterOption_Expecter{mock: &_m.Mock}
}

// applyInt64UpDownCounter provides a mock function with given fields: _a0
func (_m *MockInt64UpDownCounterOption) applyInt64UpDownCounter(_a0 metric.Int64UpDownCounterConfig) metric.Int64UpDownCounterConfig {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for applyInt64UpDownCounter")
	}

	var r0 metric.Int64UpDownCounterConfig
	if rf, ok := ret.Get(0).(func(metric.Int64UpDownCounterConfig) metric.Int64UpDownCounterConfig); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(metric.Int64UpDownCounterConfig)
	}

	return r0
}

// MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'applyInt64UpDownCounter'
type MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call struct {
	*mock.Call
}

// applyInt64UpDownCounter is a helper method to define mock.On call
//   - _a0 metric.Int64UpDownCounterConfig
func (_e *MockInt64UpDownCounterOption_Expecter) applyInt64UpDownCounter(_a0 interface{}) *MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call {
	return &MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call{Call: _e.mock.On("applyInt64UpDownCounter", _a0)}
}

func (_c *MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call) Run(run func(_a0 metric.Int64UpDownCounterConfig)) *MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(metric.Int64UpDownCounterConfig))
	})
	return _c
}

func (_c *MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call) Return(_a0 metric.Int64UpDownCounterConfig) *MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call) RunAndReturn(run func(metric.Int64UpDownCounterConfig) metric.Int64UpDownCounterConfig) *MockInt64UpDownCounterOption_applyInt64UpDownCounter_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockInt64UpDownCounterOption creates a new instance of MockInt64UpDownCounterOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockInt64UpDownCounterOption(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockInt64UpDownCounterOption {
	mock := &MockInt64UpDownCounterOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}