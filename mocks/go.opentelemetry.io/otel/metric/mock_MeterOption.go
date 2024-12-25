// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"
)

// MockMeterOption is an autogenerated mock type for the MeterOption type
type MockMeterOption struct {
	mock.Mock
}

type MockMeterOption_Expecter struct {
	mock *mock.Mock
}

func (_m *MockMeterOption) EXPECT() *MockMeterOption_Expecter {
	return &MockMeterOption_Expecter{mock: &_m.Mock}
}

// applyMeter provides a mock function with given fields: _a0
func (_m *MockMeterOption) applyMeter(_a0 metric.MeterConfig) metric.MeterConfig {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for applyMeter")
	}

	var r0 metric.MeterConfig
	if rf, ok := ret.Get(0).(func(metric.MeterConfig) metric.MeterConfig); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(metric.MeterConfig)
	}

	return r0
}

// MockMeterOption_applyMeter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'applyMeter'
type MockMeterOption_applyMeter_Call struct {
	*mock.Call
}

// applyMeter is a helper method to define mock.On call
//   - _a0 metric.MeterConfig
func (_e *MockMeterOption_Expecter) applyMeter(_a0 interface{}) *MockMeterOption_applyMeter_Call {
	return &MockMeterOption_applyMeter_Call{Call: _e.mock.On("applyMeter", _a0)}
}

func (_c *MockMeterOption_applyMeter_Call) Run(run func(_a0 metric.MeterConfig)) *MockMeterOption_applyMeter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(metric.MeterConfig))
	})
	return _c
}

func (_c *MockMeterOption_applyMeter_Call) Return(_a0 metric.MeterConfig) *MockMeterOption_applyMeter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockMeterOption_applyMeter_Call) RunAndReturn(run func(metric.MeterConfig) metric.MeterConfig) *MockMeterOption_applyMeter_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockMeterOption creates a new instance of MockMeterOption. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockMeterOption(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockMeterOption {
	mock := &MockMeterOption{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}