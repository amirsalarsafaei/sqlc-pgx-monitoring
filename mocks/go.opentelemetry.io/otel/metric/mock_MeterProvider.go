// Code generated by mockery v2.50.1. DO NOT EDIT.

package metric

import (
	mock "github.com/stretchr/testify/mock"
	metric "go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel/metric/embedded"
)

// MockMeterProvider is an autogenerated mock type for the MeterProvider type
type MockMeterProvider struct {
	mock.Mock
	embedded.MeterProvider
}

type MockMeterProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *MockMeterProvider) EXPECT() *MockMeterProvider_Expecter {
	return &MockMeterProvider_Expecter{mock: &_m.Mock}
}

// Meter provides a mock function with given fields: name, opts
func (_m *MockMeterProvider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, name)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Meter")
	}

	var r0 metric.Meter
	if rf, ok := ret.Get(0).(func(string, ...metric.MeterOption) metric.Meter); ok {
		r0 = rf(name, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Meter)
		}
	}

	return r0
}

// MockMeterProvider_Meter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Meter'
type MockMeterProvider_Meter_Call struct {
	*mock.Call
}

// Meter is a helper method to define mock.On call
//   - name string
//   - opts ...metric.MeterOption
func (_e *MockMeterProvider_Expecter) Meter(name interface{}, opts ...interface{}) *MockMeterProvider_Meter_Call {
	return &MockMeterProvider_Meter_Call{Call: _e.mock.On("Meter",
		append([]interface{}{name}, opts...)...)}
}

func (_c *MockMeterProvider_Meter_Call) Run(run func(name string, opts ...metric.MeterOption)) *MockMeterProvider_Meter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]metric.MeterOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(metric.MeterOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *MockMeterProvider_Meter_Call) Return(_a0 metric.Meter) *MockMeterProvider_Meter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockMeterProvider_Meter_Call) RunAndReturn(run func(string, ...metric.MeterOption) metric.Meter) *MockMeterProvider_Meter_Call {
	_c.Call.Return(run)
	return _c
}

// meterProvider provides a mock function with no fields
func (_m *MockMeterProvider) meterProvider() {
	_m.Called()
}

// MockMeterProvider_meterProvider_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'meterProvider'
type MockMeterProvider_meterProvider_Call struct {
	*mock.Call
}

// meterProvider is a helper method to define mock.On call
func (_e *MockMeterProvider_Expecter) meterProvider() *MockMeterProvider_meterProvider_Call {
	return &MockMeterProvider_meterProvider_Call{Call: _e.mock.On("meterProvider")}
}

func (_c *MockMeterProvider_meterProvider_Call) Run(run func()) *MockMeterProvider_meterProvider_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockMeterProvider_meterProvider_Call) Return() *MockMeterProvider_meterProvider_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockMeterProvider_meterProvider_Call) RunAndReturn(run func()) *MockMeterProvider_meterProvider_Call {
	_c.Run(run)
	return _c
}

// NewMockMeterProvider creates a new instance of MockMeterProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockMeterProvider(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockMeterProvider {
	mock := &MockMeterProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}