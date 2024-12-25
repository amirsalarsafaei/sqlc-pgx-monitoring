// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"
)

// MockFloat64UpDownCounterOption is an autogenerated mock type for the Float64UpDownCounterOption type
type MockFloat64UpDownCounterOption struct {
	mock.Mock
}

type MockFloat64UpDownCounterOption_Expecter struct {
	mock *mock.Mock
}

func (_m *MockFloat64UpDownCounterOption) EXPECT() *MockFloat64UpDownCounterOption_Expecter {
	return &MockFloat64UpDownCounterOption_Expecter{mock: &_m.Mock}
}

// applyFloat64UpDownCounter provides a mock function with given fields: _a0
func (_m *MockFloat64UpDownCounterOption) applyFloat64UpDownCounter(_a0 metric.Float64UpDownCounterConfig) metric.Float64UpDownCounterConfig {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for applyFloat64UpDownCounter")
	}

	var r0 metric.Float64UpDownCounterConfig
	if rf, ok := ret.Get(0).(func(metric.Float64UpDownCounterConfig) metric.Float64UpDownCounterConfig); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(metric.Float64UpDownCounterConfig)
	}

	return r0
}

// MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'applyFloat64UpDownCounter'
type MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call struct {
	*mock.Call
}

// applyFloat64UpDownCounter is a helper method to define mock.On call
//   - _a0 metric.Float64UpDownCounterConfig
func (_e *MockFloat64UpDownCounterOption_Expecter) applyFloat64UpDownCounter(_a0 interface{}) *MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call {
	return &MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call{Call: _e.mock.On("applyFloat64UpDownCounter", _a0)}
}

func (_c *MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call) Run(run func(_a0 metric.Float64UpDownCounterConfig)) *MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(metric.Float64UpDownCounterConfig))
	})
	return _c
}

func (_c *MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call) Return(_a0 metric.Float64UpDownCounterConfig) *MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call) RunAndReturn(run func(metric.Float64UpDownCounterConfig) metric.Float64UpDownCounterConfig) *MockFloat64UpDownCounterOption_applyFloat64UpDownCounter_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockFloat64UpDownCounterOption creates a new instance of MockFloat64UpDownCounterOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockFloat64UpDownCounterOption(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockFloat64UpDownCounterOption {
	mock := &MockFloat64UpDownCounterOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}